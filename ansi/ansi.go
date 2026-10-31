// Package ansi provides escape sequences and a diffing screen encoder.
package ansi

import "strconv"

// Common escape sequences.
const (
	Esc = "\x1b"
	CSI = "\x1b["

	Reset       = CSI + "0m"
	Bold        = CSI + "1m"
	Dim         = CSI + "2m"
	Italic      = CSI + "3m"
	Underline   = CSI + "4m"
	Blink       = CSI + "5m"
	Inverse     = CSI + "7m"
	Hidden      = CSI + "8m"
	Strike      = CSI + "9m"
	NoBold      = CSI + "22m"
	NoItalic    = CSI + "23m"
	NoUnderline = CSI + "24m"
	NoBlink     = CSI + "25m"
	NoInverse   = CSI + "27m"
	NoHidden    = CSI + "28m"
	NoStrike    = CSI + "29m"

	ClearScreen = CSI + "2J"
	ClearLine   = CSI + "2K"
	Home        = CSI + "H"
	CursorUp    = CSI + "A"
	CursorDown  = CSI + "B"
	CursorRight = CSI + "C"
	CursorLeft  = CSI + "D"

	HideCursor = CSI + "?25l"
	ShowCursor = CSI + "?25h"

	EnterAltScreen = CSI + "?1049h"
	ExitAltScreen  = CSI + "?1049l"

	EnableBracketedPaste  = CSI + "?2004h"
	DisableBracketedPaste = CSI + "?2004l"

	EnableMouse        = CSI + "?1000h" + CSI + "?1002h" + CSI + "?1006h"
	DisableMouse       = CSI + "?1000l" + CSI + "?1002l" + CSI + "?1006l"
	EnableMouseMotion  = CSI + "?1003h"
	DisableMouseMotion = CSI + "?1003l"

	BeginSynchronizedUpdate = CSI + "?2026h"
	EndSynchronizedUpdate   = CSI + "?2026l"

	EnableFocusEvents  = CSI + "?1004h"
	DisableFocusEvents = CSI + "?1004l"

	SaveCursor    = Esc + "7"
	RestoreCursor = Esc + "8"
)

// CursorTo returns a sequence moving the cursor to 1-based column,row.
func CursorTo(x, y int) string {
	if x < 1 {
		x = 1
	}
	if y < 1 {
		y = 1
	}
	return CSI + strconv.Itoa(y) + ";" + strconv.Itoa(x) + "H"
}

// MoveTo is an alias for CursorTo.
func MoveTo(x, y int) string { return CursorTo(x, y) }

// ScrollRegion sets the scrolling region to rows top..bottom (1-based, inclusive).
func ScrollRegion(top, bottom int) string {
	return CSI + strconv.Itoa(top) + ";" + strconv.Itoa(bottom) + "r"
}

// SetCursorStyle sets the cursor shape: 0 default, 1 block, 2 block (steady),
// 3 underline, 4 underline steady, 5 bar, 6 bar steady.
func SetCursorStyle(shape int, blink bool) string {
	if blink {
		if shape%2 == 1 && shape != 0 {
			shape-- // 3->2, 5->4, 1->0? keep simple
		}
	}
	return CSI + strconv.Itoa(shape) + " q"
}
