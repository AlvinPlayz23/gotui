package cell

import "github.com/bijim/gotui/color"

// BorderSides is a bitmask of box sides.
type BorderSides uint8

const (
	SideLeft BorderSides = 1 << iota
	SideBottom
	SideRight
	SideTop
)

// AllSides includes every border side.
const AllSides = SideLeft | SideBottom | SideRight | SideTop

// BorderCharacters holds the eight glyphs of a box border.
type BorderCharacters struct {
	TopLeft, TopRight, BottomLeft, BottomRight       rune
	Top, Bottom, Left, Right                         rune
	MiddleLeft, MiddleRight, MiddleTop, MiddleBottom rune // optional separators
}

// BorderStyle names a preset border style.
type BorderStyle string

const (
	BorderSingle       BorderStyle = "single"
	BorderDouble       BorderStyle = "double"
	BorderRound        BorderStyle = "round"
	BorderBold         BorderStyle = "bold"
	BorderSingleDouble BorderStyle = "singleDouble"
	BorderDoubleSingle BorderStyle = "doubleSingle"
	BorderClassic      BorderStyle = "classic"
)

// BorderChars returns the characters for a border style.
func BorderChars(style BorderStyle) BorderCharacters {
	switch style {
	case BorderDouble:
		return BorderCharacters{'╔', '╗', '╚', '╝', '═', '═', '║', '║', '╠', '╣', '╦', '╩'}
	case BorderRound:
		return BorderCharacters{'╭', '╮', '╰', '╯', '─', '─', '│', '│', '├', '┤', '┬', '┴'}
	case BorderBold:
		return BorderCharacters{'┏', '┓', '┗', '┛', '━', '━', '┃', '┃', '┣', '┫', '┳', '┻'}
	case BorderSingleDouble:
		return BorderCharacters{'╒', '╕', '╘', '╛', '═', '═', '│', '│', '╞', '╡', '╤', '╧'}
	case BorderDoubleSingle:
		return BorderCharacters{'╓', '╖', '╙', '╜', '─', '─', '║', '║', '╟', '╢', '╥', '╨'}
	case BorderClassic:
		return BorderCharacters{'+', '+', '+', '+', '-', '-', '|', '|', '+', '+', '+', '+'}
	default:
		return BorderCharacters{'┌', '┐', '└', '┘', '─', '─', '│', '│', '├', '┤', '┬', '┴'}
	}
}

// DrawBox draws a border rectangle at x,y with width w and height h.
func (b *Buffer) DrawBox(x, y, w, h int, sides BorderSides, chars BorderCharacters, fg, bg color.RGBA, attr Attribute) {
	if w <= 0 || h <= 0 {
		return
	}
	set := func(px, py int, r rune) {
		if r != 0 {
			b.Set(px, py, r, fg, bg, attr)
		}
	}
	right, bottom := x+w-1, y+h-1
	if sides&SideTop != 0 {
		for px := x + 1; px < right; px++ {
			set(px, y, chars.Top)
		}
	}
	if sides&SideBottom != 0 {
		for px := x + 1; px < right; px++ {
			set(px, bottom, chars.Bottom)
		}
	}
	if sides&SideLeft != 0 {
		for py := y + 1; py < bottom; py++ {
			set(x, py, chars.Left)
		}
	}
	if sides&SideRight != 0 {
		for py := y + 1; py < bottom; py++ {
			set(right, py, chars.Right)
		}
	}
	if sides&SideTop != 0 && sides&SideLeft != 0 {
		set(x, y, chars.TopLeft)
	}
	if sides&SideTop != 0 && sides&SideRight != 0 {
		set(right, y, chars.TopRight)
	}
	if sides&SideBottom != 0 && sides&SideLeft != 0 {
		set(x, bottom, chars.BottomLeft)
	}
	if sides&SideBottom != 0 && sides&SideRight != 0 {
		set(right, bottom, chars.BottomRight)
	}
}
