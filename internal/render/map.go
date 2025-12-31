package render

import (
	"ascii1090/internal/adsb"
	"ascii1090/internal/debug"
	"ascii1090/internal/geo"

	"github.com/gdamore/tcell/v2"
)

// MapRenderer renders geographic features and aircraft to a canvas
type MapRenderer struct {
	projection *geo.Projection
	features   map[geo.FeatureType][]*geo.Feature
	canvas     *Canvas
}

// NewMapRenderer creates a new map renderer
func NewMapRenderer(projection *geo.Projection, features map[geo.FeatureType][]*geo.Feature, canvas *Canvas) *MapRenderer {
	return &MapRenderer{
		projection: projection,
		features:   features,
		canvas:     canvas,
	}
}

// RenderMap draws all geographic features to the canvas
func (m *MapRenderer) RenderMap() {
	// Get visible bounds
	bounds := m.projection.GetBounds()

	// Render in order: coastlines, rivers, borders, highways, cities, airports
	// This ensures proper layering (airports on top for visibility)
	m.renderFeatureType(geo.FeatureCoastline, bounds)
	m.renderFeatureType(geo.FeatureRiver, bounds)
	m.renderFeatureType(geo.FeatureStateBorder, bounds)
	m.renderFeatureType(geo.FeatureHighway, bounds)

	// Render cities and airports together to avoid overlapping labels
	m.renderCitiesAndAirports(bounds)
}

// renderFeatureType renders all features of a specific type
func (m *MapRenderer) renderFeatureType(ftype geo.FeatureType, bounds *geo.Bounds) {
	features, exists := m.features[ftype]
	if !exists {
		return
	}

	// Filter to only visible features
	visibleFeatures := geo.FilterByBounds(features, bounds)

	// Debug log cities
	if ftype == geo.FeatureCity && debug.Enabled() {
		debug.Log("Rendering %d cities (of %d total)", len(visibleFeatures), len(features))
		for _, city := range visibleFeatures {
			if city.Name != "" {
				debug.Log("  City: %s at %.2f, %.2f", city.Name, city.Point.Lat, city.Point.Lon)
			}
		}
	}

	// Debug log airports
	if ftype == geo.FeatureAirport && debug.Enabled() {
		debug.Log("Rendering %d airports (of %d total)", len(visibleFeatures), len(features))
		for i, airport := range visibleFeatures {
			if airport.Name != "" && i < 20 { // Limit to first 20 to avoid spam
				debug.Log("  Airport: %s at %.2f, %.2f", airport.Name, airport.Point.Lat, airport.Point.Lon)
			}
		}
	}

	for _, feature := range visibleFeatures {
		m.RenderFeature(feature)
	}
}

// RenderFeature draws a single geographic feature
func (m *MapRenderer) RenderFeature(feature *geo.Feature) {
	style := GetStyleForFeature(feature.Type)
	char := GetCharForFeature(feature.Type)

	if feature.IsPoint() {
		// Render point feature (city, airport)
		point := m.projection.Project(feature.Point.Lat, feature.Point.Lon)
		m.canvas.Set(point.X, point.Y, '●', style)

		// Render label if available and not too close to edge
		if feature.Name != "" && point.X < m.canvas.Width()-len(feature.Name)-1 {
			m.canvas.DrawText(point.X+1, point.Y, feature.Name, StyleLabel)
		}
	} else if feature.IsLine() {
		// Render line feature (border, river, road, coastline)
		for i := 0; i < len(feature.Points)-1; i++ {
			p1 := m.projection.Project(feature.Points[i].Lat, feature.Points[i].Lon)
			p2 := m.projection.Project(feature.Points[i+1].Lat, feature.Points[i+1].Lon)
			m.DrawLine(p1.X, p1.Y, p2.X, p2.Y, char, style)
		}
	}
}

// renderCitiesAndAirports renders cities and airports, avoiding overlapping labels
func (m *MapRenderer) renderCitiesAndAirports(bounds *geo.Bounds) {
	// Get airports and cities
	airports, hasAirports := m.features[geo.FeatureAirport]
	cities, hasCities := m.features[geo.FeatureCity]

	// Filter to visible bounds
	visibleAirports := []*geo.Feature{}
	if hasAirports {
		visibleAirports = geo.FilterByBounds(airports, bounds)
	}

	visibleCities := []*geo.Feature{}
	if hasCities {
		visibleCities = geo.FilterByBounds(cities, bounds)
	}

	// Project airport positions to screen coordinates for overlap detection
	type ScreenPoint struct {
		X, Y int
	}
	airportPositions := make([]ScreenPoint, 0, len(visibleAirports))
	for _, airport := range visibleAirports {
		if airport.Point != nil {
			point := m.projection.Project(airport.Point.Lat, airport.Point.Lon)
			airportPositions = append(airportPositions, ScreenPoint{X: point.X, Y: point.Y})
		}
	}

	// Render cities - Skip city labels that overlap with airports
	for _, city := range visibleCities {
		if city.Point == nil || city.Name == "" {
			continue
		}

		point := m.projection.Project(city.Point.Lat, city.Point.Lon)

		// Skip if this city is too close to any airport 
		skipCity := false
		for _, airportPos := range airportPositions {
			if airportPos.Y == point.Y && abs(airportPos.X-point.X) <= 5 {
				skipCity = true
				break
			}
			// Also skip if directly above/below and very close horizontally
			if abs(airportPos.Y-point.Y) <= 1 && abs(airportPos.X-point.X) <= 3 {
				skipCity = true
				break
			}
		}

		if skipCity {
			continue
		}

		if point.X < m.canvas.Width()-len(city.Name)-1 {
			m.canvas.DrawText(point.X, point.Y, city.Name, StyleLabel)
		}
	}

	// Render airports with @ symbol
	for _, airport := range visibleAirports {
		if airport.Point == nil {
			continue
		}

		point := m.projection.Project(airport.Point.Lat, airport.Point.Lon)
		m.canvas.Set(point.X, point.Y, '@', StyleAirport)

		// Render label if available and not too close to edge
		if airport.Name != "" && point.X < m.canvas.Width()-len(airport.Name)-1 {
			m.canvas.DrawText(point.X+1, point.Y, airport.Name, StyleLabel)
		}
	}
}

// RenderAircraft draws aircraft symbols on the canvas
func (m *MapRenderer) RenderAircraft(aircraft []*adsb.Aircraft, selectedICAO string) {
	for _, ac := range aircraft {
		if !ac.PositionLocked() {
			continue
		}

		point := m.projection.Project(*ac.Latitude, *ac.Longitude)
		symbol := ac.CardinalDirection()

		// Use different style for selected aircraft
		style := StyleAircraft
		if ac.ICAO == selectedICAO {
			style = StyleSelected
		}

		m.canvas.Set(point.X, point.Y, symbol, style)
	}
}

// DrawLine implements Bresenham's line algorithm for drawing lines on the canvas
func (m *MapRenderer) DrawLine(x0, y0, x1, y1 int, char rune, style tcell.Style) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)

	sx := -1
	if x0 < x1 {
		sx = 1
	}

	sy := -1
	if y0 < y1 {
		sy = 1
	}

	err := dx - dy

	for {
		m.canvas.Set(x0, y0, char, style)

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err

		if e2 > -dy {
			err -= dy
			x0 += sx
		}

		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// UpdateProjection updates the renderer's projection
func (m *MapRenderer) UpdateProjection(projection *geo.Projection) {
	m.projection = projection
}

// UpdateCanvas updates the renderer's canvas
func (m *MapRenderer) UpdateCanvas(canvas *Canvas) {
	m.canvas = canvas
}
