// Package layout implements a Yoga-compatible subset of flexbox for terminal cells.
package layout

import (
	"math"
	"strconv"
	"strings"
)

// Unit is the unit of a layout value.
type Unit uint8

const (
	Undefined Unit = iota
	Point
	Percent
	UnitAuto
)

// Value is a length in cells, a percentage, auto, or undefined.
type Value struct {
	Unit  Unit
	Value float64
}

// P returns a point value.
func P(v int) Value { return Value{Unit: Point, Value: float64(v)} }

// Pf returns a fractional point value.
func Pf(v float64) Value { return Value{Unit: Point, Value: v} }

// Pct returns a percentage value.
func Pct(v float64) Value { return Value{Unit: Percent, Value: v} }

// Auto is an automatic value.
var Auto = Value{Unit: UnitAuto}
var Undef = Value{Unit: Undefined}

// IsDefinite reports whether the value resolves without a parent.
func (v Value) IsDefinite() bool { return v.Unit == Point }

// Resolve resolves a value against a reference length. ok is false for auto/undefined.
func (v Value) Resolve(ref float64, okRef bool) (float64, bool) {
	switch v.Unit {
	case Point:
		return v.Value, true
	case Percent:
		if okRef {
			return v.Value / 100 * ref, true
		}
	}
	return 0, false
}

// ParseValue parses "auto", "50%", or a plain number.
func ParseValue(s string) Value {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "undefined" {
		return Undef
	}
	if s == "auto" {
		return Auto
	}
	if strings.HasSuffix(s, "%") {
		if f, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64); err == nil {
			return Pct(f)
		}
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return Pf(f)
	}
	return Undef
}

// Direction is the main axis direction.
type Direction uint8

const (
	Column Direction = iota
	ColumnReverse
	Row
	RowReverse
)

func (d Direction) IsRow() bool { return d == Row || d == RowReverse }

// Wrap controls multi-line flex layout.
type Wrap uint8

const (
	NoWrap Wrap = iota
	WrapOn
	WrapReverse
)

// Justify controls main-axis distribution.
type Justify uint8

const (
	JustifyStart Justify = iota
	JustifyEnd
	JustifyCenter
	JustifySpaceBetween
	JustifySpaceAround
	JustifySpaceEvenly
)

// Align controls cross-axis alignment.
type Align uint8

const (
	AlignAuto Align = iota
	AlignStart
	AlignEnd
	AlignCenter
	AlignStretch
	AlignBaseline
)

// PositionType is relative or absolute.
type PositionType uint8

const (
	Relative PositionType = iota
	Absolute
)

// Display is flex or none.
type Display uint8

const (
	Flex Display = iota
	None
)

// Edges holds four sides in top,right,bottom,left order.
type Edges [4]int

// Horizontal returns left+right.
func (e Edges) Horizontal() int { return e[1] + e[3] }

// Vertical returns top+bottom.
func (e Edges) Vertical() int { return e[0] + e[2] }

// Style is a node's layout style.
type Style struct {
	Display      Display
	Position     PositionType
	Direction    Direction
	Wrap         Wrap
	Justify      Justify
	AlignItems   Align
	AlignSelf    Align
	AlignContent Align

	FlexGrow   float64
	FlexShrink float64
	FlexBasis  Value

	Width, Height       Value
	MinWidth, MinHeight Value
	MaxWidth, MaxHeight Value

	Top, Right, Bottom, Left Value

	Margin  [4]Value
	Padding [4]Value
	Border  [4]float64

	GapRow, GapCol Value
}

// DefaultStyle returns the flexbox defaults.
func DefaultStyle() Style {
	return Style{
		Display: Flex, Position: Relative, Direction: Column, Wrap: NoWrap,
		FlexShrink: 1,
	}
}

// MeasureFunc returns the content size for an available box. Width is clamped
// to availW when availW >= 0; a negative value means unconstrained.
type MeasureFunc func(availW, availH int) (int, int)

