package geo

import (
	"fmt"
	"math"
	"strings"

	"github.com/jonas-p/go-shp"
)

// ShapefileLoader loads and parses ESRI shapefiles
type ShapefileLoader struct {
	dataDir string
}

// NewShapefileLoader creates a new shapefile loader
func NewShapefileLoader(dataDir string) *ShapefileLoader {
	return &ShapefileLoader{
		dataDir: dataDir,
	}
}

// LoadAll loads all required shapefiles and returns them organized by feature type
// Missing files will be skipped with a warning - app can function with just aircraft
// highwayDetail is the scalerank threshold for highways (lower = fewer roads)
func (s *ShapefileLoader) LoadAll(highwayDetail int) (map[FeatureType][]*Feature, error) {
	features := make(map[FeatureType][]*Feature)

	// Load state borders (50m resolution)
	states, err := s.LoadShapefile(s.dataDir+"/ne_50m_admin_1_states_provinces.shp", FeatureStateBorder)
	if err != nil {
		fmt.Printf("Warning: failed to load states: %v\n", err)
		features[FeatureStateBorder] = []*Feature{}
	} else {
		features[FeatureStateBorder] = states
	}

	// Load rivers (50m resolution)
	rivers, err := s.LoadShapefile(s.dataDir+"/ne_50m_rivers_lake_centerlines.shp", FeatureRiver)
	if err != nil {
		fmt.Printf("Warning: failed to load rivers: %v\n", err)
		features[FeatureRiver] = []*Feature{}
	} else {
		features[FeatureRiver] = rivers
	}

	// Load coastlines (50m resolution)
	coasts, err := s.LoadShapefile(s.dataDir+"/ne_50m_coastline.shp", FeatureCoastline)
	if err != nil {
		fmt.Printf("Warning: failed to load coastlines: %v\n", err)
		features[FeatureCoastline] = []*Feature{}
	} else {
		features[FeatureCoastline] = coasts
	}

	// Load highways/roads (10m resolution - North America)
	// Filter by scalerank threshold (lower = fewer roads)
	highways, err := s.LoadHighways(s.dataDir+"/ne_10m_roads_north_america.shp", highwayDetail)
	if err != nil {
		fmt.Printf("Warning: failed to load highways: %v\n", err)
		features[FeatureHighway] = []*Feature{}
	} else {
		features[FeatureHighway] = highways
	}

	// Load cities (50m resolution)
	cities, err := s.LoadCities(s.dataDir + "/ne_50m_populated_places.shp")
	if err != nil {
		fmt.Printf("Warning: failed to load cities: %v\n", err)
		features[FeatureCity] = []*Feature{}
	} else {
		features[FeatureCity] = cities
	}

	// Load airports from CSV
	airportLoader := NewAirportLoader(s.dataDir + "/airports.csv")
	airports, err := airportLoader.LoadAirports()
	if err != nil {
		fmt.Printf("Warning: failed to load airports: %v\n", err)
		features[FeatureAirport] = []*Feature{}
	} else {
		features[FeatureAirport] = airports
	}

	// Show feature counts
	fmt.Printf("Loaded features: %d states, %d rivers, %d coastlines, %d highways, %d cities, %d airports\n",
		len(features[FeatureStateBorder]),
		len(features[FeatureRiver]),
		len(features[FeatureCoastline]),
		len(features[FeatureHighway]),
		len(features[FeatureCity]),
		len(features[FeatureAirport]))
	return features, nil
}

// LoadShapefile loads a shapefile and converts it to Feature objects
func (s *ShapefileLoader) LoadShapefile(path string, ftype FeatureType) ([]*Feature, error) {
	shape, err := shp.Open(path)
	if err != nil {
		return nil, err
	}
	defer shape.Close()

	features := make([]*Feature, 0)

	// Read all features
	for shape.Next() {
		_, p := shape.Shape()

		switch geom := p.(type) {
		case *shp.PolyLine:
			// Convert polyline points to features
			// In shapefiles, all points are in the Points array
			points := make([]LatLon, len(geom.Points))
			for i, point := range geom.Points {
				points[i] = LatLon{
					Lat: point.Y,
					Lon: point.X,
				}
			}
			if len(points) > 1 {
				features = append(features, NewLineFeature(ftype, points))
			}

		case *shp.Polygon:
			// Convert polygon points to line features (just the outline)
			points := make([]LatLon, len(geom.Points))
			for i, point := range geom.Points {
				points[i] = LatLon{
					Lat: point.Y,
					Lon: point.X,
				}
			}
			if len(points) > 1 {
				features = append(features, NewLineFeature(ftype, points))
			}

		case *shp.Point:
			// Point feature
			feature := NewPointFeature(ftype, LatLon{Lat: geom.Y, Lon: geom.X}, "")
			features = append(features, feature)
		}
	}

	return features, nil
}

