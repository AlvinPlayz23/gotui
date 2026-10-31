package renderables

import (
	"strings"

	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/input"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
	"github.com/bijim/gotui/width"
)

// InputOptions configures an Input.
type InputOptions struct {
	ID               string
	Value            string
	Placeholder      string
	FG               color.RGBA
	BG               color.RGBA
	CursorFG         color.RGBA
	CursorBG         color.RGBA
	PlaceholderColor color.RGBA
	Password         bool
	MaxLength        int
	Style            layout.Style
	OnChange         func(string)
	OnSubmit         func(string)
}

// Input is a single-line text editor.
type Input struct {
	*renderable.Base
	Value            []rune
	Cursor           int
	Placeholder      string
	FG               color.RGBA
	BG               color.RGBA
	CursorFG         color.RGBA
	CursorBG         color.RGBA
	PlaceholderColor color.RGBA
	Password         bool
	MaxLength        int
	OnChange         func(string)
	OnSubmit         func(string)

	scroll int
}

// NewInput creates an input.
func NewInput(opts InputOptions) *Input {
	if opts.Style == (layout.Style{}) {
		opts.Style = layout.DefaultStyle()
	}
	in := &Input{
		Value:            []rune(opts.Value),
		Placeholder:      opts.Placeholder,
		FG:               opts.FG,
		BG:               opts.BG,
		CursorFG:         opts.CursorFG,
		CursorBG:         opts.CursorBG,
		PlaceholderColor: opts.PlaceholderColor,
		Password:         opts.Password,
		MaxLength:        opts.MaxLength,
		OnChange:         opts.OnChange,
		OnSubmit:         opts.OnSubmit,
	}
	if in.FG == (color.RGBA{}) {
		in.FG = White
	}
	if in.CursorFG == (color.RGBA{}) {
		in.CursorFG = Black
	}
	if in.CursorBG == (color.RGBA{}) {
		in.CursorBG = White
	}
	if in.PlaceholderColor == (color.RGBA{}) {
		in.PlaceholderColor = Gray
	}
	if !opts.Style.Height.IsDefinite() {
		opts.Style.Height = layout.P(1)
	}
	opts.Style.MinHeight = layout.P(1)
	in.Base = renderable.NewBase(in, &layout.Node{Style: opts.Style})
	in.Base.Focusable = true
	in.Base.OnClick = func(*input.MouseEvent) { in.RequestRender() }
	in.Cursor = len(in.Value)
	return in
}

// String returns the current value.
func (in *Input) String() string { return string(in.Value) }

// SetValue replaces the value and moves the cursor to the end.
func (in *Input) SetValue(s string) {
	in.Value = []rune(s)
	in.Cursor = len(in.Value)
	tail := in.scrollToCursor()
	_ = tail
	in.Invalidate()
	in.RequestRender()
}

func (in *Input) notify() {
	if in.OnChange != nil {
		in.OnChange(string(in.Value))
	}
}

func (in *Input) insert(s []rune) {
	if in.MaxLength > 0 && len(in.Value)+len(s) > in.MaxLength {
		s = s[:maxInt(0, in.MaxLength-len(in.Value))]
	}
	if len(s) == 0 {
		return
	}
	v := make([]rune, 0, len(in.Value)+len(s))
	v = append(v, in.Value[:in.Cursor]...)
	v = append(v, s...)
	v = append(v, in.Value[in.Cursor:]...)
	in.Value = v
	in.Cursor += len(s)
	in.afterEdit()
}

func (in *Input) afterEdit() {
	in.scrollToCursor()
	in.notify()
	in.Invalidate()
	in.RequestRender()
}

func (in *Input) deleteRange(from, to int) bool {
	if from < 0 {
		from = 0
	}
	if to > len(in.Value) {
		to = len(in.Value)
	}
	if from >= to {
		return false
	}
	in.Value = append(in.Value[:from], in.Value[to:]...)
	in.Cursor = from
	in.afterEdit()
	return true
}

