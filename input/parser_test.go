package input

import "testing"

func TestSGRMouseWheel(t *testing.T) {
	cases := []struct {
		seq  string
		dir  int
		dirX int
	}{
		{"\x1b[<64;5;5M", -1, 0},
		{"\x1b[<65;5;5M", 1, 0},
		{"\x1b[<66;5;5M", 0, -1},
		{"\x1b[<67;5;5M", 0, 1},
	}
	for _, c := range cases {
		evs := NewParser().Feed([]byte(c.seq))
		if len(evs) != 1 || evs[0].Type != EventMouse {
			t.Fatalf("%q: events %+v", c.seq, evs)
		}
		m := evs[0].Mouse
		if m.Action != MouseWheel || m.Scroll != c.dir || m.ScrollX != c.dirX {
			t.Fatalf("%q: got action=%v scroll=%d scrollX=%d", c.seq, m.Action, m.Scroll, m.ScrollX)
		}
		if m.X != 4 || m.Y != 4 {
			t.Fatalf("%q: coords %d,%d", c.seq, m.X, m.Y)
		}
	}
}

func TestSGRMouseClick(t *testing.T) {
	evs := NewParser().Feed([]byte("\x1b[<0;10;3M\x1b[<0;10;3m"))
	if len(evs) != 2 {
		t.Fatalf("events %+v", evs)
	}
	if evs[0].Mouse.Action != MousePress || evs[1].Mouse.Action != MouseRelease {
		t.Fatalf("actions %v %v", evs[0].Mouse.Action, evs[1].Mouse.Action)
	}
	if evs[0].Mouse.X != 9 || evs[0].Mouse.Y != 2 {
		t.Fatalf("coords %d,%d", evs[0].Mouse.X, evs[0].Mouse.Y)
	}
}

func TestPageKeys(t *testing.T) {
	evs := NewParser().Feed([]byte("\x1b[6~\x1b[5~"))
	if len(evs) != 2 || evs[0].Key.Name != "pagedown" || evs[1].Key.Name != "pageup" {
		t.Fatalf("events %+v", evs)
	}
}

func TestSplitEscape(t *testing.T) {
	p := NewParser()
	if evs := p.Feed([]byte("\x1b")); len(evs) != 0 {
		t.Fatalf("lone escape should wait: %+v", evs)
	}
	evs := p.Flush()
	if len(evs) != 1 || evs[0].Key.Name != "escape" {
		t.Fatalf("flush events %+v", evs)
	}
}
