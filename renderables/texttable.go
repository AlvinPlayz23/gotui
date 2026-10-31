package renderables

import (
	"strings"

	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
	"github.com/bijim/gotui/text"
	"github.com/bijim/gotui/width"
)

// TableAlign controls horizontal alignment in a table column.
type TableAlign uint8

const (
	TableAlignLeft TableAlign = iota
	TableAlignCenter
	TableAlignRight
)

// TableColumn optionally sets a column title, width, and alignment.
type TableColumn struct {
	Title    string
	Width    int
	MinWidth int
	MaxWidth int
	Align    TableAlign
}

// TextTableOptions configures a terminal table.
type TextTableOptions struct {
	ID          string
	Headers     []string
	Rows        [][]string
	Columns     []TableColumn
	Border      bool
	FG          color.RGBA
	BG          color.RGBA
	HeaderFG    color.RGBA
	HeaderBG    color.RGBA
	BorderColor color.RGBA
	CellPadding int
	Style       layout.Style
}

// TextTable renders aligned, optionally wrapped tabular data.
type TextTable struct {
	*renderable.Base
	Headers            []string
	Rows               [][]string
	Columns            []TableColumn
	Border             bool
	FG, BG             color.RGBA
	HeaderFG, HeaderBG color.RGBA
	BorderColor        color.RGBA
	CellPadding        int
}

func NewTextTable(opts TextTableOptions) *TextTable {
	if opts.Style == (layout.Style{}) {
		opts.Style = layout.DefaultStyle()
	}
	if opts.FG == (color.RGBA{}) {
		opts.FG = White
	}
	if opts.HeaderFG == (color.RGBA{}) {
		opts.HeaderFG = opts.FG
	}
	if opts.BorderColor == (color.RGBA{}) {
		opts.BorderColor = Gray
	}
	if opts.CellPadding < 0 {
		opts.CellPadding = 0
	}
	columns := append([]TableColumn(nil), opts.Columns...)
	count := len(columns)
	if len(opts.Headers) > count {
		count = len(opts.Headers)
	}
	for _, row := range opts.Rows {
		if len(row) > count {
			count = len(row)
		}
	}
	for len(columns) < count {
		columns = append(columns, TableColumn{})
	}
	for i := range columns {
		if i < len(opts.Headers) && columns[i].Title == "" {
			columns[i].Title = opts.Headers[i]
		}
		if columns[i].MinWidth < 0 {
			columns[i].MinWidth = 0
		}
		if columns[i].MaxWidth > 0 && columns[i].MinWidth > columns[i].MaxWidth {
			columns[i].MinWidth = columns[i].MaxWidth
		}
	}
	t := &TextTable{Headers: append([]string(nil), opts.Headers...), Rows: cloneRows(opts.Rows), Columns: columns,
		Border: opts.Border, FG: opts.FG, BG: opts.BG, HeaderFG: opts.HeaderFG, HeaderBG: opts.HeaderBG,
		BorderColor: opts.BorderColor, CellPadding: opts.CellPadding}
	t.Base = renderable.NewBase(t, &layout.Node{Style: opts.Style})
	t.ID = opts.ID
	t.Node.Measure = t.measure
	return t
}

func cloneRows(rows [][]string) [][]string {
	out := make([][]string, len(rows))
	for i := range rows {
		out[i] = append([]string(nil), rows[i]...)
	}
	return out
}

func (t *TextTable) SetRows(rows [][]string) {
	t.Rows = cloneRows(rows)
	t.Invalidate()
	t.RequestRender()
}

func (t *TextTable) columnCount() int { return len(t.Columns) }

func (t *TextTable) naturalWidths() []int {
	w := make([]int, len(t.Columns))
	for i, col := range t.Columns {
		if col.Width > 0 {
			w[i] = col.Width
		} else {
			w[i] = 1
		}
		if col.Width <= 0 && col.Title != "" {
			w[i] = maxInt(w[i], width.StringWidth(col.Title))
		}
		if col.Width <= 0 {
			for _, row := range t.Rows {
				if i < len(row) {
					w[i] = maxInt(w[i], width.StringWidth(row[i]))
				}
			}
		}
		if col.MinWidth > 0 {
			w[i] = maxInt(w[i], col.MinWidth)
		}
		if col.MaxWidth > 0 {
			w[i] = minInt(w[i], col.MaxWidth)
		}
		w[i] = maxInt(1, w[i]+2*t.CellPadding)
	}
	return w
}

func (t *TextTable) widths(avail int) []int {
	w := t.naturalWidths()
	if !t.Border {
		return w
	}
	limit := avail - len(w) - 1
	for sumWidths(w) > limit {
		idx := -1
		for i := range w {
			if w[i] > 1 && (idx < 0 || w[i] > w[idx]) {
				idx = i
			}
		}
		if idx < 0 {
			break
		}
		w[idx]--
	}
	return w
}

func sumWidths(w []int) int {
	n := 0
	for _, v := range w {
		n += v
	}
	return n
}

