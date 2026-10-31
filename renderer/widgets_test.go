package renderer

import (
	"bytes"
	"testing"

	"github.com/bijim/gotui/input"
	"github.com/bijim/gotui/renderables"
)

func key(name string) input.Event {
	return input.Event{Type: input.EventKey, Key: input.KeyEvent{Name: name}}
}

func runeKey(r rune) input.Event {
	return input.Event{Type: input.EventKey, Key: input.KeyEvent{Name: string(r), Rune: r}}
}

func TestInputEditing(t *testing.T) {
	r := New(Options{Out: &bytes.Buffer{}, Width: 40, Height: 5})
	in := renderables.NewInput(renderables.InputOptions{})
	r.Root().Add(in)
	r.RenderOnce()
	r.SetFocus(in)

	for _, ch := range "hello" {
		r.HandleEvent(runeKey(ch))
	}
	if in.String() != "hello" {
		t.Fatalf("value = %q", in.String())
	}
	r.HandleEvent(key("left"))
	r.HandleEvent(key("left"))
	r.HandleEvent(runeKey('X'))
	if in.String() != "helXlo" {
		t.Fatalf("after insert value = %q", in.String())
	}
	r.HandleEvent(key("backspace"))
	if in.String() != "hello" {
		t.Fatalf("after backspace value = %q", in.String())
	}
	r.HandleEvent(key("home"))
	r.HandleEvent(key("delete"))
	if in.String() != "ello" {
		t.Fatalf("after delete value = %q", in.String())
	}
	r.RenderOnce()
	if got := r.Buffer().Text(); !contains(got, "ello") {
		t.Fatalf("buffer = %q", got)
	}
}

func TestSelectNavigation(t *testing.T) {
	r := New(Options{Out: &bytes.Buffer{}, Width: 40, Height: 10})
	s := renderables.NewSelect(renderables.SelectOptions{
		Options: []renderables.SelectOption{{Name: "one"}, {Name: "two"}, {Name: "three"}},
	})
	r.Root().Add(s)
	r.RenderOnce()
	r.SetFocus(s)
	r.HandleEvent(key("down"))
	if s.Selected != 1 {
		t.Fatalf("selected = %d", s.Selected)
	}
	r.HandleEvent(key("end"))
	if s.Selected != 2 {
		t.Fatalf("selected after end = %d", s.Selected)
	}
	r.HandleEvent(key("up"))
	if s.Selected != 1 {
		t.Fatalf("selected after up = %d", s.Selected)
	}
}

func TestScrollBoxKeyboard(t *testing.T) {
	r := New(Options{Out: &bytes.Buffer{}, Width: 40, Height: 6})
	sb := renderables.NewScrollBox(renderables.ScrollBoxOptions{Focusable: true})
	for i := 0; i < 50; i++ {
		sb.Content.Add(renderables.NewText("row"))
	}
	r.Root().Add(sb)
	r.RenderOnce()
	r.SetFocus(sb)
	r.HandleEvent(key("pagedown"))
	if sb.ScrollY == 0 {
		t.Fatalf("did not scroll")
	}
	r.HandleEvent(key("home"))
	if sb.ScrollY != 0 {
		t.Fatalf("home scroll = %d", sb.ScrollY)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
