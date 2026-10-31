package text

// VisualLine is one laid-out line of a TextView. Clusters are tab-expanded.
type VisualLine struct {
	Line      int
	Clusters  []DisplayCluster
	Width     int
	StartByte int // first byte column included in this visual line
	EndByte   int // one past the last byte column included
}

// String returns the tab-expanded line text.
func (v VisualLine) String() string {
	var out []byte
	for _, c := range v.Clusters {
		out = append(out, c.Text...)
	}
	return string(out)
}

// TextView lays a TextBuffer out into wrapped visual lines and owns the
// viewport scroll offsets. It does not store styled spans; callers paint the
// returned lines with their own styles.
type TextView struct {
	Buffer   *TextBuffer
	Wrap     WrapMode
	TabWidth int

	Width, Height int
	ScrollX       int
	ScrollY       int

	lines    []VisualLine
	layoutW  int
	layoutOK bool
}

// NewTextView creates a text view over buf.
func NewTextView(buf *TextBuffer) *TextView {
	if buf == nil {
		buf = NewTextBuffer("")
	}
	return &TextView{Buffer: buf, TabWidth: 4}
}

// Resize sets the viewport size and clamps the scroll offsets.
func (v *TextView) Resize(w, h int) {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	if v.Width == w && v.Height == h {
		return
	}
	v.Width, v.Height = w, h
	if w != v.layoutW {
		v.layoutOK = false
	}
	v.clampScroll()
}

// Invalidate marks the layout cache as stale.
func (v *TextView) Invalidate() { v.layoutOK = false }

// SetWrap changes the wrap mode.
func (v *TextView) SetWrap(mode WrapMode) {
	if v.Wrap == mode {
		return
	}
	v.Wrap = mode
	v.layoutOK = false
	v.clampScroll()
}

// VisualLines returns the laid-out lines.
func (v *TextView) VisualLines() []VisualLine {
	v.ensureLayout()
	return v.lines
}

// LineCount returns the number of visual lines.
func (v *TextView) LineCount() int {
	v.ensureLayout()
	return len(v.lines)
}

// MaxScrollY returns the largest vertical scroll offset.
func (v *TextView) MaxScrollY() int {
	v.ensureLayout()
	return maxInt(0, len(v.lines)-v.Height)
}

// MaxScrollX returns the largest horizontal scroll offset.
func (v *TextView) MaxScrollX() int {
	v.ensureLayout()
	maxW := 0
	for _, l := range v.lines {
		if l.Width > maxW {
			maxW = l.Width
		}
	}
	return maxInt(0, maxW-v.Width)
}

// ScrollBy scrolls by the given deltas and clamps.
func (v *TextView) ScrollBy(dx, dy int) {
	v.ensureLayout()
	v.ScrollX = clampInt(v.ScrollX+dx, 0, v.MaxScrollX())
	v.ScrollY = clampInt(v.ScrollY+dy, 0, v.MaxScrollY())
}

// ScrollTo sets the scroll offsets and clamps them.
func (v *TextView) ScrollTo(x, y int) {
	v.ensureLayout()
	v.ScrollX = clampInt(x, 0, v.MaxScrollX())
	v.ScrollY = clampInt(y, 0, v.MaxScrollY())
}

func (v *TextView) clampScroll() {
	if v.ScrollY < 0 {
		v.ScrollY = 0
	}
	if v.ScrollX < 0 {
		v.ScrollX = 0
	}
	if v.layoutOK {
		v.ScrollY = clampInt(v.ScrollY, 0, v.MaxScrollY())
		v.ScrollX = clampInt(v.ScrollX, 0, v.MaxScrollX())
	}
}

// VisibleLines returns the visual lines intersecting the viewport.
func (v *TextView) VisibleLines() []VisualLine {
	v.ensureLayout()
	if v.Height <= 0 {
		return nil
	}
	end := minInt(len(v.lines), v.ScrollY+v.Height)
	if v.ScrollY >= end {
		return nil
	}
	return v.lines[v.ScrollY:end]
}

// VisualIndex returns the index of the visual line containing logical line
// and byte column. It prefers the last matching line at a wrap boundary.
func (v *TextView) VisualIndex(line, byteCol int) (index, cellX int) {
	v.ensureLayout()
	found := -1
	for i := range v.lines {
		l := &v.lines[i]
		if l.Line != line {
			continue
		}
		if byteCol < l.StartByte || byteCol > l.EndByte {
			continue
		}
		found = i
		if byteCol < l.EndByte || l.StartByte == l.EndByte {
			break
		}
	}
	if found < 0 {
		// Offset past the end of the line (or empty buffer).
		for i := len(v.lines) - 1; i >= 0; i-- {
			if v.lines[i].Line == line {
				found = i
				break
			}
		}
	}
	if found < 0 {
		return 0, 0
	}
	return found, v.cellInLine(&v.lines[found], byteCol)
}

func (v *TextView) cellInLine(l *VisualLine, byteCol int) int {
	x := 0
	for _, c := range l.Clusters {
		if c.ByteStart >= byteCol {
			break
		}
		x += c.Width
	}
	return x
}

