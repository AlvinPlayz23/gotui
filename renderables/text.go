// Package renderables contains the built-in widgets.
package renderables

import (
	"github.com/bijim/gotui/cell"
	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
	"github.com/bijim/gotui/text"
	"github.com/bijim/gotui/width"
)

var (
	Transparent = color.RGBA{}
	White       = color.FromInts(255, 255, 255, 255)
	Black       = color.FromInts(0, 0, 0, 255)
	Gray        = color.FromInts(128, 128, 128, 255)
	DarkGray    = color.FromInts(64, 64, 64, 255)
)

// BoxOptions configures a Box.
type BoxOptions struct {
	ID                   string
	Style                layout.Style
	Background           color.RGBA
	Border               bool
	BorderSides          cell.BorderSides
	BorderStyle          cell.BorderStyle
	BorderChars          cell.BorderCharacters
	BorderColor          color.RGBA
	FocusedBorderColor   color.RGBA
	Title                string
	BottomTitle          string
	TitleColor           color.RGBA
	TitleAlignment       string
	BottomTitleAlignment string
	Padding              int
	PaddingX, PaddingY   int
}

// Box is a container with optional background, border, and titles.
type Box struct {
	*renderable.Base
	Background           color.RGBA
	Border               bool
	BorderSides          cell.BorderSides
	BorderStyle          cell.BorderStyle
	BorderChars          cell.BorderCharacters
	BorderColor          color.RGBA
	FocusedBorderColor   color.RGBA
	Title                string
	BottomTitle          string
	TitleColor           color.RGBA
	TitleAlignment       string
	BottomTitleAlignment string
}

// NewBox creates a Box.
func NewBox(opts BoxOptions) *Box {
	if opts.Style == (layout.Style{}) {
		opts.Style = layout.DefaultStyle()
	}
	b := &Box{
		Background:           opts.Background,
		Border:               opts.Border,
		BorderSides:          opts.BorderSides,
		BorderStyle:          opts.BorderStyle,
		BorderChars:          opts.BorderChars,
		BorderColor:          opts.BorderColor,
		FocusedBorderColor:   opts.FocusedBorderColor,
		Title:                opts.Title,
		BottomTitle:          opts.BottomTitle,
		TitleColor:           opts.TitleColor,
		TitleAlignment:       opts.TitleAlignment,
		BottomTitleAlignment: opts.BottomTitleAlignment,
	}
	b.Base = renderable.NewBase(b, &layout.Node{Style: opts.Style})
	if b.BorderSides == 0 && opts.Border {
		b.BorderSides = cell.AllSides
	}
	if b.BorderStyle == "" {
		b.BorderStyle = cell.BorderSingle
	}
	if b.BorderColor == (color.RGBA{}) {
		b.BorderColor = Gray
	}
	if b.TitleColor == (color.RGBA{}) {
		b.TitleColor = White
	}
	if opts.Border {
		b.Base.Node.Style.Border = [4]float64{1, 1, 1, 1}
	}
	applyPadding(&b.Base.Node.Style, opts.Padding, opts.PaddingX, opts.PaddingY)
	return b
}

func applyPadding(s *layout.Style, all, x, y int) {
	if all > 0 {
		for i := 0; i < 4; i++ {
			s.Padding[i] = layout.P(all)
		}
	}
	if x > 0 {
		s.Padding[1] = layout.P(x)
		s.Padding[3] = layout.P(x)
	}
	if y > 0 {
		s.Padding[0] = layout.P(y)
		s.Padding[2] = layout.P(y)
	}
}

// Paint draws the box and its children.
func (b *Box) Paint(ctx *renderable.Context) {
	if !b.Visible {
		return
	}
	r := b.Bounds()
	if b.Background.A > 0 {
		ctx.Fill(r.X, r.Y, r.W, r.H, b.Background)
	}
	if b.Border && b.BorderSides != 0 {
		bc := b.BorderColor
		if b.Focused && b.FocusedBorderColor.A > 0 {
			bc = b.FocusedBorderColor
		}
		chars := b.BorderChars
		if chars == (cell.BorderCharacters{}) {
			chars = cell.BorderChars(b.BorderStyle)
		}
		ctx.DrawBox(r.X, r.Y, r.W, r.H, b.BorderSides, chars, bc, b.Background, 0)
		if b.Title != "" && b.BorderSides&cell.SideTop != 0 {
			b.drawTitle(ctx, b.Title, b.TitleAlignment, r.X+1, r.Y, r.W-2)
		}
		if b.BottomTitle != "" && b.BorderSides&cell.SideBottom != 0 {
			b.drawTitle(ctx, b.BottomTitle, b.BottomTitleAlignment, r.X+1, r.Y+r.H-1, r.W-2)
		}
	}
	content := b.ContentBounds()
	b.PaintChildren(ctx.WithClip(content))
}

