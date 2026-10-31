package renderables

import (
	"strings"

	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
	"github.com/bijim/gotui/text"
	"github.com/bijim/gotui/width"
)

// CodeOptions configures a syntax-highlighted source view.
type CodeOptions struct {
	ID           string
	Language     string
	SyntaxStyle  text.SyntaxStyle
	FG           color.RGBA
	BG           color.RGBA
	Wrap         text.WrapMode
	LineNumbers  bool
	LineNumberFG color.RGBA
	LineNumberBG color.RGBA
	StartLine    int
	TabWidth     int
	Style        layout.Style
}

// Code renders source text with built-in lexical highlighting and optional line numbers.
type Code struct {
	*renderable.Base
	Content     string
	Language    string
	SyntaxStyle text.SyntaxStyle
	FG, BG      color.RGBA
	Wrap        text.WrapMode
	LineNumbers bool
	NumberFG    color.RGBA
	NumberBG    color.RGBA
	StartLine   int
	TabWidth    int
	cacheValid  bool
	cacheSource string
	cacheLang   string
	cacheStyle  text.SyntaxStyle
	cacheText   text.StyledText
}

func NewCode(content string, opts ...CodeOptions) *Code {
	var o CodeOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Style == (layout.Style{}) {
		o.Style = layout.DefaultStyle()
	}
	if o.StartLine <= 0 {
		o.StartLine = 1
	}
	c := &Code{Content: content, Language: o.Language, SyntaxStyle: o.SyntaxStyle, FG: o.FG, BG: o.BG,
		Wrap: o.Wrap, LineNumbers: o.LineNumbers, NumberFG: o.LineNumberFG, NumberBG: o.LineNumberBG,
		StartLine: o.StartLine, TabWidth: o.TabWidth}
	if c.FG == (color.RGBA{}) {
		c.FG = White
	}
	if c.NumberFG == (color.RGBA{}) {
		c.NumberFG = Gray
	}
	if c.NumberBG == (color.RGBA{}) {
		c.NumberBG = c.BG
	}
	c.Base = renderable.NewBase(c, &layout.Node{Style: o.Style})
	c.ID = o.ID
	c.Node.Measure = c.measure
	return c
}

func (c *Code) SetContent(source string) {
	c.Content = source
	c.Invalidate()
	c.RequestRender()
}

func (c *Code) SetLanguage(language string) {
	c.Language = language
	c.RequestRender()
}

func (c *Code) lines(width int) []text.Line {
	source := expandCodeTabs(c.Content, c.TabWidth)
	if !c.cacheValid || c.cacheSource != source || c.cacheLang != c.Language || c.cacheStyle != c.SyntaxStyle {
		c.cacheText = text.HighlightCode(source, c.Language, c.SyntaxStyle)
		c.cacheSource, c.cacheLang, c.cacheStyle = source, c.Language, c.SyntaxStyle
		c.cacheValid = true
	}
	return text.SplitLines(c.cacheText, width, c.Wrap)
}

func expandCodeTabs(source string, tabWidth int) string {
	if tabWidth <= 0 {
		tabWidth = 4
	}
	var b strings.Builder
	column := 0
	for _, r := range source {
		switch r {
		case '\n':
			b.WriteRune(r)
			column = 0
		case '\t':
			spaces := tabWidth - column%tabWidth
			b.WriteString(strings.Repeat(" ", spaces))
			column += spaces
		default:
			b.WriteRune(r)
			column += width.RuneWidth(r)
		}
	}
	return b.String()
}

func (c *Code) gutterWidth() int {
	if !c.LineNumbers {
		return 0
	}
	n := len(intString(c.StartLine + len(strings.Split(c.Content, "\n")) - 1))
	return n + 1
}

func visualLineNumbers(source string, start, width int, wrap text.WrapMode) []int {
	logical := strings.Split(source, "\n")
	if start <= 0 {
		start = 1
	}
	numbers := make([]int, 0, len(logical))
	for lineIndex, line := range logical {
		visual := text.SplitLines(text.Plain(line), width, wrap)
		if len(visual) == 0 {
			visual = []text.Line{{}}
		}
		for row := range visual {
			if row == 0 {
				numbers = append(numbers, start+lineIndex)
			} else {
				numbers = append(numbers, 0)
			}
		}
	}
	return numbers
}

func (c *Code) measure(availW, _ int) (int, int) {
	if availW <= 0 {
		lines := c.lines(0)
		maxW := 0
		for _, line := range lines {
			maxW = maxInt(maxW, line.Width())
		}
		return maxW + c.gutterWidth(), len(lines)
	}
	w := maxInt(0, availW-c.gutterWidth())
	lines := c.lines(w)
	maxW := 0
	for _, line := range lines {
		maxW = maxInt(maxW, line.Width())
	}
	maxW = minInt(availW, maxW+c.gutterWidth())
	return maxW, len(lines)
}

func (c *Code) Paint(ctx *renderable.Context) {
	if !c.Visible {
		return
	}
	r := c.ContentBounds()
	ctx = ctx.WithClip(r)
	gutter := minInt(c.gutterWidth(), r.W)
	textWidth := maxInt(0, r.W-gutter)
	lines := c.lines(textWidth)
	numbers := visualLineNumbers(expandCodeTabs(c.Content, c.TabWidth), c.StartLine, textWidth, c.Wrap)
	for i, line := range lines {
		y := r.Y + i
		if y >= r.Bottom() {
			break
		}
		if c.LineNumbers && gutter > 0 {
			n := 0
			if i < len(numbers) {
				n = numbers[i]
			}
			label := strings.Repeat(" ", gutter)
			if n > 0 {
				label = strings.Repeat(" ", maxInt(0, gutter-1-len(intString(n)))) + intString(n) + " "
			}
			ctx.SetString(r.X, y, label, c.NumberFG, c.NumberBG, 0)
		}
		paintLine(ctx, r.X+gutter, r.X+r.W, y, line, c.FG, c.BG)
	}
}

func intString(v int) string {
	if v == 0 {
		return "0"
	}
	var b [24]byte
	i := len(b)
	neg := v < 0
	if neg {
		v = -v
	}
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func paintLine(ctx *renderable.Context, x, right, y int, line text.Line, fg, bg color.RGBA) {
	if y < ctx.Clip.Y || y >= ctx.Clip.Bottom() {
		return
	}
	for _, span := range line {
		if x >= right {
			return
		}
		spanFG, spanBG := span.FG, span.BG
		if spanFG == (color.RGBA{}) {
			spanFG = fg
		}
		if spanBG == (color.RGBA{}) {
			spanBG = bg
		}
		ctx.SetString(x, y, span.Text, spanFG, spanBG, span.Attr)
		x += width.StringWidth(span.Text)
	}
}
