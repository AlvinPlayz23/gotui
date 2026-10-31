package text

import (
	"strings"
	"testing"
)

func TestRopeBasicEdits(t *testing.T) {
	r := NewRope("hello world")
	if got := r.Len(); got != 11 {
		t.Fatalf("Len() = %d, want 11", got)
	}
	r.Insert(5, ",")
	if got := r.String(); got != "hello, world" {
		t.Fatalf("after insert = %q", got)
	}
	r.Delete(5, 6)
	if got := r.String(); got != "hello world" {
		t.Fatalf("after delete = %q", got)
	}
	r.Replace(0, 5, "goodbye")
	if got := r.String(); got != "goodbye world" {
		t.Fatalf("after replace = %q", got)
	}
	r.Insert(0, ">> ")
	r.Insert(r.Len(), " <<")
	if got := r.String(); got != ">> goodbye world <<" {
		t.Fatalf("after ends = %q", got)
	}
	if got := r.Slice(3, 10); got != "goodbye" {
		t.Fatalf("Slice = %q", got)
	}
}

func TestRopeLines(t *testing.T) {
	r := NewRope("one\ntwo\n\nfour")
	if got := r.LineCount(); got != 4 {
		t.Fatalf("LineCount = %d, want 4", got)
	}
	want := []string{"one", "two", "", "four"}
	for i, w := range want {
		if got := r.Line(i); got != w {
			t.Errorf("Line(%d) = %q, want %q", i, got, w)
		}
	}
	if line, col := r.Position(6); line != 1 || col != 2 {
		t.Errorf("Position(6) = %d,%d, want 1,2", line, col)
	}
	if got := r.Offset(3, 2); got != 11 {
		t.Errorf("Offset(3,2) = %d, want 11", got)
	}
	// Byte columns clamp to the line end.
	if got := r.Offset(1, 99); got != 7 {
		t.Errorf("Offset(1,99) = %d, want 7", got)
	}
}

func TestRopeTrailingNewline(t *testing.T) {
	r := NewRope("a\n")
	if got := r.LineCount(); got != 2 {
		t.Fatalf("LineCount = %d, want 2", got)
	}
	if got := r.Line(1); got != "" {
		t.Fatalf("Line(1) = %q", got)
	}
	r.Insert(2, "b")
	if got := r.String(); got != "a\nb" {
		t.Fatalf("String = %q", got)
	}
	if got := r.LineCount(); got != 2 {
		t.Fatalf("LineCount = %d, want 2", got)
	}
}

func TestRopeUnicode(t *testing.T) {
	r := NewRope("héllo 🙂 世界")
	// Insert inside the multi-byte accented rune boundary is allowed at byte
	// offsets; the rope never splits runes on its own.
	r.Insert(len("héllo"), "!")
	if got := r.String(); got != "héllo! 🙂 世界" {
		t.Fatalf("String = %q", got)
	}
	if got := r.Line(0); !strings.Contains(got, "世界") {
		t.Fatalf("Line = %q", got)
	}
}

func TestRopeManyEdits(t *testing.T) {
	r := NewRope("")
	var want strings.Builder
	const n = 500
	for i := 0; i < n; i++ {
		s := "x"
		r.Insert(r.Len(), s)
		want.WriteString(s)
	}
	for i := 0; i < n/2; i++ {
		r.Delete(0, 1)
	}
	got := r.String()
	if got != want.String()[n/2:] {
		t.Fatalf("content mismatch: %d bytes", len(got))
	}
}

func TestExpandTabs(t *testing.T) {
	cases := []struct {
		in       string
		tab, col int
		want     string
	}{
		{"a\tb", 4, 0, "a   b"},
		{"\tx", 4, 0, "    x"},
		{"\tx", 4, 1, "   x"},
		{"ab\tc\td", 8, 0, "ab      c       d"},
	}
	for _, c := range cases {
		if got := ExpandTabs(c.in, c.tab, c.col); got != c.want {
			t.Errorf("ExpandTabs(%q,%d,%d) = %q, want %q", c.in, c.tab, c.col, got, c.want)
		}
	}
}

func TestDisplayClustersTabs(t *testing.T) {
	cs := DisplayClusters("a\tb", 4)
	if len(cs) != 3 {
		t.Fatalf("len = %d, want 3", len(cs))
	}
	if cs[1].Width != 3 || cs[1].Text != "   " {
		t.Errorf("tab = %+v", cs[1])
	}
	if cs[1].ByteStart != 1 || cs[1].ByteEnd != 2 {
		t.Errorf("tab bytes = %d..%d", cs[1].ByteStart, cs[1].ByteEnd)
	}
}