// Node is a layout tree node.
type Node struct {
	Style    Style
	Children []*Node
	Parent   *Node
	Measure  MeasureFunc
	// BeforeLayout runs at the start of arrange; AfterLayout runs after
	// children are arranged.
	BeforeLayout func(*Node)
	AfterLayout  func(*Node)

	// Computed layout in cells.
	X, Y, W, H int
	// Content box (inside padding and border).
	ContentX, ContentY, ContentW, ContentH int
	// Painting offset applied after layout (used by scroll containers).
	OffsetX, OffsetY int
	// Whether the node was laid out.
	done bool
}

// Add appends children.
func (n *Node) Add(children ...*Node) {
	for _, c := range children {
		if c == nil {
			continue
		}
		c.Parent = n
		n.Children = append(n.Children, c)
	}
}

// Remove detaches a child.
func (n *Node) Remove(child *Node) {
	for i, c := range n.Children {
		if c == child {
			n.Children = append(n.Children[:i], n.Children[i+1:]...)
			child.Parent = nil
			return
		}
	}
}

// Compute lays out the tree within the given terminal size and returns the root rect.
func (n *Node) Compute(width, height int) Rect {
	n.arrange(0, 0, width, height, -1, -1)
	return Rect{X: n.X, Y: n.Y, W: n.W, H: n.H}
}

// Rect is an integer rectangle in terminal cells.
type Rect struct{ X, Y, W, H int }

// Right returns X+W.
func (r Rect) Right() int { return r.X + r.W }

// Bottom returns Y+H.
func (r Rect) Bottom() int { return r.Y + r.H }

// Contains reports whether x,y is inside the rectangle.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H
}

func (n *Node) arrange(x, y, availW, availH, forcedW, forcedH int) {
	if n.Style.Display == None {
		n.X, n.Y, n.W, n.H = x, y, 0, 0
		return
	}
	if n.BeforeLayout != nil {
		n.BeforeLayout(n)
	}
	// Root/available box for this node includes its margins; the node itself is
	// placed inside them. Percent sizes resolve against the parent content box,
	// which is what availW/availH represent.
	m := resolveEdges(n.Style.Margin, availW, availH)
	pad := resolveSpacing(n.Style.Padding, availW, availH)
	bor := borderEdges(n.Style.Border)
	extraH := pad.Horizontal() + bor.Horizontal()
	extraV := pad.Vertical() + bor.Vertical()

	outerW := availW - m.Horizontal()
	if outerW < 0 {
		outerW = 0
	}
	outerH := availH - m.Vertical()
	if outerH < 0 {
		outerH = 0
	}

	w, h := n.resolveSize(outerW, outerH, extraH, extraV)
	if forcedW >= 0 {
		w = forcedW
	}
	if forcedH >= 0 {
		h = forcedH
	}

	// Clamp to available space when the node is a flex item with auto size.
	if w > outerW && outerW > 0 && !n.Style.Width.IsDefinite() && n.Style.Width.Unit != Percent {
		w = outerW
	}
	if h > outerH && outerH > 0 && !n.Style.Height.IsDefinite() && n.Style.Height.Unit != Percent {
		h = outerH
	}

	n.X = x + m[3] + n.OffsetX
	n.Y = y + m[0] + n.OffsetY
	n.W = maxInt(0, w)
	n.H = maxInt(0, h)

	n.ContentX = n.X + bor[3] + pad[3]
	n.ContentY = n.Y + bor[0] + pad[0]
	n.ContentW = maxInt(0, n.W-extraH)
	n.ContentH = maxInt(0, n.H-extraV)
	n.done = true

	n.layoutChildren()
	if n.AfterLayout != nil {
		n.AfterLayout(n)
	}
}

func (n *Node) resolveSize(outerW, outerH, extraH, extraV int) (int, int) {
	// Width.
	var w int
	if v, ok := n.Style.Width.Resolve(float64(outerW), outerW > 0); ok {
		w = int(math.Round(v))
	} else if n.Measure != nil {
		mw, _ := n.Measure(maxInt(0, outerW-extraH), -1)
		w = mw + extraH
	} else if len(n.Children) == 0 {
		w = outerW
	} else {
		w = outerW
	}
	// Height.
	var h int
	if v, ok := n.Style.Height.Resolve(float64(outerH), outerH > 0); ok {
		h = int(math.Round(v))
	} else if n.Measure != nil {
		_, mh := n.Measure(maxInt(0, w-extraH), maxInt(0, outerH-extraV))
		h = mh + extraV
	} else if len(n.Children) == 0 {
		h = outerH
	} else {
		h = n.measureChildrenHeight(maxInt(0, w-extraH)) + extraV
	}
	w = clampSize(w, n.Style.MinWidth, n.Style.MaxWidth, outerW)
	h = clampSize(h, n.Style.MinHeight, n.Style.MaxHeight, outerH)
	return w, h
}

