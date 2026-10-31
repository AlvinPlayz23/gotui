package renderables

import (
	"strings"

	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/input"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
	"github.com/bijim/gotui/width"
)

// TextareaOptions configures a Textarea.
type TextareaOptions struct {
	ID               string
	Value            string
	Placeholder      string
	FG               color.RGBA
	BG               color.RGBA
	CursorFG         color.RGBA
	CursorBG         color.RGBA
	PlaceholderColor color.RGBA
	Style            layout.Style
	OnChange         func(string)
	OnSubmit         func(string)
}

// Textarea is a multi-line text editor.
type Textarea struct {
	*renderable.Base
	Value            []rune
	Cursor           int
	Placeholder      string
	FG               color.RGBA
	BG               color.RGBA
	CursorFG         color.RGBA
	CursorBG         color.RGBA
	PlaceholderColor color.RGBA
	OnChange         func(string)
	OnSubmit         func(string)

	scrollY int
	scrollX int
}

// NewTextarea creates a textarea.
func NewTextarea(opts TextareaOptions) *Textarea {
	if opts.Style == (layout.Style{}) {
		opts.Style = layout.DefaultStyle()
	}
	ta := &Textarea{
		Value:            []rune(opts.Value),
		Placeholder:      opts.Placeholder,
		FG:               opts.FG,
		BG:               opts.BG,
		CursorFG:         opts.CursorFG,
		CursorBG:         opts.CursorBG,
		PlaceholderColor: opts.PlaceholderColor,
		OnChange:         opts.OnChange,
		OnSubmit:         opts.OnSubmit,
	}
	if ta.FG == (color.RGBA{}) {
		ta.FG = White
	}
	if ta.CursorFG == (color.RGBA{}) {
		ta.CursorFG = Black
	}
	if ta.CursorBG == (color.RGBA{}) {
		ta.CursorBG = White
	}
	if ta.PlaceholderColor == (color.RGBA{}) {
		ta.PlaceholderColor = Gray
	}
	opts.Style.MinHeight = layout.P(1)
	ta.Base = renderable.NewBase(ta, &layout.Node{Style: opts.Style})
	ta.Base.Focusable = true
	ta.Base.Node.Measure = ta.measure
	ta.Cursor = len(ta.Value)
	return ta
}

// String returns the value.
func (ta *Textarea) String() string { return string(ta.Value) }

// SetValue replaces the content.
func (ta *Textarea) SetValue(s string) {
	ta.Value = []rune(s)
	ta.Cursor = len(ta.Value)
	ta.scrollToCursor()
	ta.Invalidate()
	ta.RequestRender()
}

func (ta *Textarea) measure(availW, availH int) (int, int) {
	lines := ta.lines()
	w := 0
	for _, l := range lines {
		w = maxInt(w, width.StringWidth(string(l)))
	}
	return w, len(lines)
}

func (ta *Textarea) lines() [][]rune {
	return splitRunes(ta.Value)
}

func splitRunes(v []rune) [][]rune {
	lines := [][]rune{{}}
	for _, r := range v {
		if r == '\n' {
			lines = append(lines, []rune{})
			continue
		}
		lines[len(lines)-1] = append(lines[len(lines)-1], r)
	}
	return lines
}

func (ta *Textarea) rowCol() (row, col int) {
	for i := 0; i < ta.Cursor; i++ {
		if ta.Value[i] == '\n' {
			row++
			col = 0
		} else {
			col++
		}
	}
	return
}

func (ta *Textarea) offset(row, col int) int {
	lines := ta.lines()
	if row >= len(lines) {
		return len(ta.Value)
	}
	off := 0
	for i := 0; i < row; i++ {
		off += len(lines[i]) + 1
	}
	off += minInt(col, len(lines[row]))
	return off
}

func (ta *Textarea) afterEdit() {
	ta.scrollToCursor()
	if ta.OnChange != nil {
		ta.OnChange(string(ta.Value))
	}
	ta.Invalidate()
	ta.RequestRender()
}

func (ta *Textarea) insert(s []rune) {
	v := make([]rune, 0, len(ta.Value)+len(s))
	v = append(v, ta.Value[:ta.Cursor]...)
	v = append(v, s...)
	v = append(v, ta.Value[ta.Cursor:]...)
	ta.Value = v
	ta.Cursor += len(s)
	ta.afterEdit()
}

func (ta *Textarea) deleteRange(from, to int) bool {
	if from < 0 {
		from = 0
	}
	if to > len(ta.Value) {
		to = len(ta.Value)
	}
	if from >= to {
		return false
	}
	ta.Value = append(ta.Value[:from], ta.Value[to:]...)
	ta.Cursor = from
	ta.afterEdit()
	return true
}

