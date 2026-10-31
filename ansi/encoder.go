package ansi

import (
	"io"
	"strconv"
	"strings"

	"github.com/bijim/gotui/cell"
	"github.com/bijim/gotui/color"
)

// ColorMode selects how colors are emitted.
type ColorMode int

const (
	ColorTrue ColorMode = iota
	Color256
	Color16
)

// EncoderOptions configures a screen Encoder.
type EncoderOptions struct {
	ColorMode     ColorMode
	Synchronized  bool
	ForceRerender bool
}

// Encoder writes a cell buffer to a terminal, diffing against the previous frame.
type Encoder struct {
	out       io.Writer
	opts      EncoderOptions
	prev      *cell.Buffer
	synced    bool
	lastStyle styleState
	styleOK   bool
}

type styleState struct {
	fg, bg color.RGBA
	attr   cell.Attribute
	link   uint32
}

// NewEncoder creates an encoder writing to out.
func NewEncoder(out io.Writer, opts EncoderOptions) *Encoder {
	return &Encoder{out: out, opts: opts}
}

// Invalidate forces the next Render to repaint every cell.
func (e *Encoder) Invalidate() { e.prev = nil }

// SetColorMode updates the color mode.
func (e *Encoder) SetColorMode(m ColorMode) { e.opts.ColorMode = m; e.prev = nil }

// Render diffs buf against the previous frame and writes the changed cells.
// The terminal is assumed to have the cursor somewhere; the encoder positions it.
func (e *Encoder) Render(buf *cell.Buffer) error {
	var sb strings.Builder
	sb.Grow(buf.W * buf.H * 4)
	if e.opts.Synchronized {
		sb.WriteString(BeginSynchronizedUpdate)
	}
	sb.WriteString(HideCursor)

	prev := e.prev
	if prev == nil || prev.W != buf.W || prev.H != buf.H {
		prev = nil
		e.styleOK = false
	}
	e.lastStyle = styleState{}
	e.styleOK = false

	for y := 0; y < buf.H; y++ {
		runs := changedRuns(prev, buf, y)
		for _, run := range runs {
			if run[1] < run[0] {
				continue
			}
			sb.WriteString(CursorTo(run[0]+1, y+1))
			for x := run[0]; x <= run[1] && x < buf.W; {
				c := buf.Cells[y*buf.W+x]
				if c.Width == 0 {
					x++
					continue
				}
				e.emitStyle(&sb, c)
				sb.WriteString(c.Glyph())
				if c.Width <= 1 {
					x++
				} else {
					x += int(c.Width)
				}
			}
		}
	}
	sb.WriteString(Reset)
	if e.opts.Synchronized {
		sb.WriteString(EndSynchronizedUpdate)
	}
	e.prev = buf.Clone()
	if _, err := io.WriteString(e.out, sb.String()); err != nil {
		return err
	}
	return nil
}

func changedRuns(prev, buf *cell.Buffer, y int) [][2]int {
	if prev == nil || prev.W != buf.W || prev.H != buf.H {
		return [][2]int{{0, buf.W - 1}}
	}
	var runs [][2]int
	start := -1
	for x := 0; x < buf.W; x++ {
		i := y*buf.W + x
		if !prev.Cells[i].Equal(buf.Cells[i]) {
			if start < 0 {
				start = x
			}
			continue
		}
		if start >= 0 {
			runs = append(runs, [2]int{start, x - 1})
			start = -1
		}
	}
	if start >= 0 {
		runs = append(runs, [2]int{start, buf.W - 1})
	}
	return runs
}

func (e *Encoder) emitStyle(sb *strings.Builder, c cell.Cell) {
	st := styleState{fg: c.FG, bg: c.BG, attr: c.Attr, link: c.Link}
	if e.styleOK && st == e.lastStyle {
		return
	}
	// Reset is the simplest correct baseline; keep it cheap by grouping runs.
	sb.WriteString(Reset)
	appendAttrs(sb, c.Attr)
	if c.FG.A > 0 {
		sb.WriteString(fgSeq(c.FG, e.opts.ColorMode))
	}
	if c.BG.A > 0 {
		sb.WriteString(bgSeq(c.BG, e.opts.ColorMode))
	}
	if c.Link != 0 {
		// OSC 8 hyperlinks are emitted by higher layers; id 0 is no link.
	}
	e.lastStyle = st
	e.styleOK = true
}

func appendAttrs(sb *strings.Builder, a cell.Attribute) {
	if a == 0 {
		return
	}
	if a.Has(cell.AttrBold) {
		sb.WriteString(Bold)
	}
	if a.Has(cell.AttrDim) {
		sb.WriteString(Dim)
	}
	if a.Has(cell.AttrItalic) {
		sb.WriteString(Italic)
	}
	if a.Has(cell.AttrUnderline) {
		sb.WriteString(Underline)
	}
	if a.Has(cell.AttrBlink) {
		sb.WriteString(Blink)
	}
	if a.Has(cell.AttrInverse) {
		sb.WriteString(Inverse)
	}
	if a.Has(cell.AttrHidden) {
		sb.WriteString(Hidden)
	}
	if a.Has(cell.AttrStrikethrough) {
		sb.WriteString(Strike)
	}
}

func fgSeq(c color.RGBA, mode ColorMode) string {
	r, g, b := color.C255(c.R), color.C255(c.G), color.C255(c.B)
	switch mode {
	case ColorTrue:
		return CSI + "38;2;" + itoa(int(r)) + ";" + itoa(int(g)) + ";" + itoa(int(b)) + "m"
	case Color256:
		return CSI + "38;5;" + itoa(rgbTo256(r, g, b)) + "m"
	default:
		return CSI + "3" + itoa(int(rgbTo16(r, g, b))) + "m"
	}
}

func bgSeq(c color.RGBA, mode ColorMode) string {
	r, g, b := color.C255(c.R), color.C255(c.G), color.C255(c.B)
	switch mode {
	case ColorTrue:
		return CSI + "48;2;" + itoa(int(r)) + ";" + itoa(int(g)) + ";" + itoa(int(b)) + "m"
	case Color256:
		return CSI + "48;5;" + itoa(rgbTo256(r, g, b)) + "m"
	default:
		return CSI + "4" + itoa(int(rgbTo16(r, g, b))) + "m"
	}
}

func itoa(v int) string { return strconv.Itoa(v) }

func rgbTo256(r, g, b uint8) int {
	// Grayscale ramp check.
	if abs(int(r)-int(g)) < 8 && abs(int(g)-int(b)) < 8 {
		v := (int(r) + int(g) + int(b)) / 3
		if v < 8 {
			return 16
		}
		if v > 248 {
			return 231
		}
		return 232 + (v-8)*24/247
	}
	to6 := func(v uint8) int { return int(float64(v) / 255 * 5) }
	ri, gi, bi := to6(r), to6(g), to6(b)
	return 16 + 36*ri + 6*gi + bi
}

func rgbTo16(r, g, b uint8) int {
	// Use the standard 16-color cube thresholding.
	idx := 0
	if r >= 128 {
		idx |= 1
	}
	if g >= 128 {
		idx |= 2
	}
	if b >= 128 {
		idx |= 4
	}
	// 1,2,4 map to red, green, blue; ANSI order is black, red, green, yellow, blue...
	m := map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 4: 4, 5: 5, 6: 6, 7: 7}
	if v, ok := m[idx]; ok {
		return v
	}
	return 7
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
