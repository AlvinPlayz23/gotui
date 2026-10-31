package renderer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderables"
)

func TestRenderBoxAndText(t *testing.T) {
	var out bytes.Buffer
	r := New(Options{Out: &out, Width: 20, Height: 5})
	b := renderables.NewBox(renderables.BoxOptions{
		Border: true,
		Style:  layout.Style{Direction: layout.Column, FlexGrow: 1},
	})
	b.Add(renderables.NewText("hello", renderables.TextOptions{
		Style: layout.Style{Direction: layout.Column},
	}))
	r.Root().Add(b)
	r.RenderOnce()

	if got := r.Buffer().Text(); !strings.Contains(got, "hello") {
		t.Fatalf("buffer missing text: %q", got)
	}
	got := out.String()
	if !strings.Contains(got, "hello") {
		t.Fatalf("output missing text: %q", got)
	}
	if !strings.Contains(got, "┌") {
		t.Fatalf("output missing border: %q", got)
	}
}

func TestRenderDiffOnly(t *testing.T) {
	var out bytes.Buffer
	r := New(Options{Out: &out, Width: 10, Height: 3})
	r.Root().Add(renderables.NewText("abc"))
	r.RenderOnce()
	first := out.Len()
	r.RenderOnce()
	if out.Len() <= first {
		t.Fatalf("second frame should still emit cursor bookkeeping")
	}
	if strings.Count(out.String(), "abc") != 1 {
		t.Fatalf("unchanged text should only be emitted once, got %q", out.String())
	}
}
