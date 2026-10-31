package width

import "testing"

func TestStringWidth(t *testing.T) {
	cases := map[string]int{
		"":        0,
		"abc":     3,
		"世界":      4,
		"é":       1,
		"e\u0301": 1, // e + combining acute
		"🙂":       2,
		"👍🏽":      2,
		"\t":      0,
	}
	for in, want := range cases {
		if got := StringWidth(in); got != want {
			t.Errorf("StringWidth(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestGraphemes(t *testing.T) {
	gs := Graphemes("a👍🏽é")
	if len(gs) != 3 {
		t.Fatalf("graphemes = %d, want 3: %+v", len(gs), gs)
	}
	if gs[1].Text != "👍🏽" || gs[1].Width != 2 {
		t.Errorf("emoji = %+v", gs[1])
	}
}

func TestClusters(t *testing.T) {
	cs := Clusters("a🙂b")
	if len(cs) != 3 {
		t.Fatalf("clusters = %d", len(cs))
	}
	if cs[1].Start != 1 || cs[1].End != 5 {
		t.Errorf("emoji offsets = %d..%d", cs[1].Start, cs[1].End)
	}
	if cs[2].Start != 5 {
		t.Errorf("b start = %d", cs[2].Start)
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("abcdef", 3); got != "abc" {
		t.Errorf("Truncate = %q", got)
	}
	if got := Truncate("世界abc", 3); got != "世" {
		// 世 is 2 cells, the next glyph is 2 and would straddle the boundary.
		t.Errorf("Truncate wide = %q", got)
	}
	if got := TruncateEllipsis("abcdef", 4); got != "abc…" {
		t.Errorf("TruncateEllipsis = %q", got)
	}
	if got := TruncateEllipsis("abc", 4); got != "abc" {
		t.Errorf("TruncateEllipsis no-op = %q", got)
	}
}

func TestSliceAndPad(t *testing.T) {
	if got := Slice("abcdef", 2, 3); got != "cde" {
		t.Errorf("Slice = %q", got)
	}
	if got := PadRight("ab", 4); got != "ab  " {
		t.Errorf("PadRight = %q", got)
	}
	if got := PadLeft("ab", 4); got != "  ab" {
		t.Errorf("PadLeft = %q", got)
	}
}

func TestFirst(t *testing.T) {
	c, w, rest := First("👍🏽x")
	if c != "👍🏽" || w != 2 || rest != "x" {
		t.Errorf("First = %q,%d,%q", c, w, rest)
	}
}
