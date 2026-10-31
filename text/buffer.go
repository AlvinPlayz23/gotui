package text

import (
	"strings"

	"github.com/bijim/gotui/width"
	"github.com/rivo/uniseg"
)

// TextBuffer is an editable UTF-8 text buffer with line access and tab
// handling. It wraps a Rope for storage.
type TextBuffer struct {
	Rope     *Rope
	TabWidth int
}

// NewTextBuffer creates a buffer. TabWidth defaults to 4.
func NewTextBuffer(s string) *TextBuffer {
	return &TextBuffer{Rope: NewRope(s), TabWidth: 4}
}

// Text returns the buffer content.
func (b *TextBuffer) Text() string { return b.Rope.String() }

// Len returns the length in bytes.
func (b *TextBuffer) Len() int { return b.Rope.Len() }

// SetText replaces the content.
func (b *TextBuffer) SetText(s string) {
	b.Rope = NewRope(s)
}

// Insert inserts s at off.
func (b *TextBuffer) Insert(off int, s string) { b.Rope.Insert(off, s) }

// Delete removes [start, end).
func (b *TextBuffer) Delete(start, end int) { b.Rope.Delete(start, end) }

// Replace replaces [start, end) with s.
func (b *TextBuffer) Replace(start, end int, s string) { b.Rope.Replace(start, end, s) }

// Slice returns the text in [start, end).
func (b *TextBuffer) Slice(start, end int) string { return b.Rope.Slice(start, end) }

// LineCount returns the number of lines.
func (b *TextBuffer) LineCount() int { return b.Rope.LineCount() }

// Line returns line i without its newline.
func (b *TextBuffer) Line(i int) string { return b.Rope.Line(i) }

// LineStart returns the byte offset of the start of line i.
func (b *TextBuffer) LineStart(i int) int { return b.Rope.LineStart(i) }

// LineEnd returns the byte offset just past line i's content.
func (b *TextBuffer) LineEnd(i int) int { return b.Rope.LineEnd(i) }

// LineWidth returns the display width of line i after tab expansion.
func (b *TextBuffer) LineWidth(i int) int {
	return width.StringWidth(b.ExpandedLine(i))
}

// ExpandedLine returns line i with tabs expanded to spaces.
func (b *TextBuffer) ExpandedLine(i int) string {
	return ExpandTabs(b.Line(i), b.tabWidth(), 0)
}

// Position converts an offset into line and byte column.
func (b *TextBuffer) Position(off int) (line, col int) { return b.Rope.Position(off) }

// Offset converts line and byte column into an offset.
func (b *TextBuffer) Offset(line, col int) int { return b.Rope.Offset(line, col) }

func (b *TextBuffer) tabWidth() int {
	if b.TabWidth <= 0 {
		return 4
	}
	return b.TabWidth
}

// PrevGrapheme returns the offset of the grapheme cluster before off.
func (b *TextBuffer) PrevGrapheme(off int) int {
	if off <= 0 {
		return 0
	}
	off = clampInt(off, 0, b.Len())
	line, col := b.Position(off)
	if col == 0 {
		if line == 0 {
			return 0
		}
		return off - 1 // the newline before this line
	}
	lineText := b.Line(line)
	prev := 0
	found := false
	width.IteratePos(lineText, func(c width.Cluster) bool {
		if c.End >= col {
			prev = c.Start
			found = true
			return false
		}
		return true
	})
	if !found {
		prev = len(lineText)
	}
	return b.LineStart(line) + prev
}

// NextGrapheme returns the offset of the grapheme cluster after off.
func (b *TextBuffer) NextGrapheme(off int) int {
	off = clampInt(off, 0, b.Len())
	line, col := b.Position(off)
	lineText := b.Line(line)
	if col >= len(lineText) {
		if line >= b.LineCount()-1 {
			return b.Len()
		}
		return off + 1 // past the newline onto the next line
	}
	next := len(lineText)
	width.IteratePos(lineText, func(c width.Cluster) bool {
		if c.Start >= col {
			next = c.End
			return false
		}
		return true
	})
	if next == col {
		next = len(lineText)
	}
	return b.LineStart(line) + next
}

// WordBoundaryLeft returns the offset at the start of the previous word.
func (b *TextBuffer) WordBoundaryLeft(off int) int {
	off = clampInt(off, 0, b.Len())
	line, col := b.Position(off)
	text := b.Line(line)
	for col > 0 && isSpace(text[col-1]) {
		col--
	}
	for col > 0 && !isSpace(text[col-1]) {
		col--
	}
	return b.LineStart(line) + col
}

// WordBoundaryRight returns the offset at the end of the next word.
func (b *TextBuffer) WordBoundaryRight(off int) int {
	off = clampInt(off, 0, b.Len())
	line, col := b.Position(off)
	text := b.Line(line)
	for col < len(text) && isSpace(text[col]) {
		col++
	}
	for col < len(text) && !isSpace(text[col]) {
		col++
	}
	return b.LineStart(line) + col
}

// ClampOffset moves off back to the nearest UTF-8 boundary.
func (b *TextBuffer) ClampOffset(off int) int {
	return RuneStart(b.Text(), off)
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t'
}

// ExpandTabs converts tabs to spaces using a fixed tab width and starting
// column.
func ExpandTabs(s string, tabWidth, startCol int) string {
	if tabWidth <= 0 {
		tabWidth = 4
	}
	if !strings.ContainsRune(s, '\t') {
		return s
	}
	var out strings.Builder
	out.Grow(len(s))
	col := startCol
	for _, r := range s {
		if r == '\t' {
			n := tabWidth - col%tabWidth
			out.WriteString(strings.Repeat(" ", n))
			col += n
			continue
		}
		out.WriteRune(r)
		col += width.RuneWidth(r)
	}
	return out.String()
}

// DisplayClusters returns the grapheme clusters of s with tabs expanded. The
// byte offsets refer to s, so callers can map display cells back to the
// underlying text.
func DisplayClusters(s string, tabWidth int) []DisplayCluster {
	if tabWidth <= 0 {
		tabWidth = 4
	}
	var out []DisplayCluster
	col := 0
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		from, to := g.Positions()
		if g.Str() == "\t" {
			n := tabWidth - col%tabWidth
			out = append(out, DisplayCluster{Text: strings.Repeat(" ", n), Width: n, ByteStart: from, ByteEnd: to})
			col += n
			continue
		}
		out = append(out, DisplayCluster{Text: g.Str(), Width: g.Width(), ByteStart: from, ByteEnd: to})
		col += g.Width()
	}
	return out
}

// DisplayCluster is a grapheme or expanded tab mapped back to source bytes.
type DisplayCluster struct {
	Text      string
	Width     int
	ByteStart int
	ByteEnd   int
}
