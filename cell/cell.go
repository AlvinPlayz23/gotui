// Package cell implements the terminal cell grid and drawing primitives.
package cell

import (
	"strings"

	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/width"
)

// Attribute bits, matching OpenTUI's text attributes.
type Attribute uint16

const (
	AttrNone          Attribute = 0
	AttrBold          Attribute = 1 << 0
	AttrDim           Attribute = 1 << 1
	AttrItalic        Attribute = 1 << 2
	AttrUnderline     Attribute = 1 << 3
	AttrBlink         Attribute = 1 << 4
	AttrInverse       Attribute = 1 << 5
	AttrHidden        Attribute = 1 << 6
	AttrStrikethrough Attribute = 1 << 7
)

// Has reports whether all bits in a are set.
func (a Attribute) Has(other Attribute) bool { return a&other == other }

// Cell is one terminal cell. A wide grapheme occupies its first cell with
// Width 2 and a following continuation cell with Width 0.
type Cell struct {
	Ch      rune
	Cluster string // non-empty for multi-rune graphemes
	Width   uint8
	FG      color.RGBA
	BG      color.RGBA
	Attr    Attribute
	Link    uint32
}

// Glyph returns the text to draw for the cell.
func (c Cell) Glyph() string {
	if c.Cluster != "" {
		return c.Cluster
	}
	if c.Ch == 0 {
		return " "
	}
	return string(c.Ch)
}

// IsContinuation reports whether the cell is the tail of a wide grapheme.
func (c Cell) IsContinuation() bool { return c.Width == 0 }

// Blank returns an empty cell with the given colors.
func Blank(bg color.RGBA) Cell {
	return Cell{Ch: ' ', Width: 1, BG: bg}
}

// Equal reports whether two cells render identically.
func (c Cell) Equal(o Cell) bool {
	return c.Ch == o.Ch && c.Cluster == o.Cluster && c.Width == o.Width &&
		c.FG == o.FG && c.BG == o.BG && c.Attr == o.Attr && c.Link == o.Link
}

// Buffer is a resizable terminal cell grid.
type Buffer struct {
	W, H      int
	Cells     []Cell
	DefaultBG color.RGBA
	DefaultFG color.RGBA
}

// New creates a buffer filled with blank cells.
func New(w, h int, bg color.RGBA) *Buffer {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	b := &Buffer{W: w, H: h, DefaultBG: bg}
	b.Cells = make([]Cell, w*h)
	blank := Blank(bg)
	for i := range b.Cells {
		b.Cells[i] = blank
	}
	return b
}

// Index returns the cell offset for x,y, or -1 if out of bounds.
func (b *Buffer) Index(x, y int) int {
	if x < 0 || y < 0 || x >= b.W || y >= b.H {
		return -1
	}
	return y*b.W + x
}

// At returns the cell at x,y and whether it is in bounds.
func (b *Buffer) At(x, y int) (Cell, bool) {
	i := b.Index(x, y)
	if i < 0 {
		return Cell{}, false
	}
	return b.Cells[i], true
}

// Resize changes the buffer dimensions, preserving overlapping content.
func (b *Buffer) Resize(w, h int) {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	nb := New(w, h, b.DefaultBG)
	for y := 0; y < min(h, b.H); y++ {
		for x := 0; x < min(w, b.W); x++ {
			nb.Cells[y*w+x] = b.Cells[y*b.W+x]
		}
	}
	b.W, b.H, b.Cells = w, h, nb.Cells
}

// Clear resets every cell to a blank with bg.
func (b *Buffer) Clear(bg color.RGBA) {
	b.DefaultBG = bg
	blank := Blank(bg)
	for i := range b.Cells {
		b.Cells[i] = blank
	}
}

// ClearCells clears the half-open region [x0,x1) x [y0,y1) to bg.
func (b *Buffer) ClearCells(x0, y0, x1, y1 int, bg color.RGBA) {
	x0, y0 = max(0, x0), max(0, y0)
	x1, y1 = min(b.W, x1), min(b.H, y1)
	blank := Blank(bg)
	for y := y0; y < y1; y++ {
		row := y * b.W
		for x := x0; x < x1; x++ {
			b.Cells[row+x] = blank
		}
	}
}

