package ui

import (
	"ascii1090/internal/adsb"
	"ascii1090/internal/render"
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// DetailView displays detailed information about a selected aircraft
type DetailView struct {
	aircraft      *adsb.Aircraft
	x, y          int
	width, height int
}

// NewDetailView creates a new detail view
func NewDetailView(x, y, width, height int) *DetailView {
	return &DetailView{
		x:      x,
		y:      y,
		width:  width,
		height: height,
	}
}

// SetAircraft sets the aircraft to display
func (d *DetailView) SetAircraft(ac *adsb.Aircraft) {
	d.aircraft = ac
}

// Draw renders the detail view to the screen
func (d *DetailView) Draw(screen tcell.Screen) {
	if d.aircraft == nil {
		d.drawEmpty(screen)
		return
	}

	// Clear the entire panel area first (make it opaque)
	defaultStyle := tcell.StyleDefault
	for row := d.y + 1; row < d.y+d.height-1; row++ {
		for col := d.x + 1; col < d.x+d.width-1; col++ {
			screen.SetContent(col, row, ' ', nil, defaultStyle)
		}
	}

	// Draw border
	d.drawBorder(screen)

	// Draw title
	title := "Aircraft Details"
	titleX := d.x + (d.width-len(title))/2
	for i, ch := range title {
		screen.SetContent(titleX+i, d.y, ch, nil, render.StyleLabel)
	}

	// Draw aircraft information
	ac := d.aircraft
	lines := []string{
		fmt.Sprintf("ICAO:          %s", ac.ICAO),
		fmt.Sprintf("Flight:        %s", ac.DisplayName()),
		fmt.Sprintf("Position:      %s", ac.PositionString()),
		fmt.Sprintf("Altitude:      %d ft (FL%d)", ac.Altitude, ac.FlightLevel()),
		fmt.Sprintf("Speed:         %d kts", ac.Speed),
		fmt.Sprintf("Heading:       %d*", ac.Heading),
		fmt.Sprintf("Track:         %d*", ac.Track),
		fmt.Sprintf("Vertical Rate: %+d ft/min", ac.VerticalRate),
		fmt.Sprintf("Last Seen:     %d seconds ago", ac.SecondsSinceLastSeen()),
	}

	y := d.y + 1
	for i, line := range lines {
		if y+i >= d.y+d.height-1 {
			break
		}
		d.drawLine(screen, d.x+2, y+i, line)
	}

	// Add instructions at bottom
	instructions := "Press ESC to return"
	instX := d.x + (d.width-len(instructions))/2
	instY := d.y + d.height - 1
	for i, ch := range instructions {
		screen.SetContent(instX+i, instY, ch, nil, render.StyleLabel.Dim(true))
	}
}

// drawEmpty draws an empty detail view
func (d *DetailView) drawEmpty(screen tcell.Screen) {
	// Clear the entire panel area first (make it opaque)
	defaultStyle := tcell.StyleDefault
	for row := d.y + 1; row < d.y+d.height-1; row++ {
		for col := d.x + 1; col < d.x+d.width-1; col++ {
			screen.SetContent(col, row, ' ', nil, defaultStyle)
		}
	}

	d.drawBorder(screen)
	text := "No aircraft selected"
	x := d.x + (d.width-len(text))/2
	y := d.y + d.height/2
	for i, ch := range text {
		screen.SetContent(x+i, y, ch, nil, render.StyleLabel)
	}
}

// drawLine draws a single line of text
func (d *DetailView) drawLine(screen tcell.Screen, x, y int, text string) {
	for i := 0; i < min(len(text), d.width-4); i++ {
		screen.SetContent(x+i, y, rune(text[i]), nil, render.StyleLabel)
	}
}

// drawBorder draws the detail view border
func (d *DetailView) drawBorder(screen tcell.Screen) {
	style := render.StyleLabel

	screen.SetContent(d.x, d.y, '┌', nil, style)
	screen.SetContent(d.x+d.width-1, d.y, '┐', nil, style)
	screen.SetContent(d.x, d.y+d.height-1, '└', nil, style)
	screen.SetContent(d.x+d.width-1, d.y+d.height-1, '┘', nil, style)

	for i := 1; i < d.width-1; i++ {
		screen.SetContent(d.x+i, d.y, '─', nil, style)
		screen.SetContent(d.x+i, d.y+d.height-1, '─', nil, style)
	}

	for i := 1; i < d.height-1; i++ {
		screen.SetContent(d.x, d.y+i, '│', nil, style)
		screen.SetContent(d.x+d.width-1, d.y+i, '│', nil, style)
	}
}

// UpdateDimensions updates the view dimensions
func (d *DetailView) UpdateDimensions(x, y, width, height int) {
	d.x = x
	d.y = y
	d.width = width
	d.height = height
}
