package renderables

import (
	"fmt"
	"strings"

	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/input"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
	"github.com/bijim/gotui/text"
	"github.com/bijim/gotui/width"
)

// EditorOptions configures an Editor.
type EditorOptions struct {
	ID               string
	Value            string
	Placeholder      string
	PlaceholderColor color.RGBA
	FG               color.RGBA
	BG               color.RGBA
	CursorFG         color.RGBA
	CursorBG         color.RGBA
	SelectionFG      color.RGBA
	SelectionBG      color.RGBA
	LineNumberFG     color.RGBA
	LineNumberBG     color.RGBA
	LineNumbers      bool
	Wrap             text.WrapMode
	TabWidth         int
	ScrollMargin     int
	Style            layout.Style
	OnChange         func(string)
}

// Editor is a multi-line text editor backed by the text engine.
type Editor struct {
	*renderable.Base
	Edit             *text.EditBuffer
	View             *text.EditorView
	Placeholder      string
	PlaceholderColor color.RGBA
	FG               color.RGBA
	BG               color.RGBA
	CursorFG         color.RGBA
	CursorBG         color.RGBA
	SelectionFG      color.RGBA
	SelectionBG      color.RGBA
	LineNumberFG     color.RGBA
	LineNumberBG     color.RGBA
	LineNumbers      bool
	OnChange         func(string)

	dragging  bool
	dragStart int
}

// NewEditor creates an editor.
func NewEditor(opts EditorOptions) *Editor {
	if opts.Style == (layout.Style{}) {
		opts.Style = layout.DefaultStyle()
	}
	if opts.Style.Direction == layout.Row {
		opts.Style.Direction = layout.Column
	}
	if !opts.Style.Width.IsDefinite() && opts.Style.Width.Unit != layout.Percent {
		opts.Style.Width = layout.Pct(100)
	}
	if opts.Style.MinHeight.Unit == layout.Undefined {
		opts.Style.MinHeight = layout.P(1)
	}
	e := &Editor{
		Placeholder:      opts.Placeholder,
		PlaceholderColor: opts.PlaceholderColor,
		FG:               opts.FG,
		BG:               opts.BG,
		CursorFG:         opts.CursorFG,
		CursorBG:         opts.CursorBG,
		SelectionFG:      opts.SelectionFG,
		SelectionBG:      opts.SelectionBG,
		LineNumberFG:     opts.LineNumberFG,
		LineNumberBG:     opts.LineNumberBG,
		LineNumbers:      opts.LineNumbers,
		OnChange:         opts.OnChange,
	}
	e.defaults()
	e.Base = renderable.NewBase(e, &layout.Node{Style: opts.Style})
	e.Base.Focusable = true
	e.Base.Node.Measure = e.measure

	eb := text.NewEditBufferText(opts.Value)
	eb.Text.TabWidth = opts.TabWidth
	e.Edit = eb
	e.View = text.NewEditorView(eb, 0, 0)
	e.View.View.Wrap = opts.Wrap
	e.View.ScrollMargin = opts.ScrollMargin
	eb.OnChange = func() {
		e.View.Invalidate()
		e.Invalidate()
		e.RequestRender()
		if e.OnChange != nil {
			e.OnChange(eb.Value())
		}
	}
	e.Base.OnClick = func(*input.MouseEvent) { e.RequestRender() }
	return e
}

func (e *Editor) defaults() {
	if e.FG == (color.RGBA{}) {
		e.FG = White
	}
	if e.CursorFG == (color.RGBA{}) {
		e.CursorFG = Black
	}
	if e.CursorBG == (color.RGBA{}) {
		e.CursorBG = White
	}
	if e.PlaceholderColor == (color.RGBA{}) {
		e.PlaceholderColor = Gray
	}
	if e.SelectionBG == (color.RGBA{}) {
		e.SelectionBG = color.FromInts(60, 60, 90, 255)
	}
	if e.SelectionFG == (color.RGBA{}) {
		e.SelectionFG = e.FG
	}
	if e.LineNumberFG == (color.RGBA{}) {
		e.LineNumberFG = Gray
	}
	if e.LineNumberBG == (color.RGBA{}) {
		e.LineNumberBG = e.BG
	}
}

// Value returns the editor text.
func (e *Editor) Value() string { return e.Edit.Value() }

// SetValue replaces the content.
func (e *Editor) SetValue(s string) { e.Edit.SetValue(s) }

// SetCursor moves the cursor to a byte offset.
func (e *Editor) SetCursor(off int) { e.Edit.SetCursor(off) }

// Focused reports whether the editor has focus.
func (e *Editor) IsFocused() bool { return e.Focused }