func (n *Node) measureChildrenHeight(innerW int) int {
	dir := n.Style.Direction
	gap := 0
	if dir.IsRow() {
		gap = resolveGap(n.Style.GapCol, innerW)
	} else {
		gap = resolveGap(n.Style.GapRow, innerW)
	}
	var total int
	count := 0
	if dir.IsRow() {
		maxH := 0
		for _, c := range n.Children {
			if c.Style.Display == None || c.Style.Position == Absolute {
				continue
			}
			_, ch := c.measureFor(innerW, 0)
			if ch > maxH {
				maxH = ch
			}
			count++
		}
		total = maxH
	} else {
		for _, c := range n.Children {
			if c.Style.Display == None || c.Style.Position == Absolute {
				continue
			}
			_, ch := c.measureFor(innerW, 0)
			total += ch
			if count > 0 {
				total += gap
			}
			count++
		}
	}
	return total
}

// minMainSize returns the smallest main-axis size a child may shrink to.
func (n *Node) minMainSize(row bool, natural int) int {
	if row {
		if v, ok := n.Style.MinWidth.Resolve(float64(natural), true); ok {
			return int(math.Round(v))
		}
	} else {
		if v, ok := n.Style.MinHeight.Resolve(float64(natural), true); ok {
			return int(math.Round(v))
		}
	}
	if n.Measure != nil {
		return natural
	}
	return 0
}

// measureFor returns the outer size a child wants for the given available content box.
func (n *Node) measureFor(availW, availH int) (int, int) {
	m := resolveEdges(n.Style.Margin, availW, availH)
	pad := resolveSpacing(n.Style.Padding, availW, availH)
	bor := borderEdges(n.Style.Border)
	extraH := pad.Horizontal() + bor.Horizontal()
	extraV := pad.Vertical() + bor.Vertical()
	w, h := n.resolveSize(maxInt(0, availW-m.Horizontal()), maxInt(0, availH-m.Vertical()), extraH, extraV)
	return w + m.Horizontal(), h + m.Vertical()
}

