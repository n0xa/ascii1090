package geo

import (
	"math"
)

// Point represents a screen coordinate
type Point struct {
	X int
	Y int
}

// Projection handles conversion from lat/lon to screen coordinates
type Projection struct {
	centerLat    float64
	centerLon    float64
	radiusMiles  float64
	screenWidth  int
	screenHeight int
	aspectRatio  float64 
	scaleX       float64
	scaleY       float64
}

// NewProjection creates an equirectangular projection for a given center point and radius
// The projection will fit a circle of radiusMiles around the center point into the screen dimensions
// aspectRatio compensates for character dimensions (typically 2.0 for characters twice as tall as wide)
func NewProjection(centerLat, centerLon, radiusMiles float64, screenWidth, screenHeight int, aspectRatio float64) *Projection {
	p := &Projection{
		centerLat:    centerLat,
		centerLon:    centerLon,
		radiusMiles:  radiusMiles,
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		aspectRatio:  aspectRatio,
	}

	p.calculateScale()
	return p
}

// calculateScale computes the pixels-per-degree scaling factors
func (p *Projection) calculateScale() {
	// 1 degree latitude ≈ 69 miles (constant)
	// 1 degree longitude ≈ 69 * cos(latitude) miles

	milesPerDegreeLat := 69.0
	milesPerDegreeLon := 69.0 * math.Cos(p.centerLat*math.Pi/180.0)

	degreesLat := p.radiusMiles / milesPerDegreeLat
	degreesLon := p.radiusMiles / milesPerDegreeLon

	totalDegreesLat := degreesLat * 2
	totalDegreesLon := degreesLon * 2

	effectiveHeight := float64(p.screenHeight) * p.aspectRatio
	scaleY := effectiveHeight / totalDegreesLat
	scaleX := float64(p.screenWidth) / totalDegreesLon

	if scaleX < scaleY {
		p.scaleX = scaleX
		p.scaleY = scaleX / p.aspectRatio 
	} else {
		p.scaleX = scaleY * p.aspectRatio
		p.scaleY = scaleY
	}
}

// Project converts lat/lon to screen coordinates
// Returns screen coordinates with (0, 0) at top-left
func (p *Projection) Project(lat, lon float64) Point {
	deltaLat := lat - p.centerLat
	deltaLon := lon - p.centerLon

	// Convert to pixels
	// Note: Y is inverted (positive lat goes up, but positive screen Y goes down)
	x := int(deltaLon * p.scaleX)
	y := int(-deltaLat * p.scaleY) // Negative because screen Y increases downward

	// Translate to screen center
	x += p.screenWidth / 2
	y += p.screenHeight / 2

	return Point{X: x, Y: y}
}

// Unproject converts screen coordinates back to lat/lon
func (p *Projection) Unproject(x, y int) (lat, lon float64) {
	// Translate from screen center
	x -= p.screenWidth / 2
	y -= p.screenHeight / 2

	// Convert from pixels to degrees
	deltaLon := float64(x) / p.scaleX
	deltaLat := -float64(y) / p.scaleY // Negative because screen Y is inverted

	lat = p.centerLat + deltaLat
	lon = p.centerLon + deltaLon

	return lat, lon
}

// IsInBounds checks if a lat/lon point would be visible on screen
func (p *Projection) IsInBounds(lat, lon float64) bool {
	point := p.Project(lat, lon)
	return point.X >= 0 && point.X < p.screenWidth &&
		point.Y >= 0 && point.Y < p.screenHeight
}

// UpdateCenter recalculates the projection with a new center point
func (p *Projection) UpdateCenter(lat, lon float64) {
	p.centerLat = lat
	p.centerLon = lon
	p.calculateScale()
}

// UpdateDimensions updates the screen dimensions and recalculates scaling
func (p *Projection) UpdateDimensions(width, height int) {
	p.screenWidth = width
	p.screenHeight = height
	p.calculateScale()
}

// GetCenter returns the current center point
func (p *Projection) GetCenter() (lat, lon float64) {
	return p.centerLat, p.centerLon
}

// GetBounds returns the geographic bounds visible on screen
func (p *Projection) GetBounds() *Bounds {
	topLeftLat, topLeftLon := p.Unproject(0, 0)
	bottomRightLat, bottomRightLon := p.Unproject(p.screenWidth-1, p.screenHeight-1)

	minLat := math.Min(topLeftLat, bottomRightLat)
	maxLat := math.Max(topLeftLat, bottomRightLat)
	minLon := math.Min(topLeftLon, bottomRightLon)
	maxLon := math.Max(topLeftLon, bottomRightLon)

	return &Bounds{
		MinLat: minLat,
		MaxLat: maxLat,
		MinLon: minLon,
		MaxLon: maxLon,
	}
}