func (e *Editor) gutterWidth() int {
	if !e.LineNumbers {
		return 0
	}
	n := len(fmt.Sprintf("%d", e.Edit.Text.LineCount()))
	return n + 1
}

func (e *Editor) textX(content layout.Rect) int { return content.X + e.gutterWidth() }

func (e *Editor) measure(availW, availH int) (int, int) {
	gutter := e.gutterWidth()
	innerW := 0
	if availW > 0 {
		innerW = maxInt(0, availW-gutter)
	}
	if e.View.View.Wrap == text.NoWrap {
		w := 0
		for i := 0; i < e.Edit.Text.LineCount(); i++ {
			if lw := e.Edit.Text.LineWidth(i); lw > w {
				w = lw
			}
		}
		if availW <= 0 {
			return w + gutter, e.Edit.Text.LineCount()
		}
		return minInt(availW, w+gutter), e.Edit.Text.LineCount()
	}
	lines := text.SplitLines(text.Plain(e.Edit.Value()), innerW, e.View.View.Wrap)
	w := 0
	for _, l := range lines {
		if lw := l.Width(); lw > w {
			w = lw
		}
	}
	if w > innerW {
		w = innerW
	}
	return minInt(availW, w+gutter), len(lines)
}

// HandleKey implements editing and navigation.
func (e *Editor) HandleKey(ev *input.KeyEvent) bool {
	if ev.Type == input.KeyRelease {
		return false
	}
	if e.OnKeyDown != nil && e.OnKeyDown(ev) {
		return true
	}
	extend := ev.Shift
	eb := e.Edit
	switch ev.Name {
	case "left":
		if ev.Ctrl || ev.Alt {
			eb.MoveWordLeft(extend)
		} else {
			eb.MoveLeft(extend)
		}
	case "right":
		if ev.Ctrl || ev.Alt {
			eb.MoveWordRight(extend)
		} else {
			eb.MoveRight(extend)
		}
	case "up":
		eb.MoveUp(extend)
	case "down":
		eb.MoveDown(extend)
	case "pageup":
		e.page(-1, extend)
	case "pagedown":
		e.page(1, extend)
	case "home":
		if ev.Ctrl {
			eb.MoveDocStart(extend)
		} else {
			eb.MoveHome(extend)
		}
	case "end":
		if ev.Ctrl {
			eb.MoveDocEnd(extend)
		} else {
			eb.MoveEnd(extend)
		}
	case "backspace":
		if ev.Ctrl || ev.Alt {
			eb.DeleteWordBackward()
		} else {
			eb.Backspace()
		}
	case "delete", "del":
		if ev.Ctrl || ev.Alt {
			eb.DeleteWordForward()
		} else {
			eb.DeleteForward()
		}
	case "return", "enter":
		eb.Insert("\n")
	case "tab":
		tab := eb.Text.TabWidth
		if tab <= 0 {
			tab = 4
		}
		eb.Insert(strings.Repeat(" ", tab))
	default:
		if ev.Ctrl {
			switch ev.Name {
			case "a":
				eb.SelectAll()
			case "z":
				if ev.Shift {
					eb.Redo()
				} else {
					eb.Undo()
				}
			case "y":
				eb.Redo()
			case "u":
				eb.DeleteWordBackward()
			default:
				return false
			}
		} else if ev.IsText() {
			eb.Insert(string(ev.Rune))
		} else {
			return false
		}
	}
	e.afterChange()
	return true
}

func (e *Editor) page(dir int, extend bool) {
	rows := maxInt(1, e.View.View.Height-1)
	line, _ := e.Edit.LineCol()
	target := line + dir*rows
	if target < 0 {
		e.Edit.MoveDocStart(extend)
		return
	}
	if target >= e.Edit.Text.LineCount() {
		e.Edit.MoveDocEnd(extend)
		return
	}
	_, col := e.Edit.LineCol()
	off := e.Edit.Text.Offset(target, col)
	e.Edit.MoveTo(off, extend)
}

func (e *Editor) afterChange() {
	e.View.Invalidate()
	e.Invalidate()
	e.RequestRender()
}

// HandlePaste inserts pasted text.
func (e *Editor) HandlePaste(ev *input.PasteEvent) bool {
	if e.OnPaste != nil && e.OnPaste(ev) {
		return true
	}
	s := strings.ReplaceAll(ev.Text, "\r\n", "\n")
	e.Edit.Insert(s)
	return true
}

