package tuitest

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/bijim/gotui/cell"
	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/renderables"
	"github.com/bijim/gotui/renderer"
)

func TestSnapshotCapturesStyle(t *testing.T) {
	buf := cell.New(2, 1, color.FromInts(1, 2, 3, 255))
	buf.Set(0, 0, 'A', color.FromInts(255, 0, 0, 255), color.RGBA{}, cell.AttrBold)
	f := Snapshot(buf)
	if f.Width != 2 || f.Height != 1 {
		t.Fatalf("frame = %dx%d", f.Width, f.Height)
	}
	if f.Rows[0][0].Text != "A" || f.Rows[0][0].Attr != uint16(cell.AttrBold) {
		t.Fatalf("cell = %+v", f.Rows[0][0])
	}
	if f.Rows[0][0].FG != "ff0000ff" {
		t.Fatalf("fg = %q", f.Rows[0][0].FG)
	}
	if err := json.Unmarshal([]byte(SnapshotJSON(buf)), &FrameSnapshot{}); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
}

func TestGoldenRoundTrip(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()

	AssertGolden(t, "frame", "hello\n")
	AssertGolden(t, "frame", "hello\n")
	AssertGolden(t, "other", "world\n")
}

func TestRenderGolden(t *testing.T) {
	r := renderer.New(renderer.Options{Out: &bytes.Buffer{}, Width: 20, Height: 4})
	box := renderables.NewBox(renderables.BoxOptions{
		Border: true,
		Title:  "hi",
	})
	box.Add(renderables.NewText("ok"))
	r.Root().Add(box)
	r.RenderOnce()
	AssertPlainGolden(t, "box_text", r.Buffer())
}
