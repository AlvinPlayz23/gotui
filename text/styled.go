// Package text implements styled text, wrapping, and text buffers.
package text

import (
	"strings"

	"github.com/bijim/gotui/cell"
	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/width"
)

// Chunk is a run of styled text.
type Chunk struct {
	Text string
	FG   color.RGBA
	BG   color.RGBA
	Attr cell.Attribute
	Link uint32
}

// StyledText is a sequence of styled chunks.
type StyledText struct {
	Chunks []Chunk
}

// Plain returns unstyled text.
func Plain(s string) StyledText { return StyledText{Chunks: []Chunk{{Text: s}}} }

// String returns the unstyled content.
func (st StyledText) String() string {
	var sb strings.Builder
	for _, c := range st.Chunks {
		sb.WriteString(c.Text)
	}
	return sb.String()
}

// Width returns the display width of the longest line.
func (st StyledText) Width() int {
	return width.StringWidth(st.String())
}

// Span is a styled run within a laid-out line.
type Span struct {
	Text string
	FG   color.RGBA
	BG   color.RGBA
	Attr cell.Attribute
	Link uint32
}

// Line is one laid-out line.
type Line []Span

// Width returns the line width in cells.
func (l Line) Width() int {
	w := 0
	for _, s := range l {
		w += width.StringWidth(s.Text)
	}
	return w
}

// String returns the plain line text.
func (l Line) String() string {
	var sb strings.Builder
	for _, s := range l {
		sb.WriteString(s.Text)
	}
	return sb.String()
}

// WrapMode controls line breaking.
type WrapMode uint8

const (
	NoWrap WrapMode = iota
	WrapChar
	WrapWord
)

type unit struct {
	text string
	w    int
	fg   color.RGBA
	bg   color.RGBA
	attr cell.Attribute
	link uint32
}

// flattenWithNewlines returns units and newline markers.
func flattenStyled(st StyledText) []unit {
	var out []unit
	for _, c := range st.Chunks {
		start := 0
		for i := 0; i <= len(c.Text); i++ {
			if i == len(c.Text) || c.Text[i] == '\n' {
				seg := c.Text[start:i]
				width.Iterate(seg, func(g width.Grapheme) bool {
					out = append(out, unit{text: g.Text, w: g.Width, fg: c.FG, bg: c.BG, attr: c.Attr, link: c.Link})
					return true
				})
				if i < len(c.Text) {
					out = append(out, unit{text: "\n", w: 0, fg: c.FG, bg: c.BG, attr: c.Attr, link: c.Link})
				}
				start = i + 1
			}
		}
	}
	return out
}

// SplitLines lays out styled text into lines with the given wrap mode.
// width <= 0 means unconstrained.
func SplitLines(st StyledText, maxWidth int, mode WrapMode) []Line {
	units := flattenStyled(st)
	var lines []Line
	var cur []unit
	curW := 0
	lastSpace := -1

	flush := func() {
		lines = append(lines, toLine(cur))
		cur = nil
		curW = 0
		lastSpace = -1
	}

	for _, u := range units {
		if u.text == "\n" {
			flush()
			continue
		}
		if maxWidth > 0 && mode != NoWrap && curW+u.w > maxWidth && curW > 0 {
			if mode == WrapWord && lastSpace >= 0 {
				tail := append([]unit(nil), cur[lastSpace+1:]...)
				cur = cur[:lastSpace+1]
				flush()
				cur = tail
				for _, t := range cur {
					curW += t.w
					if t.text == " " {
						lastSpace = len(cur) - 1
					}
				}
			} else {
				flush()
			}
			if maxWidth > 0 && curW+u.w > maxWidth && curW > 0 {
				flush()
			}
		}
		if mode == WrapWord && u.text == " " {
			lastSpace = len(cur)
		}
		cur = append(cur, u)
		curW += u.w
	}
	flush()
	return lines
}

func toLine(units []unit) Line {
	out := make(Line, 0, len(units))
	for _, u := range units {
		if n := len(out); n > 0 && out[n-1].FG == u.fg && out[n-1].BG == u.bg && out[n-1].Attr == u.attr && out[n-1].Link == u.link {
			out[n-1].Text += u.text
		} else {
			out = append(out, Span{Text: u.text, FG: u.fg, BG: u.bg, Attr: u.attr, Link: u.link})
		}
	}
	return out
}

// WrapPlain wraps plain text and returns the lines.
func WrapPlain(s string, maxWidth int, mode WrapMode) []Line {
	return SplitLines(Plain(s), maxWidth, mode)
}

// TruncateLine truncates a line to max cells.
func TruncateLine(l Line, max int) Line {
	if l.Width() <= max {
		return l
	}
	out := Line{}
	used := 0
	for _, s := range l {
		var b strings.Builder
		width.Iterate(s.Text, func(g width.Grapheme) bool {
			if used+g.Width > max {
				return false
			}
			b.WriteString(g.Text)
			used += g.Width
			return true
		})
		if b.Len() > 0 {
			out = append(out, Span{Text: b.String(), FG: s.FG, BG: s.BG, Attr: s.Attr, Link: s.Link})
		}
		if used >= max {
			break
		}
	}
	return out
}

// AlignLine pads a line to width according to align ("left", "center", "right").
func AlignLine(l Line, widthCells int, align string) Line {
	w := l.Width()
	if w >= widthCells {
		return l
	}
	pad := widthCells - w
	switch align {
	case "center":
		left := pad / 2
		return padLine(l, left, pad-left)
	case "right":
		return padLine(l, pad, 0)
	default:
		return padLine(l, 0, pad)
	}
}

func padLine(l Line, left, right int) Line {
	out := make(Line, 0, len(l)+2)
	if left > 0 {
		out = append(out, Span{Text: strings.Repeat(" ", left)})
	}
	out = append(out, l...)
	if right > 0 {
		out = append(out, Span{Text: strings.Repeat(" ", right)})
	}
	return out
}
