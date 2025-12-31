package render

import (
	"github.com/gdamore/tcell/v2"
)

// Canvas represents a 2D grid of cells for ASCII rendering
type Canvas struct {
	width  int
	height int
	cells  [][]Cell
}

// Cell represents a single character cell with style
type Cell struct {
	Char  rune
	Style tcell.Style
}

// NewCanvas creates a new blank canvas
func NewCanvas(width, height int) *Canvas {
	cells := make([][]Cell, height)
	for i := range cells {
		cells[i] = make([]Cell, width)
		// Initialize with spaces and default style
		for j := range cells[i] {
			cells[i][j] = Cell{
				Char:  ' ',
				Style: tcell.StyleDefault,
			}
		}
	}

	return &Canvas{
		width:  width,
		height: height,
		cells:  cells,
	}
}

// Set sets the character and style at the given position
// Coordinates are 0-indexed with (0,0) at top-left
func (c *Canvas) Set(x, y int, char rune, style tcell.Style) {
	if x >= 0 && x < c.width && y >= 0 && y < c.height {
		c.cells[y][x] = Cell{Char: char, Style: style}
	}
}

// Get retrieves the cell at the given position
func (c *Canvas) Get(x, y int) Cell {
	if x >= 0 && x < c.width && y >= 0 && y < c.height {
		return c.cells[y][x]
	}
	return Cell{Char: ' ', Style: tcell.StyleDefault}
}

// Clear resets the entire canvas to spaces with default style
func (c *Canvas) Clear() {
	for y := range c.cells {
		for x := range c.cells[y] {
			c.cells[y][x] = Cell{Char: ' ', Style: tcell.StyleDefault}
		}
	}
}

// ClearRegion clears a rectangular region
func (c *Canvas) ClearRegion(x, y, width, height int) {
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			c.Set(x+dx, y+dy, ' ', tcell.StyleDefault)
		}
	}
}

// DrawText draws a string at the given position
func (c *Canvas) DrawText(x, y int, text string, style tcell.Style) {
	for i, char := range text {
		c.Set(x+i, y, char, style)
	}
}

// DrawBox draws a box outline using box-drawing characters
func (c *Canvas) DrawBox(x, y, width, height int, style tcell.Style) {
	if width < 2 || height < 2 {
		return
	}

	// Corners
	c.Set(x, y, '┌', style)
	c.Set(x+width-1, y, '┐', style)
	c.Set(x, y+height-1, '└', style)
	c.Set(x+width-1, y+height-1, '┘', style)

	// Horizontal lines
	for i := 1; i < width-1; i++ {
		c.Set(x+i, y, '─', style)
		c.Set(x+i, y+height-1, '─', style)
	}

	// Vertical lines
	for i := 1; i < height-1; i++ {
		c.Set(x, y+i, '│', style)
		c.Set(x+width-1, y+i, '│', style)
	}
}

// FillRect fills a rectangle with a character
func (c *Canvas) FillRect(x, y, width, height int, char rune, style tcell.Style) {
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			c.Set(x+dx, y+dy, char, style)
		}
	}
}

// Width returns the canvas width
func (c *Canvas) Width() int {
	return c.width
}

// Height returns the canvas height
func (c *Canvas) Height() int {
	return c.height
}

// Blit renders the canvas to a tcell screen
func (c *Canvas) Blit(screen tcell.Screen, offsetX, offsetY int) {
	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			cell := c.cells[y][x]
			screen.SetContent(offsetX+x, offsetY+y, cell.Char, nil, cell.Style)
		}
	}
}
