package input

import (
	"bytes"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Parser incrementally parses terminal input bytes.
type Parser struct {
	buf      []byte
	paste    bool
	pasteBuf []byte
}

// NewParser returns an empty parser.
func NewParser() *Parser { return &Parser{} }

// Feed appends bytes and returns every complete event.
func (p *Parser) Feed(data []byte) []Event {
	if len(data) == 0 {
		return nil
	}
	p.buf = append(p.buf, data...)
	var events []Event
	for {
		if p.paste {
			end := []byte("\x1b[201~")
			idx := bytes.Index(p.buf, end)
			if idx < 0 {
				p.pasteBuf = append(p.pasteBuf, p.buf...)
				p.buf = p.buf[:0]
				return events
			}
			p.pasteBuf = append(p.pasteBuf, p.buf[:idx]...)
			text := string(p.pasteBuf)
			p.pasteBuf = p.pasteBuf[:0]
			p.paste = false
			p.buf = p.buf[idx+len(end):]
			events = append(events, Event{Type: EventPaste, Paste: PasteEvent{Text: text}})
			continue
		}
		if len(p.buf) == 0 {
			return events
		}
		ev, n, need := p.parseOne(false)
		if need {
			return events
		}
		if n <= 0 {
			// Avoid an infinite loop on unknown input: consume one byte.
			n = 1
		}
		p.buf = p.buf[n:]
		if ev != nil {
			events = append(events, *ev)
		}
	}
}

// Pending reports buffered bytes not yet parsed into events.
func (p *Parser) Pending() int { return len(p.buf) }

// Flush completes a pending lone escape or partial sequence.
func (p *Parser) Flush() []Event {
	if p.paste {
		text := string(p.pasteBuf)
		p.pasteBuf = p.pasteBuf[:0]
		p.paste = false
		if text != "" {
			return []Event{{Type: EventPaste, Paste: PasteEvent{Text: text}}}
		}
		return nil
	}
	var events []Event
	for len(p.buf) > 0 {
		ev, n, _ := p.parseOne(true)
		if n <= 0 {
			n = 1
		}
		p.buf = p.buf[n:]
		if ev != nil {
			events = append(events, *ev)
		}
	}
	return events
}

// Reset clears the parser state.
func (p *Parser) Reset() {
	p.buf = p.buf[:0]
	p.pasteBuf = p.pasteBuf[:0]
	p.paste = false
}

func (p *Parser) parseOne(flush bool) (*Event, int, bool) {
	b := p.buf
	c := b[0]
	if c == 0x1b {
		if len(b) == 1 {
			if flush {
				return &Event{Type: EventKey, Key: KeyEvent{Name: "escape", Code: 27, Raw: "\x1b"}}, 1, false
			}
			return nil, 0, true
		}
		switch b[1] {
		case '[':
			return p.parseCSI(b, flush)
		case 'O':
			if len(b) < 3 {
				return nil, 0, !flush
			}
			return p.ss3(b[2]), 3, false
		case 0x1b:
			return &Event{Type: EventKey, Key: KeyEvent{Name: "escape", Code: 27, Raw: "\x1b"}}, 1, false
		default:
			// Alt + key.
			r, size := utf8.DecodeRune(b[1:])
			if r == utf8.RuneError && size == 1 && !utf8.FullRune(b[1:]) && !flush {
				return nil, 0, true
			}
			if r == utf8.RuneError && size == 1 {
				return &Event{Type: EventKey, Key: KeyEvent{Name: "escape", Code: 27}}, 1, false
			}
			ev := keyForRune(r)
			ev.Alt = true
			ev.Raw = string(b[:1+size])
			return &Event{Type: EventKey, Key: ev}, 1 + size, false
		}
	}
	return p.parseByte(b, flush)
}

func (p *Parser) parseByte(b []byte, flush bool) (*Event, int, bool) {
	c := b[0]
	switch c {
	case 0x00:
		return keyEvent("space", 0, true), 1, false
	case 0x09:
		return keyEvent("tab", 9, false), 1, false
	case 0x0a:
		return keyEvent("linefeed", 10, false), 1, false
	case 0x0d:
		return keyEvent("return", 13, false), 1, false
	case 0x7f:
		return keyEvent("backspace", 127, false), 1, false
	}
	if c < 0x20 {
		r := rune('a' + c - 1)
		if c == 0x1b {
			r = 0
		}
		ev := keyForRune(r)
		ev.Ctrl = true
		ev.Name = ctrlName(c)
		ev.Code = int(c)
		return &Event{Type: EventKey, Key: ev}, 1, false
	}
	r, size := utf8.DecodeRune(b)
	if r == utf8.RuneError && size == 1 {
		if !utf8.FullRune(b) && !flush {
			return nil, 0, true
		}
		return nil, 1, false
	}
	ev := keyForRune(r)
	ev.Raw = string(b[:size])
	return &Event{Type: EventKey, Key: ev}, size, false
}

func (p *Parser) parseCSI(b []byte, flush bool) (*Event, int, bool) {
	if len(b) < 3 {
		return nil, 0, !flush
	}
	// Find the final byte.
	final := -1
	for i := 2; i < len(b); i++ {
		if b[i] >= 0x40 && b[i] <= 0x7e {
			final = i
			break
		}
	}
	if final < 0 {
		if flush {
			return &Event{Type: EventKey, Key: KeyEvent{Name: "escape", Code: 27}}, 1, false
		}
		if len(b) > 1024 {
			return nil, len(b), false
		}
		return nil, 0, true
	}
	params := string(b[2:final])
	f := b[final]
	n := final + 1
	if strings.HasPrefix(params, "200~") || (params == "200" && f == '~') {
		p.paste = true
		return nil, n, false
	}
	if f == 'M' && strings.HasPrefix(params, "<") {
		if ev := parseSGRMouse(params[1:], true); ev != nil {
			return ev, n, false
		}
		return nil, n, false
	}
	if f == 'm' && strings.HasPrefix(params, "<") {
		if ev := parseSGRMouse(params[1:], false); ev != nil {
			return ev, n, false
		}
		return nil, n, false
	}
	if f == 'M' && len(params) == 0 {
		// Legacy X10 mouse: ESC [ M b x y (three more bytes).
		if len(b) < n+3 {
			return nil, 0, !flush
		}
		return parseX10Mouse(b[n:]), n + 3, false
	}
	if f == 'I' {
		return &Event{Type: EventFocus, Focus: FocusEvent{Focused: true}}, n, false
	}
	if f == 'O' {
		return &Event{Type: EventFocus, Focus: FocusEvent{Focused: false}}, n, false
	}
	if f == 'u' {
		return parseKittyKey(params), n, false
	}
	if f == '~' {
		return tildeKey(params), n, false
	}
	switch f {
	case 'A', 'B', 'C', 'D', 'H', 'F', 'Z', 'P', 'Q', 'R', 'S':
		return arrowKey(f, params), n, false
	}
	if strings.HasPrefix(params, "?") {
		return nil, n, false
	}
	return nil, n, false
}

func (p *Parser) ss3(c byte) *Event {
	switch c {
	case 'P':
		return keyEvent("f1", 0, false)
	case 'Q':
		return keyEvent("f2", 0, false)
	case 'R':
		return keyEvent("f3", 0, false)
	case 'S':
		return keyEvent("f4", 0, false)
	case 'A':
		return keyEvent("up", 0, false)
	case 'B':
		return keyEvent("down", 0, false)
	case 'C':
		return keyEvent("right", 0, false)
	case 'D':
		return keyEvent("left", 0, false)
	case 'H':
		return keyEvent("home", 0, false)
	case 'F':
		return keyEvent("end", 0, false)
	}
	return nil
}

func parseModifiers(mod int) (ctrl, alt, shift, meta bool) {
	if mod <= 1 {
		return
	}
	m := mod - 1
	shift = m&1 != 0
	alt = m&2 != 0
	ctrl = m&4 != 0
	meta = m&8 != 0
	return
}

func arrowKey(f byte, params string) *Event {
	name := map[byte]string{'A': "up", 'B': "down", 'C': "right", 'D': "left", 'H': "home", 'F': "end", 'Z': "tab", 'P': "f1", 'Q': "f2", 'R': "f3", 'S': "f4"}[f]
	ev := KeyEvent{Name: name, Code: int(f)}
	if name == "f1" || name == "f2" || name == "f3" || name == "f4" {
		ev.Name = name
	}
	if parts := strings.Split(params, ";"); len(parts) >= 2 {
		if mod, err := strconv.Atoi(parts[1]); err == nil {
			ev.Ctrl, ev.Alt, ev.Shift, ev.Meta = parseModifiers(mod)
		}
	}
	if f == 'Z' {
		ev.Shift = true
	}
	return &Event{Type: EventKey, Key: ev}
}

func tildeKey(params string) *Event {
	parts := strings.Split(params, ";")
	code, _ := strconv.Atoi(parts[0])
	names := map[int]string{1: "home", 2: "insert", 3: "delete", 4: "end", 5: "pageup", 6: "pagedown", 7: "home", 8: "end", 11: "f1", 12: "f2", 13: "f3", 14: "f4", 15: "f5", 17: "f6", 18: "f7", 19: "f8", 20: "f9", 21: "f10", 23: "f11", 24: "f12"}
	name, ok := names[code]
	if !ok {
		return nil
	}
	ev := KeyEvent{Name: name, Code: code}
	if len(parts) >= 2 {
		if mod, err := strconv.Atoi(parts[1]); err == nil {
			ev.Ctrl, ev.Alt, ev.Shift, ev.Meta = parseModifiers(mod)
		}
	}
	return &Event{Type: EventKey, Key: ev}
}

func parseKittyKey(params string) *Event {
	// code;mods[:event];text
	parts := strings.Split(params, ";")
	code, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil
	}
	ev := KeyEvent{Code: code}
	if len(parts) >= 2 {
		modEvent := strings.Split(parts[1], ":")
		if mod, err := strconv.Atoi(modEvent[0]); err == nil {
			ev.Ctrl, ev.Alt, ev.Shift, ev.Meta = parseModifiers(mod)
		}
		if len(modEvent) >= 2 {
			switch modEvent[1] {
			case "2":
				ev.Type = KeyRepeat
			case "3":
				ev.Type = KeyRelease
			}
		}
	}
	switch code {
	case 13, 57414:
		ev.Name = "return"
	case 9:
		ev.Name = "tab"
	case 127:
		ev.Name = "backspace"
	case 27:
		ev.Name = "escape"
	case 32:
		ev.Name = "space"
		ev.Rune = ' '
	case 57376, 57441:
		ev.Name = "left"
	case 57377, 57442:
		ev.Name = "right"
	case 57378, 57443:
		ev.Name = "up"
	case 57379, 57444:
		ev.Name = "down"
	default:
		if code >= 1 && code <= 0x10ffff {
			r := rune(code)
			ev.Rune = r
			ev.Name = string(r)
			if ev.Ctrl {
				ev.Rune = 0
			}
		}
	}
	return &Event{Type: EventKey, Key: ev}
}

