package renderables

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/bijim/gotui/cell"
	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
	"github.com/bijim/gotui/text"
	"github.com/bijim/gotui/width"
)

// MarkdownOptions configures the built-in Markdown renderer.
type MarkdownOptions struct {
	ID          string
	Content     string
	FG          color.RGBA
	BG          color.RGBA
	HeadingFG   color.RGBA
	LinkFG      color.RGBA
	QuoteFG     color.RGBA
	CodeFG      color.RGBA
	CodeBG      color.RGBA
	SyntaxStyle text.SyntaxStyle
	Wrap        text.WrapMode
	Style       layout.Style
}

// Markdown renders common Markdown blocks and inline emphasis without external dependencies.
// It supports headings, paragraphs, lists, quotes, fenced code, rules, links, and pipe tables.
type Markdown struct {
	*renderable.Base
	Content                    string
	FG, BG                     color.RGBA
	HeadingFG, LinkFG, QuoteFG color.RGBA
	CodeFG, CodeBG             color.RGBA
	SyntaxStyle                text.SyntaxStyle
	Wrap                       text.WrapMode
	styled                     text.StyledText
	valid                      bool
}

func NewMarkdown(opts MarkdownOptions) *Markdown {
	if opts.Style == (layout.Style{}) {
		opts.Style = layout.DefaultStyle()
	}
	if opts.FG == (color.RGBA{}) {
		opts.FG = White
	}
	if opts.HeadingFG == (color.RGBA{}) {
		opts.HeadingFG = opts.FG
	}
	if opts.LinkFG == (color.RGBA{}) {
		opts.LinkFG = color.FromInts(97, 175, 239, 255)
	}
	if opts.QuoteFG == (color.RGBA{}) {
		opts.QuoteFG = Gray
	}
	if opts.CodeFG == (color.RGBA{}) {
		opts.CodeFG = opts.FG
	}
	if opts.CodeBG == (color.RGBA{}) {
		opts.CodeBG = color.FromInts(35, 38, 45, 255)
	}
	m := &Markdown{Content: opts.Content, FG: opts.FG, BG: opts.BG, HeadingFG: opts.HeadingFG,
		LinkFG: opts.LinkFG, QuoteFG: opts.QuoteFG, CodeFG: opts.CodeFG, CodeBG: opts.CodeBG,
		SyntaxStyle: opts.SyntaxStyle, Wrap: opts.Wrap}
	m.Base = renderable.NewBase(m, &layout.Node{Style: opts.Style})
	m.ID = opts.ID
	m.Node.Measure = m.measure
	return m
}

func (m *Markdown) SetContent(content string) {
	m.Content = content
	m.valid = false
	m.Invalidate()
	m.RequestRender()
}

