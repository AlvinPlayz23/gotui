package renderables

import (
	"strings"

	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
	"github.com/bijim/gotui/text"
)

// LineNumberOptions configures a plain text view with a numbered gutter.
type LineNumberOptions struct {
	ID          string
	FG          color.RGBA
	BG          color.RGBA
	NumberFG    color.RGBA
	NumberBG    color.RGBA
	StartLine   int
	GutterWidth int
	Wrap        text.WrapMode
	Style       layout.Style
}

// LineNumber renders text beside a line-number gutter.
type LineNumber struct {
	*renderable.Base
	Content     string
	FG, BG      color.RGBA
	NumberFG    color.RGBA
	NumberBG    color.RGBA
	StartLine   int
	GutterWidth int
	Wrap        text.WrapMode
}

func NewLineNumber(content string, opts ...LineNumberOptions) *LineNumber {
	var o LineNumberOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Style == (layout.Style{}) {
		o.Style = layout.DefaultStyle()
	}
	if o.StartLine <= 0 {
		o.StartLine = 1
	}
	if o.FG == (color.RGBA{}) {
		o.FG = White
	}
	if o.NumberFG == (color.RGBA{}) {
		o.NumberFG = Gray
	}
	if o.GutterWidth < 0 {
		o.GutterWidth = 0
	}
	n := &LineNumber{Content: content, FG: o.FG, BG: o.BG, NumberFG: o.NumberFG, NumberBG: o.NumberBG,
		StartLine: o.StartLine, GutterWidth: o.GutterWidth, Wrap: o.Wrap}
	n.Base = renderable.NewBase(n, &layout.Node{Style: o.Style})
	n.ID = o.ID
	n.Node.Measure = n.measure
	return n
}

func (n *LineNumber) SetContent(content string) {
	n.Content = content
	n.Invalidate()
	n.RequestRender()
}

func (n *LineNumber) gutter() int {
	return maxInt(n.GutterWidth, len(intString(n.StartLine+len(strings.Split(n.Content, "\n"))-1))+1)
}

func (n *LineNumber) lines(width int) []text.Line {
	return text.SplitLines(text.Plain(n.Content), width, n.Wrap)
}

func (n *LineNumber) measure(availW, _ int) (int, int) {
	if availW <= 0 {
		lines := n.lines(0)
		maxW := 0
		for _, line := range lines {
			maxW = maxInt(maxW, line.Width())
		}
		return maxW + n.gutter(), len(lines)
	}
	w := maxInt(0, availW-n.gutter())
	lines := n.lines(w)
	maxW := 0
	for _, line := range lines {
		maxW = maxInt(maxW, line.Width())
	}
	return minInt(availW, maxW+n.gutter()), len(lines)
}

func (n *LineNumber) Paint(ctx *renderable.Context) {
	if !n.Visible {
		return
	}
	r := n.ContentBounds()
	ctx = ctx.WithClip(r)
	gutter := minInt(n.gutter(), r.W)
	textWidth := maxInt(0, r.W-gutter)
	lines := n.lines(textWidth)
	numbers := visualLineNumbers(n.Content, n.StartLine, textWidth, n.Wrap)
	for i, line := range lines {
		y := r.Y + i
		if y >= r.Bottom() {
			break
		}
		number := 0
		if i < len(numbers) {
			number = numbers[i]
		}
		label := strings.Repeat(" ", gutter)
		if number > 0 {
			value := intString(number)
			pad := maxInt(0, gutter-len(value)-1)
			label = strings.Repeat(" ", pad) + value + " "
		}
		ctx.SetString(r.X, y, label, n.NumberFG, n.NumberBG, 0)
		paintLine(ctx, r.X+gutter, r.X+r.W, y, line, n.FG, n.BG)
	}
}