// HandleKey implements keyboard editing.
func (in *Input) HandleKey(ev *input.KeyEvent) bool {
	if ev.Type == input.KeyRelease {
		return false
	}
	if in.OnKeyDown != nil && in.OnKeyDown(ev) {
		return true
	}
	switch ev.Name {
	case "left":
		if in.Cursor > 0 {
			in.Cursor--
		}
		in.scrollToCursor()
		in.RequestRender()
		return true
	case "right":
		if in.Cursor < len(in.Value) {
			in.Cursor++
		}
		in.scrollToCursor()
		in.RequestRender()
		return true
	case "home":
		in.Cursor = 0
		in.scrollToCursor()
		in.RequestRender()
		return true
	case "end":
		in.Cursor = len(in.Value)
		in.scrollToCursor()
		in.RequestRender()
		return true
	case "backspace":
		if ev.Ctrl {
			from := wordStart(in.Value, in.Cursor)
			return in.deleteRange(from, in.Cursor)
		}
		return in.deleteRange(in.Cursor-1, in.Cursor)
	case "delete":
		if ev.Ctrl {
			to := wordEnd(in.Value, in.Cursor)
			return in.deleteRange(in.Cursor, to)
		}
		return in.deleteRange(in.Cursor, in.Cursor+1)
	case "return", "enter":
		if in.OnSubmit != nil {
			in.OnSubmit(string(in.Value))
		}
		return true
	}
	if ev.Ctrl {
		switch ev.Name {
		case "a":
			in.Cursor = 0
		case "e":
			in.Cursor = len(in.Value)
		case "u":
			return in.deleteRange(0, in.Cursor)
		case "k":
			return in.deleteRange(in.Cursor, len(in.Value))
		case "w":
			return in.deleteRange(wordStart(in.Value, in.Cursor), in.Cursor)
		default:
			return false
		}
		in.scrollToCursor()
		in.RequestRender()
		return true
	}
	if ev.IsText() {
		in.insert([]rune{ev.Rune})
		return true
	}
	return false
}

// HandlePaste inserts pasted text.
func (in *Input) HandlePaste(ev *input.PasteEvent) bool {
	if in.OnPaste != nil && in.OnPaste(ev) {
		return true
	}
	s := strings.ReplaceAll(strings.ReplaceAll(ev.Text, "\r\n", " "), "\n", " ")
	in.insert([]rune(s))
	return true
}

func wordStart(v []rune, i int) int {
	for i > 0 && v[i-1] == ' ' {
		i--
	}
	for i > 0 && v[i-1] != ' ' {
		i--
	}
	return i
}

func wordEnd(v []rune, i int) int {
	for i < len(v) && v[i] == ' ' {
		i++
	}
	for i < len(v) && v[i] != ' ' {
		i++
	}
	return i
}

func (in *Input) displayValue() []rune {
	if !in.Password {
		return in.Value
	}
	out := make([]rune, len(in.Value))
	for i := range out {
		out[i] = '•'
	}
	return out
}

// scrollToCursor adjusts the horizontal scroll offset.
func (in *Input) scrollToCursor() int {
	w := in.ContentBounds().W
	if w <= 0 {
		return in.scroll
	}
	display := in.displayValue()
	before := width.StringWidth(string(display[:in.Cursor]))
	at := before
	if in.Cursor < len(display) {
		at += width.StringWidth(string(display[in.Cursor]))
	}
	if before < in.scroll {
		in.scroll = before
	}
	if at > in.scroll+w {
		in.scroll = at - w
	}
	if in.scroll < 0 {
		in.scroll = 0
	}
	return in.scroll
}

// Paint draws the input.
func (in *Input) Paint(ctx *renderable.Context) {
	if !in.Visible {
		return
	}
	r := in.Bounds()
	if in.BG.A > 0 {
		ctx.Fill(r.X, r.Y, r.W, r.H, in.BG)
	}
	content := in.ContentBounds()
	if content.W <= 0 || content.H <= 0 {
		return
	}
	display := in.displayValue()
	y := content.Y
	if len(display) == 0 && in.Placeholder != "" {
		ctx.SetString(content.X, y, width.Truncate(in.Placeholder, content.W), in.PlaceholderColor, in.BG, 0)
	}
	x := 0
	cursorCell := -1
	for i, ch := range display {
		w := width.RuneWidth(ch)
		if i == in.Cursor {
			cursorCell = x
		}
		sx := content.X + x - in.scroll
		if sx+w > content.X && sx < content.Right() {
			ctx.SetString(sx, y, string(ch), in.FG, in.BG, 0)
		}
		x += w
	}
	if in.Cursor >= len(display) {
		cursorCell = x
	}
	tailWidth := 0
	if in.Cursor < len(display) {
		tailWidth = width.RuneWidth(display[in.Cursor])
	}
	if in.Focused && cursorCell >= 0 {
		cx := content.X + cursorCell - in.scroll
		if cx >= content.X && cx < content.Right() {
			ch := " "
			if in.Cursor < len(display) {
				ch = string(display[in.Cursor])
			}
			ctx.SetString(cx, y, ch, in.CursorFG, in.CursorBG, 0)
			ctx.Renderer.SetCursor(cx, y, true)
			_ = tailWidth
		}
	}
}
