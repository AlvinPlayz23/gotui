package ansi

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bijim/gotui/cell"
	"github.com/bijim/gotui/color"
)

func newBuffer(rows ...string) *cell.Buffer {
	w := 0
	for _, r := range rows {
		if len(r) > w {
			w = len(r)
		}
	}
	b := cell.New(w, len(rows), color.RGBA{})
	for y, r := range rows {
		b.SetString(0, y, r, color.RGBA{}, color.RGBA{}, 0)
	}
	return b
}

func TestEncoderFirstFrame(t *testing.T) {
	var out bytes.Buffer
	enc := NewEncoder(&out, EncoderOptions{ColorMode: ColorTrue})
	buf := newBuffer("abc", "def")
	if err := enc.Render(buf); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "abc") || !strings.Contains(got, "def") {
		t.Fatalf("output = %q", got)
	}
	if !strings.Contains(got, CursorTo(1, 1)) || !strings.Contains(got, CursorTo(1, 2)) {
		t.Fatalf("missing cursor moves: %q", got)
	}
	if !strings.HasSuffix(got, Reset) && !strings.Contains(got, Reset) {
		t.Fatalf("missing reset: %q", got)
	}
}

func TestEncoderSkipsUnchangedFrames(t *testing.T) {
	var out bytes.Buffer
	enc := NewEncoder(&out, EncoderOptions{ColorMode: ColorTrue})
	buf := newBuffer("abc", "def")
	_ = enc.Render(buf)
	out.Reset()
	_ = enc.Render(buf)
	if strings.Contains(out.String(), "abc") || strings.Contains(out.String(), "def") {
		t.Fatalf("unchanged frame repainted rows: %q", out.String())
	}
}

func TestEncoderDiffsRows(t *testing.T) {
	var out bytes.Buffer
	enc := NewEncoder(&out, EncoderOptions{ColorMode: ColorTrue})
	buf := newBuffer("abc", "def")
	_ = enc.Render(buf)
	out.Reset()
	buf.SetString(1, 0, "X", color.RGBA{}, color.RGBA{}, 0)
	_ = enc.Render(buf)
	got := out.String()
	if !strings.Contains(got, "X") {
		t.Fatalf("changed cell not written: %q", got)
	}
	if !strings.Contains(got, CursorTo(2, 1)) {
		t.Fatalf("cursor not moved to changed cell: %q", got)
	}
	if strings.Contains(got, "def") {
		t.Fatalf("unchanged row repainted: %q", got)
	}
}

func TestEncoderTrueColor(t *testing.T) {
	var out bytes.Buffer
	enc := NewEncoder(&out, EncoderOptions{ColorMode: ColorTrue})
	buf := cell.New(1, 1, color.RGBA{})
	buf.SetCell(0, 0, cell.Cell{
		Ch: 'x', Width: 1,
		FG: color.FromInts(255, 0, 0, 255),
		BG: color.FromInts(0, 0, 255, 255),
	})
	_ = enc.Render(buf)
	got := out.String()
	if !strings.Contains(got, "38;2;255;0;0") {
		t.Fatalf("missing truecolor fg: %q", got)
	}
	if !strings.Contains(got, "48;2;0;0;255") {
		t.Fatalf("missing truecolor bg: %q", got)
	}
}

func TestEncoderWideGrapheme(t *testing.T) {
	var out bytes.Buffer
	enc := NewEncoder(&out, EncoderOptions{ColorMode: ColorTrue})
	buf := cell.New(6, 1, color.RGBA{})
	buf.SetString(0, 0, "a世界", color.RGBA{}, color.RGBA{}, 0)
	_ = enc.Render(buf)
	got := out.String()
	if !strings.Contains(got, "a世界") {
		t.Fatalf("wide text = %q", got)
	}
	if strings.Count(got, "世") != 1 || strings.Count(got, "界") != 1 {
		t.Fatalf("duplicated wide cells: %q", got)
	}
}

func TestEncoderInvalidate(t *testing.T) {
	var out bytes.Buffer
	enc := NewEncoder(&out, EncoderOptions{})
	buf := newBuffer("abc")
	_ = enc.Render(buf)
	enc.Invalidate()
	out.Reset()
	_ = enc.Render(buf)
	if !strings.Contains(out.String(), "abc") {
		t.Fatalf("invalidate did not repaint: %q", out.String())
	}
}

func TestColorMode256(t *testing.T) {
	var out bytes.Buffer
	enc := NewEncoder(&out, EncoderOptions{ColorMode: Color256})
	buf := cell.New(1, 1, color.RGBA{})
	buf.Set(0, 0, 'x', color.FromInts(255, 0, 0, 255), color.RGBA{}, 0)
	_ = enc.Render(buf)
	if !strings.Contains(out.String(), "38;5;") {
		t.Fatalf("missing 256 color: %q", out.String())
	}
}