func parseSGRMouse(params string, press bool) *Event {
	parts := strings.Split(params, ";")
	if len(parts) < 3 {
		return nil
	}
	b, _ := strconv.Atoi(parts[0])
	x, _ := strconv.Atoi(parts[1])
	y, _ := strconv.Atoi(parts[2])
	mev := MouseEvent{X: x - 1, Y: y - 1}
	mev.Shift = b&4 != 0
	mev.Alt = b&8 != 0
	mev.Ctrl = b&16 != 0
	motion := b&32 != 0
	btn := b & 3
	switch {
	case b&64 != 0:
		mev.Action = MouseWheel
		switch b & 3 {
		case 0:
			mev.Scroll = -1
		case 1:
			mev.Scroll = 1
		case 2:
			mev.ScrollX = -1
		case 3:
			mev.ScrollX = 1
		}
	case motion:
		mev.Action = MouseMove
		mev.Button = btn
		if btn == 3 {
			mev.Button = -1
		}
	default:
		mev.Button = btn
		if press {
			mev.Action = MousePress
		} else {
			mev.Action = MouseRelease
		}
	}
	return &Event{Type: EventMouse, Mouse: mev}
}

func parseX10Mouse(b []byte) *Event {
	if len(b) < 3 {
		return nil
	}
	cb := int(b[0]) - 32
	x := int(b[1]) - 33
	y := int(b[2]) - 33
	mev := MouseEvent{X: x, Y: y}
	mev.Shift = cb&4 != 0
	mev.Alt = cb&8 != 0
	mev.Ctrl = cb&16 != 0
	if cb&64 != 0 {
		mev.Action = MouseWheel
		if cb&1 != 0 {
			mev.Scroll = 1
		} else {
			mev.Scroll = -1
		}
	} else if cb&32 != 0 {
		mev.Action = MouseMove
		mev.Button = cb & 3
	} else {
		mev.Action = MousePress
		mev.Button = cb & 3
	}
	return &Event{Type: EventMouse, Mouse: mev}
}

func keyEvent(name string, code int, ctrl bool) *Event {
	return &Event{Type: EventKey, Key: KeyEvent{Name: name, Code: code, Ctrl: ctrl}}
}

func keyForRune(r rune) KeyEvent {
	if r == ' ' {
		return KeyEvent{Name: "space", Rune: r, Code: 32}
	}
	return KeyEvent{Name: string(r), Rune: r, Code: int(r)}
}

func ctrlName(c byte) string {
	switch c {
	case 0x00:
		return "space"
	case 0x08:
		return "backspace"
	case 0x09:
		return "tab"
	case 0x0a, 0x0d:
		return "return"
	case 0x1b:
		return "escape"
	}
	if c >= 1 && c <= 26 {
		return string(rune('a' + c - 1))
	}
	return ""
}
