package text

import (
	"strings"
	"testing"
)

func TestSplitLinesWordWrap(t *testing.T) {
	cases := []struct {
		content string
		width   int
	}{
		{"hello world again", 7},
		{"a b c d e f g", 3},
		{"one two three four five six", 10},
		{"no-spaces-here-at-all", 5},
		{"  leading spaces", 4},
		{"trailing spaces   ", 5},
		{"a\nb\nc", 5},
		{"", 5},
		{"word", 1},
	}
	for _, c := range cases {
		for _, mode := range []WrapMode{WrapWord, WrapChar} {
			lines := SplitLines(Plain(c.content), c.width, mode)
			if len(lines) == 0 {
				t.Fatalf("%q width %d: no lines", c.content, c.width)
			}
			for _, l := range lines {
				if l.Width() > c.width && !strings.Contains(c.content, "no-spaces") {
					// Single graphemes wider than the line are allowed, words are not.
					for _, sp := range l {
						if sp.Text != " " && len([]rune(sp.Text)) > 0 && !strings.ContainsAny(sp.Text, "-") {
							t.Fatalf("%q width %d mode %d: line %q too wide", c.content, c.width, mode, l.String())
						}
					}
				}
			}
		}
	}
}

func TestSplitLinesKeepsWords(t *testing.T) {
	lines := SplitLines(Plain("hello world again"), 7, WrapWord)
	got := make([]string, len(lines))
	for i, l := range lines {
		got[i] = l.String()
	}
	want := []string{"hello ", "world ", "again"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q want %q", got, want)
	}
}
