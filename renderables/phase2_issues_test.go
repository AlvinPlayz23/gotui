package renderables

import (
	"strings"
	"testing"

	"github.com/bijim/gotui/cell"
	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/renderable"
)

func TestDiffHunkHeadersAndRows(t *testing.T) {
	d := NewDiff(DiffOptions{Old: "a\nb\n", New: "a\nc\n"})
	rows := d.buildRows()
	if len(rows) < 6 || !strings.HasPrefix(rows[2].content, "@@ ") {
		t.Fatalf("missing hunk header: %#v", rows)
	}
	if rows[3].content != "a" || rows[4].kind != diffDelete || rows[5].kind != diffInsert {
		t.Fatalf("diff rows are out of order: %#v", rows)
	}

	added := NewDiff(DiffOptions{New: "first\n"}).buildRows()
	if added[2].content != "@@ -0,0 +1,1 @@" {
		t.Fatalf("bad add-to-empty hunk: %q", added[2].content)
	}
	deleted := NewDiff(DiffOptions{Old: "first\n"}).buildRows()
	if deleted[2].content != "@@ -1,1 +0,0 @@" {
		t.Fatalf("bad delete-to-empty hunk: %q", deleted[2].content)
	}
}

func TestLineNumberInheritsWidgetBackground(t *testing.T) {
	bg := color.FromInts(20, 30, 40, 255)
	widget := NewLineNumber("line", LineNumberOptions{BG: bg})
	if widget.NumberBG != bg {
		t.Fatalf("gutter background %v, want widget background %v", widget.NumberBG, bg)
	}
}

func TestCodePlainForegroundUsesCodeForeground(t *testing.T) {
	fg := color.FromInts(12, 34, 56, 255)
	widget := NewCode("ordinary", CodeOptions{FG: fg})
	lines := widget.lines(20)
	if len(lines) != 1 || len(lines[0]) == 0 || lines[0][0].FG != fg {
		t.Fatalf("plain code foreground = %#v, want %v", lines, fg)
	}
}

func TestTextTableSetRowsAddsColumnsAndFitsBorderlessWidth(t *testing.T) {
	table := NewTextTable(TextTableOptions{})
	table.SetRows([][]string{{"a fairly long cell", "second"}})
	if len(table.Columns) != 2 {
		t.Fatalf("SetRows created %d columns, want 2", len(table.Columns))
	}
	widths := table.widths(12)
	if sumWidths(widths) > 12 {
		t.Fatalf("borderless table width %d exceeds available 12", sumWidths(widths))
	}
}

func TestTextTableRespectsPaddingAndMinimumWidthWhenShrinking(t *testing.T) {
	table := NewTextTable(TextTableOptions{
		CellPadding: 1,
		Columns:     []TableColumn{{MinWidth: 2}, {MinWidth: 2}},
		Rows:        [][]string{{"long content", "more content"}},
	})
	widths := table.widths(10)
	for i, width := range widths {
		if width < 4 {
			t.Fatalf("column %d shrank below padding/minimum floor: %d", i, width)
		}
	}
}

func TestTextTablePaintsWrappedBordersAndCellBackgrounds(t *testing.T) {
	bg := color.FromInts(12, 30, 40, 255)
	table := NewTextTable(TextTableOptions{
		Columns: []TableColumn{{Width: 6}, {Width: 4}},
		Rows:    [][]string{{"abcdefgh", "z"}},
		Border:  true,
		BG:      bg,
	})
	table.Node.ContentX, table.Node.ContentY = 0, 0
	table.Node.ContentW, table.Node.ContentH = 13, 5
	buf := cell.New(13, 5, color.RGBA{})
	table.Paint(renderable.NewContext(buf, nil))
	for _, y := range []int{1, 2} {
		left, _ := buf.At(0, y)
		inner, _ := buf.At(7, y)
		right, _ := buf.At(12, y)
		if left.Ch != '│' || inner.Ch != '│' || right.Ch != '│' {
			t.Fatalf("missing vertical borders on wrapped visual line %d: %q %q %q", y, left.Glyph(), inner.Glyph(), right.Glyph())
		}
	}
	blank, _ := buf.At(11, 1)
	if blank.BG != bg {
		t.Fatalf("trailing cell background = %v, want %v", blank.BG, bg)
	}
}

func TestMarkdownStylesBlocksAndKeepsPlainText(t *testing.T) {
	headingFG := color.FromInts(220, 100, 80, 255)
	quoteFG := color.FromInts(80, 200, 140, 255)
	md := NewMarkdown(MarkdownOptions{Content: "# Heading\n> quote\nsnake_case\n", HeadingFG: headingFG, QuoteFG: quoteFG})
	styled := md.compiled()
	if strings.HasSuffix(styled.String(), "\n") {
		t.Fatalf("markdown has a synthetic trailing newline: %q", styled.String())
	}
	for _, want := range []struct {
		text string
		fg   color.RGBA
	}{{"Heading", headingFG}, {"quote", quoteFG}} {
		found := false
		for _, chunk := range styled.Chunks {
			if strings.Contains(chunk.Text, want.text) {
				found = chunk.FG == want.fg
			}
		}
		if !found {
			t.Errorf("%q did not use configured foreground %v", want.text, want.fg)
		}
	}
	for _, chunk := range styled.Chunks {
		if strings.Contains(chunk.Text, "snake_case") && chunk.Attr.Has(cell.AttrItalic) {
			t.Fatalf("snake_case was interpreted as emphasis: %#v", chunk)
		}
	}
}

func TestMarkdownFencesMalformedUTF8AndTableRules(t *testing.T) {
	content := "```\nfirst\n```not a closer\nlast\n```\n"
	md := NewMarkdown(MarkdownOptions{Content: content})
	if !strings.Contains(md.compiled().String(), "```not a closer") {
		t.Fatal("fence closed on a line with trailing text")
	}
	invalid := string([]byte{0xff, 'x'})
	md = NewMarkdown(MarkdownOptions{Content: invalid})
	if got := md.compiled().String(); got != invalid {
		t.Fatalf("malformed UTF-8 changed: %q", got)
	}
	md = NewMarkdown(MarkdownOptions{Content: "| alpha | beta |\n| foo-bar | ordinary |\n"})
	if !strings.Contains(md.compiled().String(), "foo-bar") {
		t.Fatal("pipe row with a hyphenated value was mistaken for a separator")
	}
}

func TestMarkdownFencedPlainTextUsesCodeForeground(t *testing.T) {
	fg := color.FromInts(210, 120, 40, 255)
	md := NewMarkdown(MarkdownOptions{Content: "```text\nplain text\n```", CodeFG: fg})
	for _, chunk := range md.compiled().Chunks {
		if strings.Contains(chunk.Text, "plain") {
			if chunk.FG != fg {
				t.Fatalf("fenced plain foreground = %v, want %v", chunk.FG, fg)
			}
			return
		}
	}
	t.Fatal("plain fenced code not found")
}
