package ui

import (
	"ascii1090/internal/adsb"
	"ascii1090/internal/debug"
	"ascii1090/internal/geo"
	"ascii1090/internal/render"

	"github.com/gdamore/tcell/v2"
)

// MapView displays the map and aircraft
type MapView struct {
	renderer    *render.MapRenderer
	projection  *geo.Projection
	canvas      *render.Canvas
	centerSet   bool
	width       int
	height      int
	radiusMiles float64
	aspectRatio float64
}

// NewMapView creates a new map view
func NewMapView(width, height int, features map[geo.FeatureType][]*geo.Feature, radiusMiles float64, aspectRatio float64) *MapView {
	centerLat := 39.8283
	centerLon := -98.5795

	projection := geo.NewProjection(centerLat, centerLon, radiusMiles, width, height, aspectRatio)
	canvas := render.NewCanvas(width, height)
	renderer := render.NewMapRenderer(projection, features, canvas)

	return &MapView{
		renderer:    renderer,
		projection:  projection,
		canvas:      canvas,
		centerSet:   false,
		width:       width,
		height:      height,
		radiusMiles: radiusMiles,
		aspectRatio: aspectRatio,
	}
}

// Draw renders the map view to the screen
func (m *MapView) Draw(screen tcell.Screen, aircraft []*adsb.Aircraft, selectedICAO string) {
	m.canvas.Clear()

	m.renderer.RenderMap()

	m.renderer.RenderAircraft(aircraft, selectedICAO)

	m.canvas.Blit(screen, 0, 0)
}

// SetCenterFromFirstAircraft sets the map center to the first aircraft with coordinates
func (m *MapView) SetCenterFromFirstAircraft(aircraft []*adsb.Aircraft) bool {
	if m.centerSet {
		return false 
	}

	for _, ac := range aircraft {
		if ac.PositionLocked() {
			m.projection.UpdateCenter(*ac.Latitude, *ac.Longitude)
			m.centerSet = true

			// Debug logging
			bounds := m.projection.GetBounds()
			debug.Log("Map centered on aircraft %s at %.4f, %.4f", ac.ICAO, *ac.Latitude, *ac.Longitude)
			debug.Log("Visible bounds: lat[%.2f to %.2f] lon[%.2f to %.2f]",
				bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)

			return true
		}
	}

	return false
}

// UpdateDimensions updates the view dimensions when the screen is resized
func (m *MapView) UpdateDimensions(width, height int) {
	m.width = width
	m.height = height

	m.projection.UpdateDimensions(width, height)

	m.canvas = render.NewCanvas(width, height)
	m.renderer.UpdateCanvas(m.canvas)
}

// GetProjection returns the current projection
func (m *MapView) GetProjection() *geo.Projection {
	return m.projection
}

// CenterOnAircraft centers the map on a specific aircraft
func (m *MapView) CenterOnAircraft(ac *adsb.Aircraft) {
	if ac == nil || !ac.PositionLocked() {
		return
	}

	m.projection.UpdateCenter(*ac.Latitude, *ac.Longitude)
	m.centerSet = true

	debug.Log("Map re-centered on aircraft %s at %.4f, %.4f", ac.ICAO, *ac.Latitude, *ac.Longitude)
}

// ZoomIn decreases the radius (zooms in)
func (m *MapView) ZoomIn() {
	newRadius := m.radiusMiles * 0.75 
	if newRadius < 10 {
		newRadius = 10 
	}
	m.SetRadius(newRadius)
}

// ZoomOut increases the radius (zooms out)
func (m *MapView) ZoomOut() {
	newRadius := m.radiusMiles * 1.33 
	if newRadius > 1000 {
		newRadius = 1000 
	}
	m.SetRadius(newRadius)
}

// SetRadius updates the map radius and recalculates the projection
func (m *MapView) SetRadius(radiusMiles float64) {
	m.radiusMiles = radiusMiles
	centerLat, centerLon := m.projection.GetCenter()
	m.projection = geo.NewProjection(centerLat, centerLon, radiusMiles, m.width, m.height, m.aspectRatio)
	m.renderer.UpdateProjection(m.projection)
	debug.Log("Map radius changed to %.0f miles", radiusMiles)
}

// GetRadius returns the current map radius
func (m *MapView) GetRadius() float64 {
	return m.radiusMiles
}
