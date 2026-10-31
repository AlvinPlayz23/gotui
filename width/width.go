// Package width implements Unicode grapheme clustering and terminal display widths.
package width

import (
	"strings"

	"github.com/rivo/uniseg"
)

// Method selects how display width is computed.
type Method int

const (
	// WCWidth approximates classic wcwidth behavior.
	WCWidth Method = iota
	// Unicode uses Unicode grapheme cluster widths.
	Unicode
	// UnicodeWide treats East Asian ambiguous characters as wide.
	UnicodeWide
)

// Grapheme is one user-perceived character and its display width.
type Grapheme struct {
	Text  string
	Width int
}

// StringWidth returns the display width of s in terminal cells.
func StringWidth(s string) int { return uniseg.StringWidth(s) }

// RuneWidth returns the display width of a single rune.
func RuneWidth(r rune) int {
	if r == 0 {
		return 0
	}
	return uniseg.StringWidth(string(r))
}

// Graphemes splits s into user-perceived characters with widths.
func Graphemes(s string) []Grapheme {
	if s == "" {
		return nil
	}
	var out []Grapheme
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		out = append(out, Grapheme{Text: g.Str(), Width: g.Width()})
	}
	return out
}

// Iterate calls fn for each grapheme cluster. Returning false stops iteration.
func Iterate(s string, fn func(Grapheme) bool) {
	if s == "" {
		return
	}
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		if !fn(Grapheme{Text: g.Str(), Width: g.Width()}) {
			return
		}
	}
}

// Cluster is one grapheme cluster with its byte range in the source string.
type Cluster struct {
	Text  string
	Width int
	Start int
	End   int
}

// Clusters splits s into grapheme clusters with byte offsets.
func Clusters(s string) []Cluster {
	if s == "" {
		return nil
	}
	var out []Cluster
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		from, to := g.Positions()
		out = append(out, Cluster{Text: g.Str(), Width: g.Width(), Start: from, End: to})
	}
	return out
}

// IteratePos calls fn for each grapheme cluster with its byte offsets.
func IteratePos(s string, fn func(Cluster) bool) {
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		from, to := g.Positions()
		if !fn(Cluster{Text: g.Str(), Width: g.Width(), Start: from, End: to}) {
			return
		}
	}
}

// First returns the first grapheme cluster, its width, and the remainder.
func First(s string) (cluster string, w int, rest string) {
	if s == "" {
		return "", 0, ""
	}
	g := uniseg.NewGraphemes(s)
	if !g.Next() {
		return "", 0, ""
	}
	_, to := g.Positions()
	return g.Str(), g.Width(), s[to:]
}

// Truncate returns the prefix of s that fits in max cells. A wide character
// that would straddle the boundary is dropped.
func Truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if StringWidth(s) <= max {
		return s
	}
	var b strings.Builder
	used := 0
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		w := g.Width()
		if used+w > max {
			break
		}
		b.WriteString(g.Str())
		used += w
	}
	return b.String()
}

// TruncateEllipsis truncates s to max cells, appending "…" when cut.
func TruncateEllipsis(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if StringWidth(s) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return Truncate(s, max-1) + "…"
}

// Slice returns the substring of s covering the half-open cell range [start, start+width).
func Slice(s string, start, width int) string {
	if width <= 0 {
		return ""
	}
	var b strings.Builder
	col := 0
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		w := g.Width()
		if col+w <= start {
			col += w
			continue
		}
		if col >= start+width {
			break
		}
		if col >= start && col+w <= start+width {
			b.WriteString(g.Str())
		}
		col += w
	}
	return b.String()
}

// PadRight pads s with spaces to width cells.
func PadRight(s string, width int) string {
	w := StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// PadLeft left-pads s with spaces to width cells.
func PadLeft(s string, width int) string {
	w := StringWidth(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}