// HandleKey implements editing.
func (ta *Textarea) HandleKey(ev *input.KeyEvent) bool {
	if ev.Type == input.KeyRelease {
		return false
	}
	if ta.OnKeyDown != nil && ta.OnKeyDown(ev) {
		return true
	}
	row, col := ta.rowCol()
	lines := ta.lines()
	switch ev.Name {
	case "left":
		if ta.Cursor > 0 {
			ta.Cursor--
		}
	case "right":
		if ta.Cursor < len(ta.Value) {
			ta.Cursor++
		}
	case "up":
		if row > 0 {
			ta.Cursor = ta.offset(row-1, col)
		}
	case "down":
		if row < len(lines)-1 {
			ta.Cursor = ta.offset(row+1, col)
		}
	case "home":
		ta.Cursor = ta.offset(row, 0)
	case "end":
		ta.Cursor = ta.offset(row, len(lines[row]))
	case "pageup":
		rows := maxInt(1, ta.ContentBounds().H-1)
		ta.Cursor = ta.offset(maxInt(0, row-rows), col)
	case "pagedown":
		rows := maxInt(1, ta.ContentBounds().H-1)
		ta.Cursor = ta.offset(minInt(len(lines)-1, row+rows), col)
	case "backspace":
		return ta.deleteRange(ta.Cursor-1, ta.Cursor)
	case "delete":
		return ta.deleteRange(ta.Cursor, ta.Cursor+1)
	case "return", "enter":
		if ev.Ctrl {
			if ta.OnSubmit != nil {
				ta.OnSubmit(string(ta.Value))
			}
			return true
		}
		ta.insert([]rune{'\n'})
		return true
	case "tab":
		ta.insert([]rune{' ', ' '})
		return true
	default:
		if ev.Ctrl {
			switch ev.Name {
			case "u":
				return ta.deleteRange(ta.offset(row, 0), ta.Cursor)
			case "k":
				return ta.deleteRange(ta.Cursor, ta.offset(row+1, 0)-1)
			case "a":
				ta.Cursor = 0
			case "e":
				ta.Cursor = len(ta.Value)
			default:
				return false
			}
		} else if ev.IsText() {
			ta.insert([]rune{ev.Rune})
			return true
		} else {
			return false
		}
	}
	ta.scrollToCursor()
	ta.RequestRender()
	return true
}

// HandlePaste inserts pasted text.
func (ta *Textarea) HandlePaste(ev *input.PasteEvent) bool {
	if ta.OnPaste != nil && ta.OnPaste(ev) {
		return true
	}
	s := strings.ReplaceAll(ev.Text, "\r\n", "\n")
	ta.insert([]rune(s))
	return true
}

func (ta *Textarea) scrollToCursor() {
	content := ta.ContentBounds()
	row, col := ta.rowCol()
	lines := ta.lines()
	if content.H > 0 {
		if row < ta.scrollY {
			ta.scrollY = row
		}
		if row >= ta.scrollY+content.H {
			ta.scrollY = row - content.H + 1
		}
	}
	if content.W > 0 {
		cx := width.StringWidth(string(lines[row][:col]))
		if cx < ta.scrollX {
			ta.scrollX = cx
		}
		if cx >= ta.scrollX+content.W {
			ta.scrollX = cx - content.W + 1
		}
	}
}

// Paint draws the textarea.
func (ta *Textarea) Paint(ctx *renderable.Context) {
	if !ta.Visible {
		return
	}
	r := ta.Bounds()
	if ta.BG.A > 0 {
		ctx.Fill(r.X, r.Y, r.W, r.H, ta.BG)
	}
	content := ta.ContentBounds()
	if content.W <= 0 || content.H <= 0 {
		return
	}
	if len(ta.Value) == 0 && ta.Placeholder != "" {
		for i, line := range strings.Split(ta.Placeholder, "\n") {
			if i >= content.H {
				break
			}
			ctx.SetString(content.X, content.Y+i, width.Truncate(line, content.W), ta.PlaceholderColor, ta.BG, 0)
		}
	}
	lines := ta.lines()
	row, col := ta.rowCol()
	for i := 0; i < content.H; i++ {
		li := ta.scrollY + i
		if li >= len(lines) {
			break
		}
		line := string(lines[li])
		x := content.X - ta.scrollX
		ctx.SetString(x, content.Y+i, line, ta.FG, ta.BG, 0)
	}
	if ta.Focused {
		line := lines[row]
		col = minInt(col, len(line))
		cx := content.X + width.StringWidth(string(line[:col])) - ta.scrollX
		cy := content.Y + row - ta.scrollY
		if cx >= content.X && cx < content.Right() && cy >= content.Y && cy < content.Bottom() {
			ch := " "
			if col < len(line) {
				ch = string(line[col])
			}
			ctx.SetString(cx, cy, ch, ta.CursorFG, ta.CursorBG, 0)
			ctx.Renderer.SetCursor(cx, cy, true)
		}
	}
}
