package render

import (
	"ascii1090/internal/geo"

	"github.com/gdamore/tcell/v2"
)

// Style definitions for different map features
var (
	StyleStateBorder = tcell.StyleDefault.Foreground(tcell.ColorDarkGray)
	StyleHighway     = tcell.StyleDefault.Foreground(tcell.ColorYellow)
	StyleRiver       = tcell.StyleDefault.Foreground(tcell.ColorDarkCyan)
	StyleCoastline   = tcell.StyleDefault.Foreground(tcell.ColorDarkBlue)
	StyleCity        = tcell.StyleDefault.Foreground(tcell.ColorWhite)
	StyleAirport     = tcell.StyleDefault.Foreground(tcell.ColorOrange)
	StyleAircraft    = tcell.StyleDefault.Foreground(tcell.ColorGreen).Bold(true)
	StyleSelected    = tcell.StyleDefault.Foreground(tcell.ColorGreen).Bold(true).Reverse(true)
	StyleLabel       = tcell.StyleDefault.Foreground(tcell.ColorWhite)
	StyleListItem    = tcell.StyleDefault.Foreground(tcell.ColorWhite)
	StyleListSelected = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite)
)

// GetStyleForFeature returns the appropriate style for a feature type
func GetStyleForFeature(ftype geo.FeatureType) tcell.Style {
	switch ftype {
	case geo.FeatureStateBorder:
		return StyleStateBorder
	case geo.FeatureHighway:
		return StyleHighway
	case geo.FeatureRiver:
		return StyleRiver
	case geo.FeatureCoastline:
		return StyleCoastline
	case geo.FeatureCity:
		return StyleCity
	case geo.FeatureAirport:
		return StyleAirport
	default:
		return tcell.StyleDefault
	}
}

// GetCharForFeature returns the appropriate character for drawing a feature
func GetCharForFeature(ftype geo.FeatureType) rune {
	switch ftype {
	case geo.FeatureStateBorder:
		return '-' // Simple dash for borders
	case geo.FeatureHighway:
		return '=' // Double line for highways
	case geo.FeatureRiver:
		return '~' // Wavy for rivers
	case geo.FeatureCoastline:
		return '-' // Dash for coastlines
	default:
		return '·'
	}
}
