package text

import (
	"sort"
	"strings"
	"unicode/utf8"
)

// piece is a span of text. Pieces point either at the immutable original text
// or at the append-only buffer of inserted text, so editing never copies the
// whole document.
type piece struct {
	added  bool
	start  int
	length int
}

// Rope is a UTF-8 text buffer backed by a piece table. Offsets are byte
// offsets. Edits split pieces at the edit boundaries; reads concatenate the
// pieces. The line index is rebuilt lazily.
type Rope struct {
	original string
	added    []byte
	pieces   []piece
	length   int

	lineStarts []int
	lineValid  bool
}

// NewRope creates a rope containing s.
func NewRope(s string) *Rope {
	r := &Rope{original: s}
	if len(s) > 0 {
		r.pieces = []piece{{start: 0, length: len(s)}}
		r.length = len(s)
	}
	return r
}

// Len returns the length in bytes.
func (r *Rope) Len() int { return r.length }

// String returns the full content.
func (r *Rope) String() string {
	var b strings.Builder
	b.Grow(r.length)
	for _, p := range r.pieces {
		if p.length == 0 {
			continue
		}
		if p.added {
			b.Write(r.added[p.start : p.start+p.length])
		} else {
			b.WriteString(r.original[p.start : p.start+p.length])
		}
	}
	return b.String()
}

// Slice returns the text in [start, end).
func (r *Rope) Slice(start, end int) string {
	start, end = clampRange(start, end, r.length)
	if start == end {
		return ""
	}
	var b strings.Builder
	b.Grow(end - start)
	pos := 0
	for _, p := range r.pieces {
		pEnd := pos + p.length
		if pEnd <= start {
			pos = pEnd
			continue
		}
		if pos >= end {
			break
		}
		from := maxInt(start, pos) - pos
		to := minInt(end, pEnd) - pos
		if p.added {
			b.Write(r.added[p.start+from : p.start+to])
		} else {
			b.WriteString(r.original[p.start+from : p.start+to])
		}
		pos = pEnd
	}
	return b.String()
}

// Insert adds s at byte offset off.
func (r *Rope) Insert(off int, s string) {
	if s == "" {
		return
	}
	off = clampInt(off, 0, r.length)
	idx := r.split(off)
	addStart := len(r.added)
	r.added = append(r.added, s...)
	r.pieces = append(r.pieces, piece{})
	copy(r.pieces[idx+1:], r.pieces[idx:])
	r.pieces[idx] = piece{added: true, start: addStart, length: len(s)}
	r.length += len(s)
	r.lineValid = false
}

// Delete removes the text in [start, end).
func (r *Rope) Delete(start, end int) {
	start, end = clampRange(start, end, r.length)
	if start == end {
		return
	}
	i1 := r.split(start)
	i2 := r.split(end)
	r.pieces = append(r.pieces[:i1], r.pieces[i2:]...)
	r.length -= end - start
	r.lineValid = false
}

// Replace replaces [start, end) with s.
func (r *Rope) Replace(start, end int, s string) {
	start, end = clampRange(start, end, r.length)
	if start == end {
		r.Insert(start, s)
		return
	}
	r.Delete(start, end)
	r.Insert(start, s)
}

// split makes sure there is a piece boundary at off and returns the number of
// pieces before it.
func (r *Rope) split(off int) int {
	if off <= 0 {
		return 0
	}
	if off >= r.length {
		return len(r.pieces)
	}
	pos := 0
	for i, p := range r.pieces {
		if off == pos {
			return i
		}
		if off < pos+p.length {
			left := off - pos
			r.pieces = append(r.pieces, piece{})
			copy(r.pieces[i+2:], r.pieces[i+1:])
			r.pieces[i+1] = piece{added: p.added, start: p.start + left, length: p.length - left}
			r.pieces[i].length = left
			return i + 1
		}
		pos += p.length
	}
	return len(r.pieces)
}

func (r *Rope) buildLines() {
	if r.lineValid {
		return
	}
	starts := []int{0}
	pos := 0
	for _, p := range r.pieces {
		var text []byte
		if p.added {
			text = r.added[p.start : p.start+p.length]
		} else {
			text = []byte(r.original[p.start : p.start+p.length])
		}
		for i := 0; i < len(text); i++ {
			if text[i] == '\n' {
				starts = append(starts, pos+i+1)
			}
		}
		pos += p.length
	}
	r.lineStarts = starts
	r.lineValid = true
}

// LineCount returns the number of lines. A trailing newline yields a final
// empty line.
func (r *Rope) LineCount() int {
	r.buildLines()
	return len(r.lineStarts)
}

// LineStart returns the byte offset of the start of line i.
func (r *Rope) LineStart(i int) int {
	r.buildLines()
	if i <= 0 {
		return 0
	}
	if i >= len(r.lineStarts) {
		return r.length
	}
	return r.lineStarts[i]
}

// LineEnd returns the byte offset just past the content of line i, excluding
// its newline.
func (r *Rope) LineEnd(i int) int {
	r.buildLines()
	if i < 0 || i >= len(r.lineStarts) {
		return r.length
	}
	end := r.length
	if i+1 < len(r.lineStarts) {
		end = r.lineStarts[i+1] - 1
	}
	return end
}

// Line returns the text of line i without its newline.
func (r *Rope) Line(i int) string {
	return r.Slice(r.LineStart(i), r.LineEnd(i))
}

// Position converts a byte offset into a zero-based line and byte column.
func (r *Rope) Position(off int) (line, col int) {
	off = clampInt(off, 0, r.length)
	r.buildLines()
	i := sort.Search(len(r.lineStarts), func(i int) bool { return r.lineStarts[i] > off }) - 1
	if i < 0 {
		i = 0
	}
	return i, off - r.lineStarts[i]
}

// Offset converts a zero-based line and byte column into an offset, clamped to
// the line content.
func (r *Rope) Offset(line, col int) int {
	r.buildLines()
	if line < 0 {
		return 0
	}
	if line >= len(r.lineStarts) {
		return r.length
	}
	col = clampInt(col, 0, r.LineEnd(line)-r.LineStart(line))
	return r.lineStarts[line] + col
}

// RuneStart returns the start of the UTF-8 rune containing off.
func RuneStart(s string, off int) int {
	off = clampInt(off, 0, len(s))
	for off > 0 && off < len(s) && !utf8.RuneStart(s[off]) {
		off--
	}
	return off
}

func clampRange(start, end, length int) (int, int) {
	start = clampInt(start, 0, length)
	end = clampInt(end, 0, length)
	if start > end {
		start, end = end, start
	}
	return start, end
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
