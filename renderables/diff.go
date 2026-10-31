package renderables

import (
	"fmt"
	"strings"

	"github.com/bijim/gotui/cell"
	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
	"github.com/bijim/gotui/width"
)

type diffKind uint8

const (
	diffEqual diffKind = iota
	diffDelete
	diffInsert
	diffHunk
	diffHeader
)

type diffRow struct {
	kind    diffKind
	oldLine int
	newLine int
	content string
}

// DiffOptions configures a unified line diff.
type DiffOptions struct {
	ID        string
	Old       string
	New       string
	OldLabel  string
	NewLabel  string
	Context   int
	FG        color.RGBA
	BG        color.RGBA
	AddedFG   color.RGBA
	AddedBG   color.RGBA
	RemovedFG color.RGBA
	RemovedBG color.RGBA
	HunkFG    color.RGBA
	Style     layout.Style
}

// Diff renders a unified diff with line numbers and change colors.
type Diff struct {
	*renderable.Base
	Old, New             string
	OldLabel, NewLabel   string
	Context              int
	FG, BG               color.RGBA
	AddedFG, AddedBG     color.RGBA
	RemovedFG, RemovedBG color.RGBA
	HunkFG               color.RGBA
	rows                 []diffRow
	valid                bool
}

func NewDiff(opts DiffOptions) *Diff {
	if opts.Style == (layout.Style{}) {
		opts.Style = layout.DefaultStyle()
	}
	if opts.Context < 0 {
		opts.Context = 0
	}
	if opts.FG == (color.RGBA{}) {
		opts.FG = White
	}
	if opts.AddedFG == (color.RGBA{}) {
		opts.AddedFG = color.FromInts(152, 195, 121, 255)
	}
	if opts.AddedBG == (color.RGBA{}) {
		opts.AddedBG = color.FromInts(26, 54, 36, 255)
	}
	if opts.RemovedFG == (color.RGBA{}) {
		opts.RemovedFG = color.FromInts(224, 108, 117, 255)
	}
	if opts.RemovedBG == (color.RGBA{}) {
		opts.RemovedBG = color.FromInts(66, 35, 40, 255)
	}
	if opts.HunkFG == (color.RGBA{}) {
		opts.HunkFG = color.FromInts(97, 175, 239, 255)
	}
	d := &Diff{Old: opts.Old, New: opts.New, OldLabel: opts.OldLabel, NewLabel: opts.NewLabel,
		Context: opts.Context, FG: opts.FG, BG: opts.BG, AddedFG: opts.AddedFG, AddedBG: opts.AddedBG,
		RemovedFG: opts.RemovedFG, RemovedBG: opts.RemovedBG, HunkFG: opts.HunkFG}
	if d.OldLabel == "" {
		d.OldLabel = "a"
	}
	if d.NewLabel == "" {
		d.NewLabel = "b"
	}
	d.Base = renderable.NewBase(d, &layout.Node{Style: opts.Style})
	d.ID = opts.ID
	d.Node.Measure = d.measure
	return d
}

func (d *Diff) SetDiff(oldText, newText string) {
	d.Old, d.New = oldText, newText
	d.valid = false
	d.Invalidate()
	d.RequestRender()
}

func splitKeepEmpty(s string) []string { return strings.Split(strings.TrimSuffix(s, "\n"), "\n") }

func (d *Diff) buildRows() []diffRow {
	if d.valid {
		return d.rows
	}
	oldLines, newLines := splitKeepEmpty(d.Old), splitKeepEmpty(d.New)
	if d.Old == "" {
		oldLines = nil
	}
	if d.New == "" {
		newLines = nil
	}
	ops := lineDiff(oldLines, newLines)
	if len(ops) == 0 {
		d.rows = []diffRow{{kind: diffHunk, content: "No changes"}}
		d.valid = true
		return d.rows
	}
	context := d.Context
	if context == 0 {
		context = 3
	}
	changed := make([]bool, len(ops))
	hasChanges := false
	for i, op := range ops {
		changed[i] = op.kind != diffEqual
		hasChanges = hasChanges || changed[i]
	}
	if !hasChanges {
		d.rows = []diffRow{{kind: diffHunk, content: "No changes"}}
		d.valid = true
		return d.rows
	}
	var ranges [][2]int
	for i, yes := range changed {
		if !yes {
			continue
		}
		lo, hi := maxInt(0, i-context), minInt(len(ops), i+context+1)
		if len(ranges) > 0 && lo <= ranges[len(ranges)-1][1] {
			if hi > ranges[len(ranges)-1][1] {
				ranges[len(ranges)-1][1] = hi
			}
		} else {
			ranges = append(ranges, [2]int{lo, hi})
		}
	}
	rows := []diffRow{{kind: diffHeader, content: "--- " + d.OldLabel}, {kind: diffHeader, content: "+++ " + d.NewLabel}}
	for _, span := range ranges {
		start := len(rows)
		oldStart, newStart := 0, 0
		for _, op := range ops[:span[0]] {
			if op.kind != diffInsert {
				oldStart++
			}
			if op.kind != diffDelete {
				newStart++
			}
		}
		oldCount, newCount := 0, 0
		for _, op := range ops[span[0]:span[1]] {
			if op.kind != diffInsert {
				oldCount++
			}
			if op.kind != diffDelete {
				newCount++
			}
			rows = append(rows, diffRow{kind: op.kind, oldLine: op.oldLine, newLine: op.newLine, content: op.text})
		}
		oldHunkStart, newHunkStart := oldStart+1, newStart+1
		if oldCount == 0 {
			oldHunkStart = oldStart
		}
		if newCount == 0 {
			newHunkStart = newStart
		}
		rows = append(rows, diffRow{kind: diffHunk, content: fmt.Sprintf("@@ -%d,%d +%d,%d @@", oldHunkStart, oldCount, newHunkStart, newCount)})
		// Keep the hunk header before its contents.
		hunk := rows[len(rows)-1]
		copy(rows[start+1:], rows[start:len(rows)-1])
		rows[start] = hunk
	}
	d.rows, d.valid = rows, true
	return rows
}

