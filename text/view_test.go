package text

import "testing"

func TestTextViewNoWrap(t *testing.T) {
	buf := NewTextBuffer("hello\nworld")
	v := NewTextView(buf)
	v.Resize(80, 10)
	lines := v.VisualLines()
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(lines))
	}
	if lines[0].String() != "hello" || lines[1].String() != "world" {
		t.Fatalf("lines = %q, %q", lines[0].String(), lines[1].String())
	}
	if lines[0].StartByte != 0 || lines[0].EndByte != 5 {
		t.Errorf("line 0 range = %d..%d", lines[0].StartByte, lines[0].EndByte)
	}
}

func TestTextViewCharWrap(t *testing.T) {
	buf := NewTextBuffer("abcdefg")
	v := NewTextView(buf)
	v.Wrap = WrapChar
	v.Resize(3, 10)
	lines := v.VisualLines()
	if len(lines) != 3 {
		t.Fatalf("lines = %d, want 3", len(lines))
	}
	want := []string{"abc", "def", "g"}
	for i, w := range want {
		if got := lines[i].String(); got != w {
			t.Errorf("line %d = %q, want %q", i, got, w)
		}
	}
	if lines[0].EndByte != 3 || lines[1].StartByte != 3 {
		t.Errorf("wrap ranges = %d..%d / %d..%d", lines[0].StartByte, lines[0].EndByte, lines[1].StartByte, lines[1].EndByte)
	}
}

func TestTextViewWordWrap(t *testing.T) {
	buf := NewTextBuffer("the quick brown fox")
	v := NewTextView(buf)
	v.Wrap = WrapWord
	v.Resize(10, 10)
	lines := v.VisualLines()
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2: %v", len(lines), lineStrings(lines))
	}
	if lines[0].String() != "the quick " || lines[1].String() != "brown fox" {
		t.Fatalf("lines = %q, %q", lines[0].String(), lines[1].String())
	}
}

func TestTextViewWrapLongWord(t *testing.T) {
	buf := NewTextBuffer("abcdefghij")
	v := NewTextView(buf)
	v.Wrap = WrapWord
	v.Resize(4, 10)
	lines := v.VisualLines()
	if len(lines) != 3 {
		t.Fatalf("lines = %d, want 3: %v", len(lines), lineStrings(lines))
	}
}

func TestTextViewTabs(t *testing.T) {
	buf := NewTextBuffer("a\tb")
	v := NewTextView(buf)
	v.Resize(20, 5)
	lines := v.VisualLines()
	if got := lines[0].String(); got != "a   b" {
		t.Fatalf("line = %q", got)
	}
	if lines[0].Width != 5 {
		t.Fatalf("width = %d, want 5", lines[0].Width)
	}
}

func TestTextViewScrollAndVisible(t *testing.T) {
	buf := NewTextBuffer("0\n1\n2\n3\n4\n5")
	v := NewTextView(buf)
	v.Resize(10, 2)
	if got := v.MaxScrollY(); got != 4 {
		t.Fatalf("MaxScrollY = %d, want 4", got)
	}
	v.ScrollTo(0, 3)
	vis := v.VisibleLines()
	if len(vis) != 2 || vis[0].Line != 3 || vis[1].Line != 4 {
		t.Fatalf("visible = %v", lineStrings(vis))
	}
}

func TestTextViewHitTest(t *testing.T) {
	buf := NewTextBuffer("hello\nworld")
	v := NewTextView(buf)
	v.Resize(20, 5)
	if line, col := v.HitTest(2, 1); line != 1 || col != 2 {
		t.Errorf("HitTest = %d,%d, want 1,2", line, col)
	}
	if line, col := v.HitTest(99, 0); line != 0 || col != 5 {
		t.Errorf("HitTest past end = %d,%d, want 0,5", line, col)
	}
}

func TestTextViewVisualIndex(t *testing.T) {
	buf := NewTextBuffer("abcdef")
	v := NewTextView(buf)
	v.Wrap = WrapChar
	v.Resize(2, 5)
	idx, x := v.VisualIndex(0, 3)
	if idx != 1 || x != 1 {
		t.Errorf("VisualIndex = %d,%d, want 1,1", idx, x)
	}
	// At a wrap boundary the caret belongs to the following line.
	idx, x = v.VisualIndex(0, 2)
	if idx != 1 || x != 0 {
		t.Errorf("VisualIndex boundary = %d,%d, want 1,0", idx, x)
	}
}

func TestTextViewSelectionCells(t *testing.T) {
	buf := NewTextBuffer("abc\ndef")
	v := NewTextView(buf)
	v.Resize(10, 5)
	cells := v.SelectionCells(1, 6)
	if len(cells) != 2 {
		t.Fatalf("cells = %+v", cells)
	}
	if cells[0].Line != 0 || cells[0].Start != 1 || cells[0].End != 3 {
		t.Errorf("line 0 selection = %+v", cells[0])
	}
	if cells[1].Line != 1 || cells[1].Start != 0 || cells[1].End != 2 {
		t.Errorf("line 1 selection = %+v", cells[1])
	}
}

func TestTextViewEnsureVisible(t *testing.T) {
	buf := NewTextBuffer("0\n1\n2\n3\n4\n5\n6\n7\n8\n9")
	v := NewTextView(buf)
	v.Resize(10, 3)
	v.EnsureVisible(7, 0, 0)
	if v.ScrollY != 5 {
		t.Fatalf("ScrollY = %d, want 5", v.ScrollY)
	}
	v.EnsureVisible(1, 0, 0)
	if v.ScrollY != 1 {
		t.Fatalf("ScrollY = %d, want 1", v.ScrollY)
	}
}

func lineStrings(lines []VisualLine) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.String()
	}
	return out
}