func (n *Node) layoutChildren() {
	if len(n.Children) == 0 {
		return
	}
	dir := n.Style.Direction
	row := dir.IsRow()
	reverse := dir == RowReverse || dir == ColumnReverse

	mainSize := n.ContentW
	if !row {
		mainSize = n.ContentH
	}
	gap := resolveGap(n.Style.GapCol, n.ContentW)
	if !row {
		gap = resolveGap(n.Style.GapRow, n.ContentH)
	}

	type item struct {
		node    *Node
		main    int
		cross   int
		grow    float64
		shrink  float64
		minMain int
		marginM Edges // margins resolved for this child
	}
	var items []*item
	for _, c := range n.Children {
		if c.Style.Display == None {
			c.X, c.Y, c.W, c.H = 0, 0, 0, 0
			continue
		}
		if c.Style.Position == Absolute {
			continue
		}
		m := resolveEdges(c.Style.Margin, n.ContentW, n.ContentH)
		cw, ch := c.measureFor(n.ContentW, n.ContentH)
		it := &item{node: c, grow: c.Style.FlexGrow, shrink: c.Style.FlexShrink, marginM: m}
		if row {
			it.main = cw
			it.cross = ch
		} else {
			it.main = ch
			it.cross = cw
		}
		// Flex basis overrides the main base size.
		if v, ok := c.Style.FlexBasis.Resolve(float64(mainSize), true); ok {
			it.main = int(math.Round(v)) + m.Horizontal()
			if !row {
				it.main = int(math.Round(v)) + m.Vertical()
			}
		}
		it.minMain = c.minMainSize(row, it.main)
		items = append(items, it)
	}
	if len(items) == 0 {
		n.layoutAbsolute()
		return
	}

	total := 0
	for i, it := range items {
		total += it.main
		if i > 0 {
			total += gap
		}
	}
	free := mainSize - total
	// Grow / shrink along the main axis.
	if free > 0 {
		var growSum float64
		for _, it := range items {
			growSum += it.grow
		}
		if growSum > 0 {
			remaining := free
			for i, it := range items {
				if it.grow <= 0 {
					continue
				}
				share := int(math.Floor(float64(free) * it.grow / growSum))
				if i == len(items)-1 {
					share = remaining
				}
				if share > remaining {
					share = remaining
				}
				it.main += share
				remaining -= share
			}
			free = 0
		}
	} else if free < 0 {
		var shrinkSum float64
		for _, it := range items {
			shrinkSum += it.shrink * float64(maxInt(1, it.main))
		}
		if shrinkSum > 0 {
			over := -free
			for _, it := range items {
				if it.shrink <= 0 || over <= 0 {
					continue
				}
				want := int(math.Ceil(float64(over) * (it.shrink * float64(maxInt(1, it.main))) / shrinkSum))
				if room := it.main - it.minMain; want > room {
					want = room
				}
				if want < 0 {
					want = 0
				}
				if want > over {
					want = over
				}
				it.main -= want
				over -= want
			}
		}
		_ = free
	}

	// Justify offsets.
	start := 0
	between := gap
	extra := mainSize
	for i, it := range items {
		extra -= it.main
		if i > 0 {
			extra -= gap
		}
	}
	switch n.Style.Justify {
	case JustifyEnd:
		start = maxInt(0, extra)
	case JustifyCenter:
		start = maxInt(0, extra/2)
	case JustifySpaceBetween:
		if len(items) > 1 {
			between = gap + maxInt(0, extra/(len(items)-1))
		}
	case JustifySpaceAround:
		if len(items) > 0 {
			padEach := maxInt(0, extra/len(items))
			start = padEach / 2
			between = gap + padEach
		}
	case JustifySpaceEvenly:
		if len(items) > 0 {
			padEach := maxInt(0, extra/(len(items)+1))
			start = padEach
			between = gap + padEach
		}
	}

	cursor := start
	order := make([]*item, len(items))
	copy(order, items)
	if reverse {
		for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
			order[i], order[j] = order[j], order[i]
		}
	}
	for _, it := range order {
		c := it.node
		m := it.marginM
		var mainStart, crossStart, cw, ch int
		if row {
			cw = maxInt(0, it.main-m.Horizontal())
			ch = it.cross - m.Vertical()
			mainStart = n.ContentX + cursor + m[3]
			crossStart = n.ContentY + m[0]
			// Cross-axis alignment.
			crossStart += crossOffset(n.Style.AlignItems, c.Style.AlignSelf, n.ContentH, it.cross, m)
			// Re-measure height for the final width when no explicit height.
			if !c.Style.Height.IsDefinite() && c.Style.Height.Unit != Percent {
				_, ch2 := c.measureFor(cw, n.ContentH)
				if ch2 > ch && ch2-m.Vertical() > 0 {
					ch = ch2 - m.Vertical()
				}
			}
		} else {
			cw = it.cross - m.Horizontal()
			ch = maxInt(0, it.main-m.Vertical())
			crossStart = n.ContentX + m[3]
			crossStart += crossOffset(n.Style.AlignItems, c.Style.AlignSelf, n.ContentW, it.cross, m)
			mainStart = n.ContentY + cursor + m[0]
			if !c.Style.Width.IsDefinite() && c.Style.Width.Unit != Percent {
				cw2, _ := c.measureFor(n.ContentW, ch)
				if cw2 > 0 {
					cw = cw2 - m.Horizontal()
				}
			}
		}
		// Stretch cross axis.
		align := effectiveAlign(n.Style.AlignItems, c.Style.AlignSelf)
		if align == AlignStretch && !explicitCross(c, row) {
			if row {
				ch = n.ContentH - m.Vertical()
			} else {
				cw = n.ContentW - m.Horizontal()
			}
		}
		if row {
			c.arrange(mainStart, crossStart, maxInt(0, cw+m.Horizontal()), maxInt(0, ch+m.Vertical()), maxInt(0, cw+m.Horizontal()), maxInt(0, ch+m.Vertical()))
		} else {
			c.arrange(crossStart, mainStart, maxInt(0, cw+m.Horizontal()), maxInt(0, ch+m.Vertical()), maxInt(0, cw+m.Horizontal()), maxInt(0, ch+m.Vertical()))
		}
		cursor += it.main + between
	}
	n.layoutAbsolute()
}