func (t *TextTable) measuredRows(widths []int) ([][][]text.Line, int) {
	all := make([][]string, 0, len(t.Rows)+1)
	if len(t.Headers) > 0 || hasColumnTitle(t.Columns) {
		head := make([]string, len(t.Columns))
		for i, c := range t.Columns {
			head[i] = c.Title
			if i < len(t.Headers) && head[i] == "" {
				head[i] = t.Headers[i]
			}
		}
		all = append(all, head)
	}
	all = append(all, t.Rows...)
	result := make([][][]text.Line, len(all))
	height := 0
	innerWidth := func(i int) int { return maxInt(1, widths[i]-2*t.CellPadding) }
	for r, row := range all {
		result[r] = make([][]text.Line, len(t.Columns))
		rowHeight := 1
		for c := range t.Columns {
			value := ""
			if c < len(row) {
				value = row[c]
			}
			lines := text.SplitLines(text.Plain(value), innerWidth(c), text.WrapWord)
			if len(lines) == 0 {
				lines = []text.Line{{}}
			}
			result[r][c] = lines
			rowHeight = maxInt(rowHeight, len(lines))
		}
		height += rowHeight
		if t.Border {
			height++
		}
	}
	if t.Border && len(t.Columns) > 0 {
		if len(all) == 0 {
			height = 2
		} else {
			height++
		}
	}
	return result, height
}

func hasColumnTitle(cols []TableColumn) bool {
	for _, c := range cols {
		if c.Title != "" {
			return true
		}
	}
	return false
}

func (t *TextTable) measure(availW, _ int) (int, int) {
	if len(t.Columns) == 0 {
		return 0, 0
	}
	var w []int
	if availW <= 0 {
		w = t.naturalWidths()
	} else {
		w = t.widths(availW)
	}
	rows, h := t.measuredRows(w)
	_ = rows
	total := sumWidths(w)
	if t.Border && len(w) > 0 {
		total += len(w) + 1
	}
	if availW <= 0 {
		return total, h
	}
	return minInt(availW, total), h
}

func (t *TextTable) Paint(ctx *renderable.Context) {
	if !t.Visible || len(t.Columns) == 0 {
		return
	}
	r := t.ContentBounds()
	ctx = ctx.WithClip(r)
	w := t.widths(r.W)
	rows, _ := t.measuredRows(w)
	y := r.Y
	rowIndex := 0
	if t.Border && len(w) > 0 {
		t.paintBorder(ctx, r.X, y, w, '┌', '┬', '┐', '─')
		y++
	}
	for rowIdx, cells := range rows {
		if y >= r.Bottom() {
			break
		}
		rowHeight := 1
		for _, lines := range cells {
			if len(lines) > 0 {
				rowHeight = maxInt(rowHeight, len(lines))
			}
		}
		x := r.X
		if t.Border {
			ctx.SetString(x, y, "│", t.BorderColor, t.BG, 0)
			x++
		}
		for col, colWidth := range w {
			if t.Border && col > 0 {
				ctx.SetString(x, y, "│", t.BorderColor, t.BG, 0)
				x++
			}
			for lineNo := 0; lineNo < rowHeight && y+lineNo < r.Bottom(); lineNo++ {
				var line text.Line
				if col < len(cells) && lineNo < len(cells[col]) {
					line = cells[col][lineNo]
				}
				contentW := maxInt(0, colWidth-2*t.CellPadding)
				textW := line.Width()
				pad := maxInt(0, contentW-textW)
				align := t.Columns[col].Align
				left := t.CellPadding
				if align == TableAlignRight {
					left += pad
				} else if align == TableAlignCenter {
					left += pad / 2
				}
				if lineNo == 0 {
					ctx.SetString(x, y+lineNo, strings.Repeat(" ", left), t.FG, t.BG, 0)
				}
				fg, bg := t.FG, t.BG
				if rowIndex == 0 && (len(t.Headers) > 0 || hasColumnTitle(t.Columns)) {
					fg, bg = t.HeaderFG, t.HeaderBG
				}
				paintLine(ctx, x+left, x+colWidth-t.CellPadding, y+lineNo, line, fg, bg)
			}
			x += colWidth
		}
		if t.Border {
			ctx.SetString(x, y, "│", t.BorderColor, t.BG, 0)
		}
		y += rowHeight
		if t.Border && rowIdx+1 < len(rows) {
			t.paintBorder(ctx, r.X, y, w, '├', '┼', '┤', '─')
			y++
		}
		rowIndex++
	}
	if t.Border && y < r.Bottom() {
		t.paintBorder(ctx, r.X, y, w, '└', '┴', '┘', '─')
	}
}

func (t *TextTable) paintBorder(ctx *renderable.Context, x, y int, widths []int, left, mid, right, fill rune) {
	if y < ctx.Clip.Y || y >= ctx.Clip.Bottom() {
		return
	}
	cur := x
	ctx.SetString(cur, y, string(left), t.BorderColor, t.BG, 0)
	cur++
	for i, w := range widths {
		ctx.SetString(cur, y, strings.Repeat(string(fill), w), t.BorderColor, t.BG, 0)
		cur += w
		if i+1 < len(widths) {
			ctx.SetString(cur, y, string(mid), t.BorderColor, t.BG, 0)
			cur++
		}
	}
	ctx.SetString(cur, y, string(right), t.BorderColor, t.BG, 0)
}
