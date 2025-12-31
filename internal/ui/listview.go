package ui

import (
	"ascii1090/internal/adsb"
	"ascii1090/internal/render"
	"github.com/gdamore/tcell/v2"
)

// ListView displays a scrollable list of aircraft
type ListView struct {
	aircraft      []*adsb.Aircraft
	selectedIndex int
	scrollOffset  int
	maxVisible    int
	x, y          int
	width, height int
}

// NewListView creates a new aircraft list view
func NewListView(x, y, width, height int) *ListView {
	maxVisible := height - 2 // Account for border
	if maxVisible < 1 {
		maxVisible = 1
	}

	return &ListView{
		aircraft:      make([]*adsb.Aircraft, 0),
		selectedIndex: 0,
		scrollOffset:  0,
		maxVisible:    maxVisible,
		x:             x,
		y:             y,
		width:         width,
		height:        height,
	}
}

// Update refreshes the aircraft list
func (l *ListView) Update(aircraft []*adsb.Aircraft) {
	l.aircraft = aircraft

	if l.selectedIndex >= len(l.aircraft) {
		l.selectedIndex = len(l.aircraft) - 1
	}
	if l.selectedIndex < 0 {
		l.selectedIndex = 0
	}

	l.adjustScroll()
}

// SelectNext moves selection down
func (l *ListView) SelectNext() {
	if l.selectedIndex < len(l.aircraft)-1 {
		l.selectedIndex++
		l.adjustScroll()
	}
}

// SelectPrev moves selection up
func (l *ListView) SelectPrev() {
	if l.selectedIndex > 0 {
		l.selectedIndex--
		l.adjustScroll()
	}
}

// adjustScroll adjusts scroll offset to keep selected item visible
func (l *ListView) adjustScroll() {
	if l.selectedIndex >= l.scrollOffset+l.maxVisible {
		l.scrollOffset = l.selectedIndex - l.maxVisible + 1
	}

	if l.selectedIndex < l.scrollOffset {
		l.scrollOffset = l.selectedIndex
	}

	if l.scrollOffset < 0 {
		l.scrollOffset = 0
	}
}

// GetSelected returns the currently selected aircraft
func (l *ListView) GetSelected() *adsb.Aircraft {
	if l.selectedIndex >= 0 && l.selectedIndex < len(l.aircraft) {
		return l.aircraft[l.selectedIndex]
	}
	return nil
}

// Draw renders the list view to the screen
func (l *ListView) Draw(screen tcell.Screen) {
	// Clear the entire panel area first (make it opaque)
	defaultStyle := tcell.StyleDefault
	for row := l.y + 1; row < l.y+l.height-1; row++ {
		for col := l.x + 1; col < l.x+l.width-1; col++ {
			screen.SetContent(col, row, ' ', nil, defaultStyle)
		}
	}

	l.drawBorder(screen)

	title := "Aircraft"
	titleX := l.x + (l.width-len(title))/2
	for i, ch := range title {
		screen.SetContent(titleX+i, l.y, ch, nil, render.StyleLabel)
	}

	visibleCount := min(l.maxVisible, len(l.aircraft)-l.scrollOffset)
	for i := 0; i < visibleCount; i++ {
		acIndex := l.scrollOffset + i
		if acIndex >= len(l.aircraft) {
			break
		}

		ac := l.aircraft[acIndex]
		text := ac.ListDisplay()

		style := render.StyleListItem
		if acIndex == l.selectedIndex {
			style = render.StyleListSelected
		}

		x := l.x + 1
		y := l.y + i + 1
		for j := 0; j < min(len(text), l.width-2); j++ {
			if j < len(text) {
				screen.SetContent(x+j, y, rune(text[j]), nil, style)
			}
		}

		for j := len(text); j < l.width-2; j++ {
			screen.SetContent(x+j, y, ' ', nil, style)
		}
	}

	if len(l.aircraft) > l.maxVisible {
		scrollInfo := "↕"
		screen.SetContent(l.x+l.width-2, l.y, rune(scrollInfo[0]), nil, render.StyleLabel)
	}
}

// drawBorder draws the list border
func (l *ListView) drawBorder(screen tcell.Screen) {
	style := render.StyleLabel

	screen.SetContent(l.x, l.y, '┌', nil, style)
	screen.SetContent(l.x+l.width-1, l.y, '┐', nil, style)
	screen.SetContent(l.x, l.y+l.height-1, '└', nil, style)
	screen.SetContent(l.x+l.width-1, l.y+l.height-1, '┘', nil, style)

	for i := 1; i < l.width-1; i++ {
		screen.SetContent(l.x+i, l.y, '─', nil, style)
		screen.SetContent(l.x+i, l.y+l.height-1, '─', nil, style)
	}

	for i := 1; i < l.height-1; i++ {
		screen.SetContent(l.x, l.y+i, '│', nil, style)
		screen.SetContent(l.x+l.width-1, l.y+i, '│', nil, style)
	}
}

// UpdateDimensions updates the view dimensions
func (l *ListView) UpdateDimensions(x, y, width, height int) {
	l.x = x
	l.y = y
	l.width = width
	l.height = height
	l.maxVisible = height - 2
	if l.maxVisible < 1 {
		l.maxVisible = 1
	}
	l.adjustScroll()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
