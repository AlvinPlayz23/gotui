package text

import "testing"

func TestGraphemeNavigation(t *testing.T) {
	b := NewTextBuffer("a👍🏽b")
	// a=0..1, thumbs up with skin tone = 1..9, b=9..10
	if got := b.NextGrapheme(0); got != 1 {
		t.Errorf("NextGrapheme(0) = %d, want 1", got)
	}
	if got := b.NextGrapheme(1); got != 9 {
		t.Errorf("NextGrapheme(1) = %d, want 9", got)
	}
	if got := b.PrevGrapheme(10); got != 9 {
		t.Errorf("PrevGrapheme(10) = %d, want 9", got)
	}
	if got := b.PrevGrapheme(9); got != 1 {
		t.Errorf("PrevGrapheme(9) = %d, want 1", got)
	}
}

func TestGraphemeAcrossLines(t *testing.T) {
	b := NewTextBuffer("ab\ncd")
	if got := b.NextGrapheme(2); got != 3 {
		t.Errorf("NextGrapheme(line end) = %d, want 3", got)
	}
	if got := b.PrevGrapheme(3); got != 2 {
		t.Errorf("PrevGrapheme(line start) = %d, want 2", got)
	}
}

func TestWordBoundaries(t *testing.T) {
	b := NewTextBuffer("hello brave  world")
	if got := b.WordBoundaryLeft(8); got != 6 {
		t.Errorf("WordBoundaryLeft(8) = %d, want 6", got)
	}
	if got := b.WordBoundaryRight(0); got != 5 {
		t.Errorf("WordBoundaryRight(0) = %d, want 5", got)
	}
	if got := b.WordBoundaryRight(5); got != 11 {
		t.Errorf("WordBoundaryRight(5) = %d, want 11", got)
	}
}

func TestLineWidthAndOffsets(t *testing.T) {
	b := NewTextBuffer("ab\tc\nd")
	if got := b.LineWidth(0); got != 5 {
		t.Errorf("LineWidth(0) = %d, want 5", got)
	}
	if got := b.ExpandedLine(0); got != "ab  c" {
		t.Errorf("ExpandedLine = %q", got)
	}
	b.TabWidth = 8
	if got := b.LineWidth(0); got != 9 {
		t.Errorf("LineWidth with tab 8 = %d, want 9", got)
	}
}

func TestClampOffset(t *testing.T) {
	b := NewTextBuffer("é")
	// "é" is two bytes; offset 1 is mid-rune.
	if got := b.ClampOffset(1); got != 0 {
		t.Errorf("ClampOffset(1) = %d, want 0", got)
	}
	if got := b.ClampOffset(2); got != 2 {
		t.Errorf("ClampOffset(2) = %d, want 2", got)
	}
}