// HitTest maps viewport coordinates to a logical line and byte column.
func (v *TextView) HitTest(x, y int) (line, byteCol int) {
	v.ensureLayout()
	if len(v.lines) == 0 {
		return 0, 0
	}
	idx := clampInt(v.ScrollY+y, 0, len(v.lines)-1)
	l := &v.lines[idx]
	target := x + v.ScrollX
	col := 0
	byteCol = l.StartByte
	for _, c := range l.Clusters {
		if target < col+c.Width {
			// Snap to the nearest cluster edge.
			if target-col >= (c.Width+1)/2 {
				byteCol = c.ByteEnd
			} else {
				byteCol = c.ByteStart
			}
			return l.Line, byteCol
		}
		col += c.Width
		byteCol = c.ByteEnd
	}
	return l.Line, l.EndByte
}

// EnsureVisible scrolls so that line and byteCol are inside the viewport,
// keeping margin cells around the cursor.
func (v *TextView) EnsureVisible(line, byteCol, margin int) {
	v.ensureLayout()
	idx, x := v.VisualIndex(line, byteCol)
	if v.Height > 0 {
		top := idx - margin
		bottom := idx + margin
		if top < v.ScrollY {
			v.ScrollY = maxInt(0, top)
		}
		if bottom >= v.ScrollY+v.Height {
			v.ScrollY = bottom - v.Height + 1
		}
	}
	if v.Width > 0 {
		if x < v.ScrollX {
			v.ScrollX = x
		}
		if x >= v.ScrollX+v.Width {
			v.ScrollX = x - v.Width + 1
		}
	}
	v.clampScroll()
}

// SelectionCells returns per-visual-line cell ranges for the byte range
// [start, end). Ranges are relative to the visual line start.
type SelectionCell struct {
	Line       int // visual line index
	Start, End int // cell columns in the visual line
}

// SelectionCells computes selection rectangles for a byte range.
func (v *TextView) SelectionCells(start, end int) []SelectionCell {
	start, end = clampRange(start, end, v.Buffer.Len())
	if start == end {
		return nil
	}
	v.ensureLayout()
	var out []SelectionCell
	// Positions are logical.
	startLine, startCol := v.Buffer.Position(start)
	endLine, endCol := v.Buffer.Position(end)
	for i := range v.lines {
		l := &v.lines[i]
		if l.Line < startLine || l.Line > endLine {
			continue
		}
		from := l.StartByte
		if l.Line == startLine && startCol > from {
			from = startCol
		}
		to := l.EndByte
		if l.Line == endLine && endCol < to {
			to = endCol
		}
		if from > to {
			continue
		}
		x0 := v.cellInLine(l, from)
		x1 := v.cellInLine(l, to)
		if l.Line != endLine && endLine > l.Line {
			x1 = l.Width
		}
		if l.Line == endLine && endCol >= l.EndByte {
			x1 = l.Width
		}
		if x1 > x0 {
			out = append(out, SelectionCell{Line: i, Start: x0, End: x1})
		}
	}
	return out
}

func (v *TextView) ensureLayout() {
	w := v.Width
	if w < 0 {
		w = 0
	}
	if v.layoutOK && v.layoutW == w {
		return
	}
	v.layoutW = w
	v.layoutOK = true
	v.lines = v.lines[:0]
	buf := v.Buffer
	tab := v.TabWidth
	if tab <= 0 {
		tab = 4
	}
	for li := 0; li < buf.LineCount(); li++ {
		text := buf.Line(li)
		clusters := DisplayClusters(text, tab)
		if v.Wrap == NoWrap || w <= 0 {
			v.lines = append(v.lines, makeVisualLine(li, clusters))
			continue
		}
		v.wrapLine(li, clusters, w)
	}
	if len(v.lines) == 0 {
		v.lines = append(v.lines, VisualLine{})
	}
	v.ScrollY = clampInt(v.ScrollY, 0, maxInt(0, len(v.lines)-v.Height))
}

func makeVisualLine(line int, clusters []DisplayCluster) VisualLine {
	l := VisualLine{Line: line, Clusters: clusters}
	if len(clusters) > 0 {
		l.StartByte = clusters[0].ByteStart
		l.EndByte = clusters[len(clusters)-1].ByteEnd
	}
	for _, c := range clusters {
		l.Width += c.Width
	}
	return l
}

func (v *TextView) wrapLine(line int, clusters []DisplayCluster, maxW int) {
	if len(clusters) == 0 {
		v.lines = append(v.lines, VisualLine{Line: line})
		return
	}
	cur := make([]DisplayCluster, 0, len(clusters))
	curW := 0
	lastSpace := -1
	flush := func() {
		v.lines = append(v.lines, makeVisualLine(line, append([]DisplayCluster(nil), cur...)))
		cur = cur[:0]
		curW = 0
		lastSpace = -1
	}
	for _, c := range clusters {
		if curW > 0 && curW+c.Width > maxW {
			if v.Wrap == WrapWord && lastSpace >= 0 && lastSpace < len(cur)-1 {
				tail := append([]DisplayCluster(nil), cur[lastSpace+1:]...)
				cur = cur[:lastSpace+1]
				flush()
				for _, t := range tail {
					cur = append(cur, t)
					curW += t.Width
					if t.Text == " " {
						lastSpace = len(cur) - 1
					}
				}
			} else {
				flush()
			}
		}
		if c.Text == " " {
			lastSpace = len(cur)
		}
		cur = append(cur, c)
		curW += c.Width
	}
	if len(cur) > 0 {
		flush()
	}
}
