package renderer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bijim/gotui/input"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderables"
)

func TestSliderKeyboardAndPaint(t *testing.T) {
	r := New(Options{Out: &bytes.Buffer{}, Width: 20, Height: 3})
	s := renderables.NewSlider(renderables.SliderOptions{Value: 0.5, Focusable: true})
	r.Root().Add(s)
	r.RenderOnce()
	r.SetFocus(s)
	r.HandleEvent(key("right"))
	if s.Value <= 0.5 {
		t.Fatalf("value = %v, want > .5", s.Value)
	}
	r.HandleEvent(key("home"))
	if s.Value != 0 {
		t.Fatalf("home value = %v", s.Value)
	}
	r.RenderOnce()
	if got := r.Buffer().Text(); !contains(got, "●") || !contains(got, "─") {
		t.Fatalf("buffer = %q", got)
	}
}

func TestSliderAbsolute(t *testing.T) {
	s := renderables.NewSlider(renderables.SliderOptions{Min: 10, Max: 20, Value: 0.5})
	if got := s.Absolute(); got != 15 {
		t.Fatalf("Absolute = %v, want 15", got)
	}
	s.SetAbsolute(20)
	if s.Value != 1 {
		t.Fatalf("Value = %v, want 1", s.Value)
	}
}

func TestScrollBarPaintAndKeys(t *testing.T) {
	r := New(Options{Out: &bytes.Buffer{}, Width: 20, Height: 10})
	sb := renderables.NewScrollBar(renderables.ScrollBarOptions{
		Vertical:    true,
		Value:       20,
		ContentSize: 100,
		Viewport:    10,
		Focusable:   true,
	})
	r.Root().Add(sb)
	r.RenderOnce()
	r.SetFocus(sb)
	r.HandleEvent(key("down"))
	if sb.Value != 21 {
		t.Fatalf("value = %d, want 21", sb.Value)
	}
	r.HandleEvent(key("end"))
	if sb.Value != 90 {
		t.Fatalf("end value = %d, want 90", sb.Value)
	}
	r.RenderOnce()
	if got := r.Buffer().Text(); !contains(got, "█") {
		t.Fatalf("buffer = %q", got)
	}
}

func TestASCIIFont(t *testing.T) {
	r := New(Options{Out: &bytes.Buffer{}, Width: 40, Height: 10})
	a := renderables.NewASCIIFont(renderables.ASCIIFontOptions{
		Text: "AB",
		FG:   renderables.White,
		Style: layout.Style{
			Direction: layout.Column,
			Width:     layout.P(7),
		},
	})
	r.Root().Add(a)
	r.RenderOnce()
	text := r.Buffer().Text()
	if !strings.Contains(text, "█") {
		t.Fatalf("buffer has no blocks:\n%s", text)
	}
	if a.LayoutNode().H != 5 {
		t.Fatalf("height = %d, want 5", a.LayoutNode().H)
	}
	// A 3x5 glyph plus one spacing column, two glyphs.
	if a.LayoutNode().W != 7 {
		t.Fatalf("width = %d, want 7", a.LayoutNode().W)
	}
}

func TestASCIIFontScale(t *testing.T) {
	r := New(Options{Out: &bytes.Buffer{}, Width: 40, Height: 12})
	a := renderables.NewASCIIFont(renderables.ASCIIFontOptions{
		Text:  "I",
		Scale: 2,
		Style: layout.Style{Direction: layout.Column, Width: layout.P(6)},
	})
	r.Root().Add(a)
	r.RenderOnce()
	if a.LayoutNode().H != 10 || a.LayoutNode().W != 6 {
		t.Fatalf("size = %dx%d, want 6x10", a.LayoutNode().W, a.LayoutNode().H)
	}
}

func TestSliderMouse(t *testing.T) {
	r := New(Options{Out: &bytes.Buffer{}, Width: 11, Height: 2})
	s := renderables.NewSlider(renderables.SliderOptions{Value: 0})
	r.Root().Add(s)
	r.RenderOnce()
	r.HandleEvent(mouseEvent(10, 0, input.MousePress))
	if s.Value <= 0.9 {
		t.Fatalf("value = %v", s.Value)
	}
}