// LoadCities loads city/populated place features with names
func (s *ShapefileLoader) LoadCities(path string) ([]*Feature, error) {
	shape, err := shp.Open(path)
	if err != nil {
		return nil, err
	}
	defer shape.Close()

	features := make([]*Feature, 0)

	// Read all features
	for shape.Next() {
		n, p := shape.Shape()

		point, ok := p.(*shp.Point)
		if !ok {
			continue
		}

		// Try to get the name from attributes
		name := ""
		if shape.AttributeCount() > n {
			fields := shape.Fields()
			for i, field := range fields {
				// Look for name field (NAME, NAMEASCII, etc.)
				// Field names in shapefiles are byte arrays, convert to string and trim nulls
				fieldName := string(field.Name[:])
				// Trim null bytes and spaces
				fieldName = strings.TrimRight(fieldName, "\x00 ")

				if fieldName == "NAME" || fieldName == "NAMEASCII" || fieldName == "NAME_EN" {
					if attr := shape.ReadAttribute(n, i); attr != "" {
						name = strings.TrimSpace(attr)
						break
					}
				}
			}
		}

		feature := NewPointFeature(FeatureCity, LatLon{Lat: point.Y, Lon: point.X}, name)
		features = append(features, feature)
	}

	return features, nil
}

// LoadHighways loads highway/road features with filtering for major roads only
// maxScalerank is the threshold - only roads with scalerank <= maxScalerank are loaded
func (s *ShapefileLoader) LoadHighways(path string, maxScalerank int) ([]*Feature, error) {
	shape, err := shp.Open(path)
	if err != nil {
		return nil, err
	}
	defer shape.Close()

	features := make([]*Feature, 0)

	// Find the scalerank field index
	scalerankIdx := -1
	fields := shape.Fields()
	for i, field := range fields {
		fieldName := strings.TrimRight(string(field.Name[:]), "\x00 ")
		if fieldName == "scalerank" {
			scalerankIdx = i
			break
		}
	}

	fmt.Printf("Loading highways with scalerank filtering (scalerank <= %d)...\n", maxScalerank)

	// Read all features
	for shape.Next() {
		n, p := shape.Shape()

		// Filter by scalerank if available
		if scalerankIdx >= 0 {
			scalerankStr := shape.ReadAttribute(n, scalerankIdx)
			if scalerankStr != "" {
				var scalerank int
				if _, err := fmt.Sscanf(scalerankStr, "%d", &scalerank); err == nil {
					if scalerank > maxScalerank {
						continue // Skip roads above threshold
					}
				}
			}
		}

		switch geom := p.(type) {
		case *shp.PolyLine:
			// Convert polyline points to features
			points := make([]LatLon, len(geom.Points))
			for i, point := range geom.Points {
				points[i] = LatLon{
					Lat: point.Y,
					Lon: point.X,
				}
			}
			if len(points) > 1 {
				features = append(features, NewLineFeature(FeatureHighway, points))
			}
		}
	}

	return features, nil
}

// FilterByBounds filters features to only those within or intersecting the given bounds
func FilterByBounds(features []*Feature, bounds *Bounds) []*Feature {
	filtered := make([]*Feature, 0)

	for _, feature := range features {
		if feature.IsPoint() {
			// Check if point is within bounds
			if bounds.Contains(feature.Point.Lat, feature.Point.Lon) {
				filtered = append(filtered, feature)
			}
		} else if feature.IsLine() {
			// Check if any point in the line is within bounds
			// (More sophisticated clipping could be added later)
			hasPointInBounds := false
			for _, point := range feature.Points {
				if bounds.Contains(point.Lat, point.Lon) {
					hasPointInBounds = true
					break
				}
			}
			if hasPointInBounds {
				filtered = append(filtered, feature)
			}
		}
	}

	return filtered
}

// Bounds represents a geographic bounding box
type Bounds struct {
	MinLat float64
	MaxLat float64
	MinLon float64
	MaxLon float64
}

// NewBounds creates a bounding box from center point and radius
func NewBounds(centerLat, centerLon, radiusMiles float64) *Bounds {
	// Approximate degrees per mile at this latitude
	// 1 degree latitude ≈ 69 miles
	// 1 degree longitude ≈ 69 * cos(latitude) miles
	latDegrees := radiusMiles / 69.0
	lonDegrees := radiusMiles / (69.0 * math.Cos(centerLat*math.Pi/180.0))

	return &Bounds{
		MinLat: centerLat - latDegrees,
		MaxLat: centerLat + latDegrees,
		MinLon: centerLon - lonDegrees,
		MaxLon: centerLon + lonDegrees,
	}
}

// Contains checks if a point is within the bounds
func (b *Bounds) Contains(lat, lon float64) bool {
	return lat >= b.MinLat && lat <= b.MaxLat &&
		lon >= b.MinLon && lon <= b.MaxLon
}