// Fill fills a rectangle with bg.
func (b *Buffer) Fill(x, y, w, h int, bg color.RGBA) {
	b.ClearCells(x, y, x+w, y+h, bg)
}

// Set writes one rune at x,y with the given style.
func (b *Buffer) Set(x, y int, ch rune, fg, bg color.RGBA, attr Attribute) int {
	return b.SetCell(x, y, Cell{Ch: ch, Width: 1, FG: fg, BG: bg, Attr: attr})
}

// SetCell writes a pre-built cell, handling wide-cell cleanup. It returns the
// number of cells consumed (0 if out of bounds).
func (b *Buffer) SetCell(x, y int, c Cell) int {
	i := b.Index(x, y)
	if i < 0 {
		return 0
	}
	w := int(c.Width)
	if w == 0 {
		w = 1
	}
	if w == 2 && x+1 >= b.W {
		// A wide glyph cannot fit at the right edge.
		return 0
	}
	b.cleanWide(x, y)
	if w == 2 {
		b.cleanWide(x+1, y)
	}
	c.Width = uint8(w)
	b.Cells[i] = c
	if w == 2 {
		b.Cells[i+1] = Cell{Ch: 0, Width: 0, FG: c.FG, BG: c.BG, Attr: c.Attr, Link: c.Link}
	}
	return w
}

func (b *Buffer) cleanWide(x, y int) {
	i := b.Index(x, y)
	if i < 0 {
		return
	}
	cur := b.Cells[i]
	if cur.Width == 0 && x > 0 {
		b.Cells[i-1] = Blank(b.DefaultBG)
	} else if cur.Width == 2 && x+1 < b.W {
		b.Cells[i+1] = Blank(b.DefaultBG)
	}
}

// SetString draws s at x,y and returns the columns consumed. It stops at the
// right edge and does not wrap.
func (b *Buffer) SetString(x, y int, s string, fg, bg color.RGBA, attr Attribute) int {
	if y < 0 || y >= b.H {
		return 0
	}
	col := 0
	width.Iterate(s, func(g width.Grapheme) bool {
		if x+col >= b.W {
			return false
		}
		if x+col+g.Width > b.W {
			return false
		}
		if g.Width == 0 {
			return true
		}
		c := Cell{FG: fg, BG: bg, Attr: attr, Width: uint8(g.Width)}
		if len(g.Text) == 1 {
			c.Ch = []rune(g.Text)[0]
		} else if len(g.Text) == 0 {
			return true
		} else {
			r := []rune(g.Text)
			if len(r) == 1 {
				c.Ch = r[0]
			} else {
				c.Cluster = g.Text
			}
		}
		b.SetCell(x+col, y, c)
		col += g.Width
		return true
	})
	return col
}

// SetStringClipped draws s clipped to [clipX, clipX+clipW).
func (b *Buffer) SetStringClipped(x, y, clipX, clipW int, s string, fg, bg color.RGBA, attr Attribute) int {
	start := max(x, clipX)
	end := min(x+width.StringWidth(s), clipX+clipW)
	if start >= end {
		return 0
	}
	sub := width.Slice(s, start-x, end-start)
	return b.SetString(start, y, sub, fg, bg, attr)
}

// SetBG changes the background color of the cell at x,y, preserving its
// glyph and attributes.
func (b *Buffer) SetBG(x, y int, bg color.RGBA) {
	i := b.Index(x, y)
	if i < 0 {
		return
	}
	b.Cells[i].BG = bg
}

// Text returns the buffer content as plain text rows separated by '\n'.
func (b *Buffer) Text() string {
	var sb strings.Builder
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			c := b.Cells[y*b.W+x]
			if c.Width == 0 {
				continue
			}
			sb.WriteString(c.Glyph())
		}
		if y != b.H-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// Clone returns a deep copy.
func (b *Buffer) Clone() *Buffer {
	nb := &Buffer{W: b.W, H: b.H, DefaultBG: b.DefaultBG, DefaultFG: b.DefaultFG}
	nb.Cells = append([]Cell(nil), b.Cells...)
	return nb
}

// ResizeCopy is Clone+Resize.
func (b *Buffer) ResizeCopy(w, h int) *Buffer {
	n := b.Clone()
	n.Resize(w, h)
	return n
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