// HandleMouse handles clicking, dragging, and wheel scrolling.
func (e *Editor) HandleMouse(ev *input.MouseEvent) bool {
	if e.OnMouse != nil && e.OnMouse(ev) {
		return true
	}
	content := e.ContentBounds()
	textX := e.textX(content)
	switch ev.Action {
	case input.MouseWheel:
		e.View.View.ScrollBy(ev.ScrollX, ev.Scroll)
		e.RequestRender()
		return true
	case input.MousePress:
		if off, ok := e.offsetAt(ev, textX, content); ok {
			e.dragging = true
			e.dragStart = off
			e.Edit.SetCursor(off)
			e.afterChange()
		}
		return true
	case input.MouseMove:
		if e.dragging {
			if off, ok := e.offsetAt(ev, textX, content); ok {
				e.Edit.Select(e.dragStart, off)
				e.afterChange()
			}
			return true
		}
		return false
	case input.MouseRelease:
		e.dragging = false
		return true
	}
	return false
}

func (e *Editor) offsetAt(ev *input.MouseEvent, textX int, content layout.Rect) (int, bool) {
	if !content.Contains(ev.X, ev.Y) {
		return 0, false
	}
	line, col := e.View.View.HitTest(ev.X-textX, ev.Y-content.Y)
	return e.Edit.Text.Offset(line, col), true
}

// Paint draws the editor.
func (e *Editor) Paint(ctx *renderable.Context) {
	if !e.Visible {
		return
	}
	r := e.Bounds()
	if e.BG.A > 0 {
		ctx.Fill(r.X, r.Y, r.W, r.H, e.BG)
	}
	content := e.ContentBounds()
	if content.W <= 0 || content.H <= 0 {
		return
	}
	gutter := e.gutterWidth()
	textX := content.X + gutter
	innerW := maxInt(0, content.W-gutter)
	e.View.View.Resize(innerW, content.H)
	if e.Focused {
		e.View.EnsureCursorVisible()
	}

	if e.Edit.Len() == 0 && e.Placeholder != "" {
		ctx.SetString(textX, content.Y, width.Truncate(e.Placeholder, innerW), e.PlaceholderColor, e.BG, 0)
	}

	// Selection backgrounds first, then text on top.
	sel := make(map[int][]text.SelectionCell)
	for _, c := range e.View.SelectionCells() {
		sel[c.Line] = append(sel[c.Line], c)
	}
	scrollX := e.View.View.ScrollX
	scrollY := e.View.View.ScrollY
	lines := e.View.View.VisualLines()
	lineNums := len(fmt.Sprintf("%d", e.Edit.Text.LineCount()))

	if gutter > 0 {
		last := -1
		for row := 0; row < content.H; row++ {
			idx := scrollY + row
			if idx >= len(lines) {
				break
			}
			if lines[idx].Line == last {
				continue
			}
			last = lines[idx].Line
			num := fmt.Sprintf("%*d", lineNums, lines[idx].Line+1)
			ctx.SetString(content.X, content.Y+row, num, e.LineNumberFG, e.LineNumberBG, 0)
		}
	}

	for row := 0; row < content.H; row++ {
		idx := scrollY + row
		if idx >= len(lines) {
			break
		}
		vl := lines[idx]
		y := content.Y + row
		for _, c := range sel[idx] {
			for x := c.Start; x < c.End; x++ {
				sx := textX + x - scrollX
				if sx >= textX && sx < content.Right() {
					ctx.Buffer.SetBG(sx, y, e.SelectionBG)
				}
			}
		}
		for _, c := range vl.Clusters {
			sx := textX + cellColumn(vl, c.ByteStart) - scrollX
			if sx < textX {
				continue
			}
			if sx >= content.Right() {
				break
			}
			fg := e.FG
			bg := e.BG
			if selectionCovers(sel[idx], cellColumn(vl, c.ByteStart), cellColumn(vl, c.ByteStart)+c.Width) {
				fg = e.SelectionFG
				bg = e.SelectionBG
			}
			ctx.SetString(sx, y, c.Text, fg, bg, 0)
		}
	}

	if e.Focused {
		if x, y, visible := e.View.CursorPosition(); visible {
			cx := textX + x
			cy := content.Y + y
			ch := " "
			line, col := e.Edit.LineCol()
			lineText := e.Edit.Text.Line(line)
			if col < len(lineText) {
				cluster, _, _ := width.First(lineText[col:])
				ch = cluster
			}
			ctx.SetString(cx, cy, ch, e.CursorFG, e.CursorBG, 0)
			ctx.Renderer.SetCursor(cx, cy, true)
		}
	}
}

func cellColumn(vl text.VisualLine, byteCol int) int {
	x := 0
	for _, c := range vl.Clusters {
		if c.ByteStart >= byteCol {
			break
		}
		x += c.Width
	}
	return x
}

func selectionCovers(cells []text.SelectionCell, start, end int) bool {
	for _, c := range cells {
		if start < c.End && end > c.Start {
			return true
		}
	}
	return false
}

// Compile-time check that Editor is a renderable.
var _ renderable.Renderable = (*Editor)(nil)
