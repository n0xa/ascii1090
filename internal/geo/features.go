package geo

// FeatureType represents the type of geographic feature
type FeatureType int

const (
	FeatureStateBorder FeatureType = iota
	FeatureHighway
	FeatureRiver
	FeatureCoastline
	FeatureCity
	FeatureAirport
)

// String returns a string representation of the feature type
func (f FeatureType) String() string {
	switch f {
	case FeatureStateBorder:
		return "StateBorder"
	case FeatureHighway:
		return "Highway"
	case FeatureRiver:
		return "River"
	case FeatureCoastline:
		return "Coastline"
	case FeatureCity:
		return "City"
	case FeatureAirport:
		return "Airport"
	default:
		return "Unknown"
	}
}

// LatLon represents a geographic coordinate
type LatLon struct {
	Lat float64
	Lon float64
}

// Feature represents a geographic feature (line, polygon, or point)
type Feature struct {
	Type       FeatureType        // Type of feature
	Points     []LatLon           // Polyline/polygon points (empty for point features)
	Point      *LatLon            // Single point (for cities, airports)
	Name       string             // Label for cities, airports, etc.
	Properties map[string]interface{} // Additional properties from shapefile
}

// NewLineFeature creates a new line/polyline feature
func NewLineFeature(ftype FeatureType, points []LatLon) *Feature {
	return &Feature{
		Type:       ftype,
		Points:     points,
		Properties: make(map[string]interface{}),
	}
}

// NewPointFeature creates a new point feature (city, airport)
func NewPointFeature(ftype FeatureType, point LatLon, name string) *Feature {
	return &Feature{
		Type:       ftype,
		Point:      &point,
		Name:       name,
		Properties: make(map[string]interface{}),
	}
}

// IsPoint returns true if this is a point feature
func (f *Feature) IsPoint() bool {
	return f.Point != nil
}

// IsLine returns true if this is a line/polyline feature
func (f *Feature) IsLine() bool {
	return len(f.Points) > 0
}