type lineOp struct {
	kind             diffKind
	oldLine, newLine int
	text             string
}

func lineDiff(a, b []string) []lineOp {
	n, m := len(a), len(b)
	if n*m > 2_000_000 {
		ops := make([]lineOp, 0, n+m)
		limit := maxInt(n, m)
		for i := 0; i < limit; i++ {
			if i < n && i < m && a[i] == b[i] {
				ops = append(ops, lineOp{diffEqual, i + 1, i + 1, a[i]})
				continue
			}
			if i < n {
				ops = append(ops, lineOp{diffDelete, i + 1, 0, a[i]})
			}
			if i < m {
				ops = append(ops, lineOp{diffInsert, 0, i + 1, b[i]})
			}
		}
		return ops
	}
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	ops := make([]lineOp, 0, n+m)
	i, j := 0, 0
	for i < n || j < m {
		if i < n && j < m && a[i] == b[j] {
			ops = append(ops, lineOp{diffEqual, i + 1, j + 1, a[i]})
			i++
			j++
		} else if i < n && (j == m || dp[i+1][j] >= dp[i][j+1]) {
			ops = append(ops, lineOp{diffDelete, i + 1, 0, a[i]})
			i++
		} else {
			ops = append(ops, lineOp{diffInsert, 0, j + 1, b[j]})
			j++
		}
	}
	return ops
}

func (d *Diff) measure(availW, _ int) (int, int) {
	rows := d.buildRows()
	w := 0
	for _, row := range rows {
		w = maxInt(w, width.StringWidth(diffRowText(row)))
	}
	return minInt(availW, w), len(rows)
}

func diffRowText(r diffRow) string {
	switch r.kind {
	case diffDelete:
		return fmt.Sprintf("%4d      - %s", r.oldLine, r.content)
	case diffInsert:
		return fmt.Sprintf("      %4d + %s", r.newLine, r.content)
	case diffEqual:
		return fmt.Sprintf("%4d %4d   %s", r.oldLine, r.newLine, r.content)
	default:
		return r.content
	}
}

func (d *Diff) Paint(ctx *renderable.Context) {
	if !d.Visible {
		return
	}
	r := d.ContentBounds()
	ctx = ctx.WithClip(r)
	for i, row := range d.buildRows() {
		y := r.Y + i
		if y >= r.Bottom() {
			break
		}
		fg, bg, attr := d.FG, d.BG, cell.Attribute(0)
		prefix := " "
		switch row.kind {
		case diffDelete:
			fg, bg, prefix = d.RemovedFG, d.RemovedBG, "-"
		case diffInsert:
			fg, bg, prefix = d.AddedFG, d.AddedBG, "+"
		case diffHunk:
			fg, prefix = d.HunkFG, ""
		case diffHeader:
			fg, prefix = d.HunkFG, ""
		}
		if row.kind == diffHunk || row.kind == diffHeader {
			ctx.SetString(r.X, y, row.content, fg, bg, cell.AttrBold)
			continue
		}
		oldNum, newNum := "", ""
		if row.oldLine > 0 {
			oldNum = fmt.Sprintf("%4d", row.oldLine)
		} else {
			oldNum = "    "
		}
		if row.newLine > 0 {
			newNum = fmt.Sprintf("%4d", row.newLine)
		} else {
			newNum = "    "
		}
		line := fmt.Sprintf("%s %s %s %s", oldNum, newNum, prefix, row.content)
		ctx.SetString(r.X, y, line, fg, bg, attr)
	}
}
