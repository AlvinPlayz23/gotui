// Package tuitest provides golden-frame snapshots for renderer parity tests.
package tuitest

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bijim/gotui/cell"
)

var update = flag.Bool("update", false, "update golden files")

// CellSnapshot is a serializable cell for golden comparisons.
type CellSnapshot struct {
	Text  string `json:"text"`
	Width uint8  `json:"width"`
	FG    string `json:"fg"`
	BG    string `json:"bg"`
	Attr  uint16 `json:"attr,omitempty"`
	Link  uint32 `json:"link,omitempty"`
}

// FrameSnapshot is a serializable cell grid.
type FrameSnapshot struct {
	Width  int              `json:"width"`
	Height int              `json:"height"`
	Rows   [][]CellSnapshot `json:"rows"`
}

// Snapshot captures the buffer's cells, including colors and attributes.
func Snapshot(buf *cell.Buffer) FrameSnapshot {
	f := FrameSnapshot{Width: buf.W, Height: buf.H, Rows: make([][]CellSnapshot, buf.H)}
	for y := 0; y < buf.H; y++ {
		row := make([]CellSnapshot, buf.W)
		for x := 0; x < buf.W; x++ {
			c := buf.Cells[y*buf.W+x]
			row[x] = CellSnapshot{
				Text:  c.Glyph(),
				Width: c.Width,
				FG:    fmt.Sprintf("%08x", c.FG.Uint32()),
				BG:    fmt.Sprintf("%08x", c.BG.Uint32()),
				Attr:  uint16(c.Attr),
				Link:  c.Link,
			}
		}
		f.Rows[y] = row
	}
	return f
}

// SnapshotJSON marshals a snapshot into stable, indented JSON.
func SnapshotJSON(buf *cell.Buffer) string {
	return snapshotJSON(Snapshot(buf))
}

func snapshotJSON(f FrameSnapshot) string {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return ""
	}
	return string(data) + "\n"
}

// AssertGolden compares got against testdata/<name>.golden, creating or
// updating the file when -update is set or the file does not exist.
func AssertGolden(t testing.TB, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := writeFile(path, got); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := writeFile(path, got); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(want) != got {
		t.Fatalf("golden mismatch for %s\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
	}
}

// AssertPlainGolden compares buf's plain text against a golden file.
func AssertPlainGolden(t testing.TB, name string, buf *cell.Buffer) {
	t.Helper()
	AssertGolden(t, name, buf.Text()+"\n")
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
