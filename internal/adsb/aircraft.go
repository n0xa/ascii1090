package adsb

import (
	"fmt"
	"time"
)

// Aircraft represents an ADS-B transponder broadcast from an aircraft
type Aircraft struct {
	ICAO          string     // ICAO hex identifier (e.g., "A12345")
	FlightNumber  string     // Flight number (e.g., "UAL123"), empty if not available
	Latitude      *float64   // Decimal degrees (nil if not locked)
	Longitude     *float64   // Decimal degrees (nil if not locked)
	Altitude      int        // Feet above sea level
	Speed         int        // Ground speed in knots
	Heading       int        // Heading in degrees (0-359)
	Track         int        // Ground track in degrees (0-359)
	VerticalRate  int        // Vertical rate in feet per minute
	LastSeen      time.Time  // Last update timestamp
}

// FlightLevel returns the altitude divided by 100 (Flight Level)
func (a *Aircraft) FlightLevel() int {
	return a.Altitude / 100
}

// PositionLocked returns true if the aircraft has valid coordinates
func (a *Aircraft) PositionLocked() bool {
	return a.Latitude != nil && a.Longitude != nil
}

// CardinalDirection returns a directional arrow based on the aircraft's heading
// Returns 8-direction arrows using box-drawing characters for diagonals
// N: ^, NE: ┐, E: >, SE: ┘, S: v, SW: └, W: <, NW: ┌
func (a *Aircraft) CardinalDirection() rune {
	// Use track if available, fall back to heading
	direction := a.Track
	if direction == 0 && a.Heading != 0 {
		direction = a.Heading
	}

	// Normalize to 0-359
	direction = direction % 360
	if direction < 0 {
		direction += 360
	}

	// Divide into 8 directions with 45-degree arcs
	// Each direction covers 45 degrees centered on the cardinal/intercardinal point
	switch {
	case direction >= 338 || direction < 23:
		return '^' // North (337.5° - 22.5°)
	case direction >= 23 && direction < 68:
		return '┐' // Northeast (22.5° - 67.5°) - ASCII 191
	case direction >= 68 && direction < 113:
		return '>' // East (67.5° - 112.5°)
	case direction >= 113 && direction < 158:
		return '┘' // Southeast (112.5° - 157.5°) - ASCII 217
	case direction >= 158 && direction < 203:
		return 'v' // South (157.5° - 202.5°)
	case direction >= 203 && direction < 248:
		return '└' // Southwest (202.5° - 247.5°) - ASCII 192
	case direction >= 248 && direction < 293:
		return '<' // West (247.5° - 292.5°)
	case direction >= 293 && direction < 338:
		return '┌' // Northwest (292.5° - 337.5°) - ASCII 218
	default:
		return '·' // Unknown (shouldn't happen)
	}
}

// IsStale returns true if the aircraft hasn't been seen in 60+ seconds
func (a *Aircraft) IsStale() bool {
	return time.Since(a.LastSeen) >= 60*time.Second
}

// DisplayName returns the flight number if available, otherwise the ICAO hex
func (a *Aircraft) DisplayName() string {
	if a.FlightNumber != "" {
		return a.FlightNumber
	}
	return a.ICAO
}

// PositionString returns a formatted lat/lon string
func (a *Aircraft) PositionString() string {
	if !a.PositionLocked() {
		return "Position Unknown"
	}

	lat := *a.Latitude
	lon := *a.Longitude

	latDir := "N"
	if lat < 0 {
		latDir = "S"
		lat = -lat
	}

	lonDir := "E"
	if lon < 0 {
		lonDir = "W"
		lon = -lon
	}

	return fmt.Sprintf("%.4f*%s, %.4f*%s", lat, latDir, lon, lonDir)
}

// SecondsSinceLastSeen returns the number of seconds since the aircraft was last seen
func (a *Aircraft) SecondsSinceLastSeen() int {
	return int(time.Since(a.LastSeen).Seconds())
}

// ListDisplay returns the formatted string for the aircraft list
// Format: "(+) UAL123 FL450 500kts" or "( ) A12345 FL0 0kts"
func (a *Aircraft) ListDisplay() string {
	indicator := "( )"
	if a.PositionLocked() {
		indicator = "(+)"
	}

	return fmt.Sprintf("%s %-7s FL%-3d %3dkts",
		indicator,
		a.DisplayName(),
		a.FlightLevel(),
		a.Speed)
}