// HitTest clips child hits to the content box.
func (b *Box) HitTest(x, y int) renderable.Renderable {
	if !b.Visible || !b.Bounds().Contains(x, y) {
		return nil
	}
	if b.ContentBounds().Contains(x, y) {
		if hit := b.Base.HitTest(x, y); hit != nil {
			return hit
		}
	}
	return b.Self()
}

func (b *Box) drawTitle(ctx *renderable.Context, title, align string, x, y, maxW int) {
	if maxW <= 0 {
		return
	}
	t := " " + title + " "
	tw := width.StringWidth(t)
	if tw > maxW {
		t = width.TruncateEllipsis(t, maxW)
		tw = width.StringWidth(t)
	}
	switch align {
	case "center":
		x += (maxW - tw) / 2
	case "right":
		x += maxW - tw
	}
	ctx.SetString(x, y, t, b.TitleColor, b.Background, 0)
}

// TextOptions configures a Text renderable.
type TextOptions struct {
	ID      string
	Content string
	Styled  text.StyledText
	FG      color.RGBA
	BG      color.RGBA
	Attr    cell.Attribute
	Wrap    text.WrapMode
	Align   string
	Style   layout.Style
}

// Text renders styled, optionally wrapped text.
type Text struct {
	*renderable.Base
	Content text.StyledText
	FG      color.RGBA
	BG      color.RGBA
	Attr    cell.Attribute
	Wrap    text.WrapMode
	Align   string

	lines       []text.Line
	layoutW     int
	layoutValid bool
}

// NewText creates a Text renderable. Optional TextOptions may be supplied.
func NewText(content string, opts ...TextOptions) *Text {
	var o TextOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Style == (layout.Style{}) {
		o.Style = layout.DefaultStyle()
	}
	t := &Text{FG: o.FG, BG: o.BG, Attr: o.Attr, Wrap: o.Wrap, Align: o.Align}
	if t.FG == (color.RGBA{}) {
		t.FG = White
	}
	if o.Styled.Chunks != nil {
		t.Content = o.Styled
	} else {
		t.Content = text.Plain(content)
	}
	t.Base = renderable.NewBase(t, &layout.Node{Style: o.Style})
	t.Node.Measure = t.measure
	return t
}

// SetContent replaces the text.
func (t *Text) SetContent(s string) {
	t.Content = text.Plain(s)
	t.layoutValid = false
	t.Invalidate()
	t.RequestRender()
}

// Invalidate clears cached layout.
func (t *Text) Invalidate() {
	t.layoutValid = false
	t.Base.Invalidate()
}

func (t *Text) layout(width int) []text.Line {
	w := 0
	if t.Wrap != text.NoWrap {
		w = width
	}
	if t.layoutValid && t.layoutW == w {
		return t.lines
	}
	t.lines = text.SplitLines(t.Content, w, t.Wrap)
	t.layoutW = w
	t.layoutValid = true
	return t.lines
}

// SetStyled replaces the styled content.
func (t *Text) SetStyled(st text.StyledText) {
	t.Content = st
	t.layoutValid = false
	t.Invalidate()
	t.RequestRender()
}

func (t *Text) measure(availW, availH int) (int, int) {
	lines := t.layout(availW)
	maxW := 0
	for _, l := range lines {
		if w := l.Width(); w > maxW {
			maxW = w
		}
	}
	if t.Wrap != text.NoWrap && availW > 0 && maxW > availW {
		maxW = availW
	}
	return maxW, len(lines)
}

// Paint draws the text.
func (t *Text) Paint(ctx *renderable.Context) {
	if !t.Visible {
		return
	}
	content := t.ContentBounds()
	lines := t.layout(content.W)
	for i, line := range lines {
		y := content.Y + i
		if y >= content.Y+content.H {
			break
		}
		if t.Align != "" && t.Align != "left" {
			line = text.AlignLine(line, content.W, t.Align)
		}
		x := content.X
		for _, sp := range line {
			fg := sp.FG
			if fg == (color.RGBA{}) {
				fg = t.FG
			}
			bg := sp.BG
			if bg == (color.RGBA{}) {
				bg = t.BG
			}
			attr := sp.Attr | t.Attr
			ctx.SetString(x, y, sp.Text, fg, bg, attr)
			x += width.StringWidth(sp.Text)
			if x >= content.Right() {
				break
			}
		}
	}
}