func (m *Markdown) compiled() text.StyledText {
	if m.valid {
		return m.styled
	}
	lines := strings.Split(strings.ReplaceAll(m.Content, "\r\n", "\n"), "\n")
	chunks := make([]text.Chunk, 0, len(lines)*2)
	appendChunk := func(s string, fg, bg color.RGBA, attr cell.Attribute) {
		if s == "" {
			return
		}
		chunks = append(chunks, text.Chunk{Text: s, FG: fg, BG: bg, Attr: attr})
	}
	for i := 0; i < len(lines); {
		line := lines[i]
		if strings.HasPrefix(strings.TrimSpace(line), "```") || strings.HasPrefix(strings.TrimSpace(line), "~~~") {
			marker := strings.TrimSpace(line)
			fenceChar := marker[0]
			fenceLen := 0
			for fenceLen < len(marker) && marker[fenceLen] == fenceChar {
				fenceLen++
			}
			lang := strings.TrimSpace(marker[fenceLen:])
			i++
			var code strings.Builder
			for i < len(lines) && !isClosingMarkdownFence(lines[i], fenceChar, fenceLen) {
				code.WriteString(lines[i])
				code.WriteByte('\n')
				i++
			}
			if i < len(lines) {
				i++
			}
			codeStyle := m.SyntaxStyle
			if codeStyle == (text.SyntaxStyle{}) {
				codeStyle = text.DefaultSyntaxStyle()
				codeStyle.Plain.FG = m.CodeFG
				codeStyle.Plain.BG = m.CodeBG
			}
			styled := text.HighlightCode(strings.TrimSuffix(code.String(), "\n"), lang, codeStyle)
			appendChunk("  ", m.CodeFG, m.CodeBG, 0)
			for _, c := range styled.Chunks {
				bg := c.BG
				if bg == (color.RGBA{}) {
					bg = m.CodeBG
				}
				appendChunk(c.Text, c.FG, bg, c.Attr)
			}
			appendChunk("\n", m.CodeFG, m.CodeBG, 0)
			continue
		}
		trim := strings.TrimSpace(line)
		if trim == "" {
			appendChunk("\n", m.FG, m.BG, 0)
			i++
			continue
		}
		if isMarkdownRule(trim) {
			appendChunk("────────────────────────────────────────\n", Gray, m.BG, 0)
			i++
			continue
		}
		if strings.HasPrefix(trim, "|") && i+1 < len(lines) && isMarkdownTableRule(lines[i+1]) {
			rows := [][]string{markdownCells(line)}
			i += 2
			for i < len(lines) && strings.Contains(lines[i], "|") && strings.TrimSpace(lines[i]) != "" {
				rows = append(rows, markdownCells(lines[i]))
				i++
			}
			for _, row := range formatMarkdownTable(rows) {
				appendChunk(row+"\n", m.FG, m.BG, 0)
			}
			continue
		}
		level, heading := markdownHeading(trim)
		if level > 0 {
			style := cell.AttrBold
			for range level - 1 {
				style |= cell.AttrUnderline
			}
			writeInlineMarkdown(&chunks, heading, m.HeadingFG, m.BG, m.HeadingFG, style, m.LinkFG)
			appendChunk("\n", m.FG, m.BG, 0)
			i++
			continue
		}
		if strings.HasPrefix(trim, ">") {
			quote := strings.TrimSpace(strings.TrimPrefix(trim, ">"))
			appendChunk("│ ", m.QuoteFG, m.BG, 0)
			writeInlineMarkdown(&chunks, quote, m.QuoteFG, m.BG, m.QuoteFG, 0, m.LinkFG)
			appendChunk("\n", m.FG, m.BG, 0)
			i++
			continue
		}
		if marker, body, ok := markdownListItem(trim); ok {
			appendChunk("  "+marker+" ", m.FG, m.BG, 0)
			writeInlineMarkdown(&chunks, body, m.FG, m.BG, m.HeadingFG, 0, m.LinkFG)
			appendChunk("\n", m.FG, m.BG, 0)
			i++
			continue
		}
		paragraph := []string{strings.TrimSpace(line)}
		i++
		for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !isMarkdownBlockLine(lines[i]) {
			paragraph = append(paragraph, strings.TrimSpace(lines[i]))
			i++
		}
		writeInlineMarkdown(&chunks, strings.Join(paragraph, " "), m.FG, m.BG, m.HeadingFG, 0, m.LinkFG)
		appendChunk("\n", m.FG, m.BG, 0)
	}
	for len(chunks) > 0 {
		last := len(chunks) - 1
		trimmed := strings.TrimRight(chunks[last].Text, "\n")
		if trimmed == chunks[last].Text {
			break
		}
		chunks[last].Text = trimmed
		if trimmed == "" {
			chunks = chunks[:last]
		} else {
			break
		}
	}
	m.styled = text.StyledText{Chunks: chunks}
	m.valid = true
	return m.styled
}

func writeInlineMarkdown(dst *[]text.Chunk, s string, fg, bg, accent color.RGBA, baseAttr cell.Attribute, linkFG color.RGBA) {
	append := func(v string, colorFG color.RGBA, attr cell.Attribute) {
		if v != "" {
			*dst = append(*dst, text.Chunk{Text: v, FG: colorFG, BG: bg, Attr: attr | baseAttr})
		}
	}
	for i := 0; i < len(s); {
		if s[i] == '`' {
			j := strings.IndexByte(s[i+1:], '`')
			if j >= 0 {
				append(s[i+1:i+1+j], accent, cell.AttrNone)
				i += j + 2
				continue
			}
		}
		if s[i] == '[' {
			mid := strings.Index(s[i:], "](")
			if mid > 0 {
				end := strings.IndexByte(s[i+mid+2:], ')')
				if end >= 0 {
					append(s[i+1:i+mid], linkFG, cell.AttrUnderline)
					i += mid + 2 + end + 1
					continue
				}
			}
		}
		if strings.HasPrefix(s[i:], "**") || strings.HasPrefix(s[i:], "__") {
			mark := s[i : i+2]
			j := strings.Index(s[i+2:], mark)
			closeAt := i + 2 + j
			valid := j >= 0 && (mark == "**" || validUnderscoreEmphasis(s, i, closeAt, 2))
			if valid {
				append(s[i+2:i+2+j], accent, cell.AttrBold)
				i += j + 4
				continue
			}
		}
		if s[i] == '*' || s[i] == '_' {
			mark := s[i : i+1]
			j := strings.Index(s[i+1:], mark)
			closeAt := i + 1 + j
			valid := j > 0 && (mark == "*" || validUnderscoreEmphasis(s, i, closeAt, 1))
			if valid {
				append(s[i+1:i+1+j], accent, cell.AttrItalic)
				i += j + 2
				continue
			}
		}
		start := i
		_, size := runeAt(s, i)
		i += size
		for i < len(s) && !strings.ContainsRune("`[*_", rune(s[i])) {
			_, size = runeAt(s, i)
			i += size
		}
		append(s[start:i], fg, cell.AttrNone)
	}
}

func runeAt(s string, i int) (rune, int) {
	r, size := utf8.DecodeRuneInString(s[i:])
	if size == 0 {
		return utf8.RuneError, 1
	}
	return r, size
}