func (n *Node) layoutAbsolute() {
	for _, c := range n.Children {
		if c.Style.Display == None || c.Style.Position != Absolute {
			continue
		}
		m := resolveEdges(c.Style.Margin, n.ContentW, n.ContentH)
		cw, ch := c.measureFor(n.ContentW, n.ContentH)
		x, y := n.ContentX+m[3], n.ContentY+m[0]
		if v, ok := c.Style.Left.Resolve(float64(n.ContentW), true); ok {
			x = n.ContentX + int(math.Round(v)) + m[3]
		} else if v, ok := c.Style.Right.Resolve(float64(n.ContentW), true); ok {
			x = n.ContentX + n.ContentW - int(math.Round(v)) - cw + m[3]
		}
		if v, ok := c.Style.Top.Resolve(float64(n.ContentH), true); ok {
			y = n.ContentY + int(math.Round(v)) + m[0]
		} else if v, ok := c.Style.Bottom.Resolve(float64(n.ContentH), true); ok {
			y = n.ContentY + n.ContentH - int(math.Round(v)) - ch + m[0]
		}
		c.arrange(x, y, maxInt(0, cw-m.Horizontal()), maxInt(0, ch-m.Vertical()), maxInt(0, cw-m.Horizontal()), maxInt(0, ch-m.Vertical()))
	}
}

func explicitCross(c *Node, row bool) bool {
	if row {
		return c.Style.Height.IsDefinite() || c.Style.Height.Unit == Percent
	}
	return c.Style.Width.IsDefinite() || c.Style.Width.Unit == Percent
}

func effectiveAlign(parent, self Align) Align {
	if self != AlignAuto {
		return self
	}
	if parent == AlignAuto {
		return AlignStretch
	}
	return parent
}

func crossOffset(parentAlign, selfAlign Align, crossSize, itemOuter int, m Edges) int {
	align := effectiveAlign(parentAlign, selfAlign)
	avail := crossSize - itemOuter
	if avail <= 0 {
		return 0
	}
	switch align {
	case AlignEnd:
		return avail
	case AlignCenter:
		return avail / 2
	default:
		return 0
	}
}

func resolveEdges(v [4]Value, refW, refH int) Edges {
	var e Edges
	for i := 0; i < 4; i++ {
		ref := float64(refW)
		if i%2 == 0 {
			ref = float64(refH)
		}
		if r, ok := v[i].Resolve(ref, true); ok {
			e[i] = int(math.Round(r))
		}
	}
	return e
}

func resolveSpacing(v [4]Value, refW, refH int) Edges {
	var e Edges
	for i := 0; i < 4; i++ {
		ref := float64(refW)
		if i%2 == 0 {
			ref = float64(refH)
		}
		if r, ok := v[i].Resolve(ref, true); ok {
			e[i] = maxInt(0, int(math.Round(r)))
		}
	}
	return e
}

func borderEdges(v [4]float64) Edges {
	return Edges{int(math.Round(v[0])), int(math.Round(v[1])), int(math.Round(v[2])), int(math.Round(v[3]))}
}

func resolveGap(v Value, ref int) int {
	if r, ok := v.Resolve(float64(ref), true); ok {
		return maxInt(0, int(math.Round(r)))
	}
	return 0
}

func clampSize(v int, minV, maxV Value, ref int) int {
	if r, ok := minV.Resolve(float64(ref), ref > 0); ok {
		if v < int(math.Round(r)) {
			v = int(math.Round(r))
		}
	}
	if r, ok := maxV.Resolve(float64(ref), ref > 0); ok {
		if v > int(math.Round(r)) {
			v = int(math.Round(r))
		}
	}
	return maxInt(0, v)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
