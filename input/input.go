// Package input parses terminal keyboard, mouse, paste, and focus input.
package input

// EventType distinguishes event kinds.
type EventType uint8

const (
	EventKey EventType = iota
	EventMouse
	EventPaste
	EventFocus
	EventResize
)

// KeyType is press, repeat, or release (kitty protocol).
type KeyType uint8

const (
	KeyPress KeyType = iota
	KeyRepeat
	KeyRelease
)

// KeyEvent describes a keyboard event.
type KeyEvent struct {
	Name  string // "a", "up", "return", "f1", ...
	Rune  rune
	Code  int
	Ctrl  bool
	Alt   bool
	Shift bool
	Meta  bool
	Type  KeyType
	Raw   string
}

// IsText reports whether the event inserts a printable character.
func (e KeyEvent) IsText() bool {
	return e.Rune != 0 && !e.Ctrl && !e.Alt && e.Rune >= 0x20 && e.Rune != 0x7f
}

// MouseAction is a mouse button state.
type MouseAction uint8

const (
	MousePress MouseAction = iota
	MouseRelease
	MouseMove
	MouseWheel
)

// MouseEvent describes a mouse event.
type MouseEvent struct {
	X, Y    int // 0-based cell coordinates
	Button  int // 0 left, 1 middle, 2 right
	Action  MouseAction
	Scroll  int // -1 up, +1 down; or horizontal
	ScrollX int
	Ctrl    bool
	Alt     bool
	Shift   bool
	Motion  bool
	// ScrollDeltaX/Y carry the remaining cell delta while a wheel event bubbles
	// through nested scroll containers. Applications normally leave these unset.
	ScrollDeltaX, ScrollDeltaY int
	ScrollDeltaSet             bool
}

// PasteEvent carries pasted text.
type PasteEvent struct {
	Text string
}

// FocusEvent reports terminal focus.
type FocusEvent struct {
	Focused bool
}

// Event is a union of terminal events.
type Event struct {
	Type   EventType
	Key    KeyEvent
	Mouse  MouseEvent
	Paste  PasteEvent
	Focus  FocusEvent
	Width  int
	Height int
}