func validUnderscoreEmphasis(s string, open, close, markerWidth int) bool {
	contentStart, contentEnd := open+markerWidth, close
	if contentStart >= contentEnd || unicode.IsSpace([]rune(s[contentStart:])[0]) || unicode.IsSpace([]rune(s[:contentEnd])[len([]rune(s[:contentEnd]))-1]) {
		return false
	}
	isWord := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }
	if open > 0 {
		prev, _ := utf8.DecodeLastRuneInString(s[:open])
		if isWord(prev) {
			return false
		}
	}
	after := close + markerWidth
	if after < len(s) {
		next, _ := utf8.DecodeRuneInString(s[after:])
		if isWord(next) {
			return false
		}
	}
	return true
}

func isClosingMarkdownFence(line string, char byte, minLength int) bool {
	trimmed := strings.TrimSpace(line)
	n := 0
	for n < len(trimmed) && trimmed[n] == char {
		n++
	}
	return n >= minLength && strings.TrimSpace(trimmed[n:]) == ""
}

func markdownHeading(s string) (int, string) {
	n := 0
	for n < len(s) && n < 6 && s[n] == '#' {
		n++
	}
	if n > 0 && len(s) > n && s[n] == ' ' {
		return n, strings.TrimSpace(s[n:])
	}
	return 0, ""
}

func markdownListItem(s string) (string, string, bool) {
	if len(s) >= 2 && (s[0] == '-' || s[0] == '*' || s[0] == '+') && s[1] == ' ' {
		return string(s[0]), strings.TrimSpace(s[2:]), true
	}
	for i := 0; i < len(s) && i < 8; i++ {
		if s[i] == '.' && i+1 < len(s) && s[i+1] == ' ' {
			return s[:i+1], strings.TrimSpace(s[i+2:]), true
		}
		if s[i] < '0' || s[i] > '9' {
			break
		}
	}
	return "", "", false
}

func isMarkdownRule(s string) bool {
	compact := strings.ReplaceAll(strings.ReplaceAll(s, " ", ""), "\t", "")
	return len(compact) >= 3 && (strings.Trim(compact, "-") == "" || strings.Trim(compact, "*") == "" || strings.Trim(compact, "_") == "")
}

func isMarkdownBlockLine(line string) bool {
	trim := strings.TrimSpace(line)
	if trim == "" || strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") || strings.HasPrefix(trim, ">") || isMarkdownRule(trim) {
		return true
	}
	if level, _ := markdownHeading(trim); level > 0 {
		return true
	}
	if _, _, ok := markdownListItem(trim); ok {
		return true
	}
	return strings.HasPrefix(trim, "|")
}
func isMarkdownTableRule(s string) bool {
	cells := markdownCells(s)
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		if strings.HasPrefix(cell, ":") {
			cell = cell[1:]
		}
		if strings.HasSuffix(cell, ":") {
			cell = cell[:len(cell)-1]
		}
		if strings.Count(cell, "-") < 3 || strings.Trim(cell, "-") != "" {
			return false
		}
	}
	return true
}

func markdownCells(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimSuffix(s, "|")
	parts := strings.Split(s, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func formatMarkdownTable(rows [][]string) []string {
	cols := 0
	for _, row := range rows {
		cols = maxInt(cols, len(row))
	}
	w := make([]int, cols)
	for _, row := range rows {
		for i, c := range row {
			w[i] = maxInt(w[i], width.StringWidth(c))
		}
	}
	out := make([]string, 0, len(rows)*2)
	for ri, row := range rows {
		var b strings.Builder
		b.WriteString("│")
		for i := 0; i < cols; i++ {
			value := ""
			if i < len(row) {
				value = row[i]
			}
			b.WriteByte(' ')
			b.WriteString(value)
			b.WriteString(strings.Repeat(" ", maxInt(0, w[i]-width.StringWidth(value))))
			b.WriteString(" │")
		}
		out = append(out, b.String())
		if ri == 0 {
			var sep strings.Builder
			sep.WriteString("├")
			for i, v := range w {
				sep.WriteString(strings.Repeat("─", v+2))
				if i+1 < cols {
					sep.WriteString("┼")
				} else {
					sep.WriteString("┤")
				}
			}
			out = append(out, sep.String())
		}
	}
	return out
}

func (m *Markdown) lines(width int) []text.Line { return text.SplitLines(m.compiled(), width, m.Wrap) }
func (m *Markdown) measure(availW, _ int) (int, int) {
	lines := m.lines(availW)
	maxW := 0
	for _, l := range lines {
		maxW = maxInt(maxW, l.Width())
	}
	if availW <= 0 {
		return maxW, len(lines)
	}
	return minInt(availW, maxW), len(lines)
}
func (m *Markdown) Paint(ctx *renderable.Context) {
	if !m.Visible {
		return
	}
	r := m.ContentBounds()
	ctx = ctx.WithClip(r)
	for i, line := range m.lines(r.W) {
		if r.Y+i >= r.Bottom() {
			break
		}
		paintLine(ctx, r.X, r.Right(), r.Y+i, line, m.FG, m.BG)
	}
}
