// Package renderable provides the renderable tree and painting primitives.
package renderable

import (
	"sort"

	"github.com/bijim/gotui/cell"
	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/input"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/width"
)

// Renderer is the renderer surface widgets may use.
type Renderer interface {
	RequestRender()
	SetFocus(Renderable)
	Width() int
	Height() int
	SetCursor(x, y int, visible bool)
}

// Renderable is a node in the UI tree.
type Renderable interface {
	BaseNode() *Base
	LayoutNode() *layout.Node
	Paint(ctx *Context)
	HitTest(x, y int) Renderable
	HandleKey(ev *input.KeyEvent) bool
	HandleKeyUp(ev *input.KeyEvent) bool
	HandleMouse(ev *input.MouseEvent) bool
	HandlePaste(ev *input.PasteEvent) bool
}

// Context carries the frame buffer and clip for painting.
type Context struct {
	Buffer   *cell.Buffer
	Clip     layout.Rect
	Width    int
	Height   int
	Renderer Renderer
	Time     float64
}

// NewContext creates a paint context.
func NewContext(buf *cell.Buffer, r Renderer) *Context {
	return &Context{Buffer: buf, Clip: layout.Rect{W: buf.W, H: buf.H}, Width: buf.W, Height: buf.H, Renderer: r}
}

// WithClip returns a child context intersected with r.
func (c *Context) WithClip(r layout.Rect) *Context {
	clip := intersect(c.Clip, r)
	nc := *c
	nc.Clip = clip
	return &nc
}

// Fill fills a rectangle with bg, clipped.
func (c *Context) Fill(x, y, w, h int, bg color.RGBA) {
	r := intersect(c.Clip, layout.Rect{X: x, Y: y, W: w, H: h})
	if r.W <= 0 || r.H <= 0 {
		return
	}
	c.Buffer.Fill(r.X, r.Y, r.W, r.H, bg)
}

// SetString draws text at x,y clipped.
func (c *Context) SetString(x, y int, s string, fg, bg color.RGBA, attr cell.Attribute) int {
	if y < c.Clip.Y || y >= c.Clip.Bottom() {
		return 0
	}
	return c.Buffer.SetStringClipped(x, y, c.Clip.X, c.Clip.W, s, fg, bg, attr)
}

// DrawBox draws a border rectangle clipped.
func (c *Context) DrawBox(x, y, w, h int, sides cell.BorderSides, chars cell.BorderCharacters, fg, bg color.RGBA, attr cell.Attribute) {
	// Simple approach: clip by drawing only cells inside the clip.
	draw := func(px, py int, r rune) {
		if r != 0 && c.Clip.Contains(px, py) {
			c.Buffer.Set(px, py, r, fg, bg, attr)
		}
	}
	right, bottom := x+w-1, y+h-1
	if sides&cell.SideTop != 0 {
		for px := x + 1; px < right; px++ {
			draw(px, y, chars.Top)
		}
	}
	if sides&cell.SideBottom != 0 {
		for px := x + 1; px < right; px++ {
			draw(px, bottom, chars.Bottom)
		}
	}
	if sides&cell.SideLeft != 0 {
		for py := y + 1; py < bottom; py++ {
			draw(x, py, chars.Left)
		}
	}
	if sides&cell.SideRight != 0 {
		for py := y + 1; py < bottom; py++ {
			draw(right, py, chars.Right)
		}
	}
	if sides&cell.SideTop != 0 && sides&cell.SideLeft != 0 {
		draw(x, y, chars.TopLeft)
	}
	if sides&cell.SideTop != 0 && sides&cell.SideRight != 0 {
		draw(right, y, chars.TopRight)
	}
	if sides&cell.SideBottom != 0 && sides&cell.SideLeft != 0 {
		draw(x, bottom, chars.BottomLeft)
	}
	if sides&cell.SideBottom != 0 && sides&cell.SideRight != 0 {
		draw(right, bottom, chars.BottomRight)
	}
}

// Base is embedded by every renderable.
type Base struct {
	ID        string
	Node      *layout.Node
	Visible   bool
	ZIndex    int
	Focusable bool
	Focused   bool

	OnKeyDown     func(*input.KeyEvent) bool
	OnKeyUp       func(*input.KeyEvent) bool
	OnMouse       func(*input.MouseEvent) bool
	OnMouseDown   func(*input.MouseEvent) bool
	OnMouseUp     func(*input.MouseEvent) bool
	OnMouseMove   func(*input.MouseEvent) bool
	OnMouseDrag   func(*input.MouseEvent) bool
	OnScroll      func(*input.MouseEvent) bool
	OnClick       func(*input.MouseEvent)
	OnDoubleClick func(*input.MouseEvent)
	OnMouseOver   func(*input.MouseEvent)
	OnMouseOut    func(*input.MouseEvent)
	OnPaste       func(*input.PasteEvent) bool
	OnFocus       func(bool)
	OnResize      func(w, h int)
	OnUpdate      func(dt float64) bool

	Renderer Renderer
	self     Renderable
	parent   Renderable
	children []Renderable
	lastW    int
	lastH    int
	dirty    bool
}

// NewBase binds a renderable to its base state.
func NewBase(self Renderable, node *layout.Node) *Base {
	if node == nil {
		node = &layout.Node{Style: layout.DefaultStyle()}
	}
	return &Base{self: self, Node: node, Visible: true}
}

// BaseNode returns the embedded base.
func (b *Base) BaseNode() *Base { return b }

// LayoutNode returns the layout node.
func (b *Base) LayoutNode() *layout.Node { return b.Node }

// Self returns the owning renderable.
func (b *Base) Self() Renderable { return b.self }

// SetSelf rebinds the owning renderable.
func (b *Base) SetSelf(self Renderable) { b.self = self }

// Parent returns the parent renderable.
func (b *Base) Parent() Renderable { return b.parent }

// Children returns the child renderables.
func (b *Base) Children() []Renderable { return b.children }

// Add appends children.
func (b *Base) Add(children ...Renderable) {
	for _, c := range children {
		if c == nil {
			continue
		}
		cb := c.BaseNode()
		cb.parent = b.self
		cb.Renderer = b.Renderer
		b.children = append(b.children, c)
		b.Node.Add(c.LayoutNode())
	}
	b.Invalidate()
}

// InsertBefore inserts child before anchor.
func (b *Base) InsertBefore(child, anchor Renderable) {
	if child == nil {
		return
	}
	idx := -1
	for i, c := range b.children {
		if c == anchor {
			idx = i
			break
		}
	}
	if idx < 0 {
		b.Add(child)
		return
	}
	child.BaseNode().parent = b.self
	child.BaseNode().Renderer = b.Renderer
	b.children = append(b.children, nil)
	copy(b.children[idx+1:], b.children[idx:])
	b.children[idx] = child
	// Rebuild layout children to match order.
	b.rebuildLayoutChildren()
	b.Invalidate()
}

// Remove detaches a child.
func (b *Base) Remove(child Renderable) {
	for i, c := range b.children {
		if c == child {
			b.children = append(b.children[:i], b.children[i+1:]...)
			child.BaseNode().parent = nil
			b.rebuildLayoutChildren()
			b.Invalidate()
			return
		}
	}
}

// RemoveAll detaches every child.
func (b *Base) RemoveAll() {
	for _, c := range b.children {
		c.BaseNode().parent = nil
	}
	b.children = nil
	b.Node.Children = nil
	b.Invalidate()
}

func (b *Base) rebuildLayoutChildren() {
	b.Node.Children = b.Node.Children[:0]
	for _, c := range b.children {
		b.Node.Add(c.LayoutNode())
	}
}

// GetRenderable finds a descendant by id.
func (b *Base) GetRenderable(id string) Renderable {
	if b.ID == id {
		return b.self
	}
	for _, c := range b.children {
		if found := c.BaseNode().GetRenderable(id); found != nil {
			return found
		}
	}
	return nil
}

// Bounds returns the computed layout rect.
func (b *Base) Bounds() layout.Rect {
	return layout.Rect{X: b.Node.X, Y: b.Node.Y, W: b.Node.W, H: b.Node.H}
}

// ContentBounds returns the rect inside padding and border.
func (b *Base) ContentBounds() layout.Rect {
	return layout.Rect{X: b.Node.ContentX, Y: b.Node.ContentY, W: b.Node.ContentW, H: b.Node.ContentH}
}

// Invalidate marks the renderable for a re-render.
func (b *Base) Invalidate() {
	b.dirty = true
	if b.self != nil && b.self.BaseNode() != nil && b.self.BaseNode().parent != nil {
		b.self.BaseNode().parent.BaseNode().Invalidate()
	}
}

// RequestRender asks the renderer for a new frame.
func (b *Base) RequestRender() {
	if b.Renderer != nil {
		b.Renderer.RequestRender()
	}
}

// Paint draws the renderable. The default paints children.
func (b *Base) Paint(ctx *Context) {
	if !b.Visible {
		return
	}
	b.PaintChildren(ctx)
}

// PaintChildren paints visible children sorted by z-index.
func (b *Base) PaintChildren(ctx *Context) {
	kids := make([]Renderable, 0, len(b.children))
	for _, c := range b.children {
		if c.BaseNode().Visible {
			kids = append(kids, c)
		}
	}
	if len(kids) > 1 {
		sort.SliceStable(kids, func(i, j int) bool {
			return kids[i].BaseNode().ZIndex < kids[j].BaseNode().ZIndex
		})
	}
	for _, c := range kids {
		c.Paint(ctx)
	}
}

// Update advances animations, returning true if a redraw is needed.
func (b *Base) Update(dt float64) bool {
	dirty := false
	if b.OnUpdate != nil && b.OnUpdate(dt) {
		dirty = true
	}
	for _, c := range b.children {
		if c.BaseNode().Update(dt) {
			dirty = true
		}
	}
	return dirty
}

// SetFocused updates focus state and notifies listeners.
func (b *Base) SetFocused(v bool) {
	if b.Focused == v {
		return
	}
	b.Focused = v
	if b.OnFocus != nil {
		b.OnFocus(v)
	}
	b.Invalidate()
	b.RequestRender()
}

// HandleKey dispatches a key event.
func (b *Base) HandleKey(ev *input.KeyEvent) bool {
	if b.OnKeyDown != nil {
		return b.OnKeyDown(ev)
	}
	return false
}

// HandleKeyUp dispatches a key release.
func (b *Base) HandleKeyUp(ev *input.KeyEvent) bool {
	if b.OnKeyUp != nil {
		return b.OnKeyUp(ev)
	}
	return false
}

// HandlePaste dispatches a paste event.
func (b *Base) HandlePaste(ev *input.PasteEvent) bool {
	if b.OnPaste != nil {
		return b.OnPaste(ev)
	}
	return false
}

// HandleMouse dispatches a mouse event.
func (b *Base) HandleMouse(ev *input.MouseEvent) bool {
	if b.OnMouse != nil && b.OnMouse(ev) {
		return true
	}
	switch ev.Action {
	case input.MousePress:
		if b.OnMouseDown != nil {
			return b.OnMouseDown(ev)
		}
	case input.MouseRelease:
		if b.OnMouseUp != nil {
			return b.OnMouseUp(ev)
		}
	case input.MouseMove:
		if b.OnMouseMove != nil {
			return b.OnMouseMove(ev)
		}
	case input.MouseWheel:
		if b.OnScroll != nil {
			return b.OnScroll(ev)
		}
	}
	return false
}

// HitTest returns the deepest visible renderable containing x,y.
func (b *Base) HitTest(x, y int) Renderable {
	if !b.Visible {
		return nil
	}
	// Prefer topmost (highest z, last) children.
	best := Renderable(nil)
	bestZ := -1 << 30
	for _, c := range b.children {
		if !c.BaseNode().Visible {
			continue
		}
		if hit := c.HitTest(x, y); hit != nil {
			if c.BaseNode().ZIndex >= bestZ {
				best = hit
				bestZ = c.BaseNode().ZIndex
			}
		}
	}
	if best != nil {
		return best
	}
	if b.Bounds().Contains(x, y) {
		return b.self
	}
	return nil
}

// Resize notifies the renderable of a size change.
func (b *Base) Resize(w, h int) {
	if b.lastW == w && b.lastH == h {
		return
	}
	b.lastW, b.lastH = w, h
	if b.OnResize != nil {
		b.OnResize(w, h)
	}
	for _, c := range b.children {
		c.BaseNode().Resize(c.LayoutNode().W, c.LayoutNode().H)
	}
}

// MeasureText returns (min(content width), wrapped height) for text at availW.
func MeasureText(s string, availW int, wrap bool) (int, int) {
	if s == "" {
		return 0, 0
	}
	lines := splitLines(s)
	maxW := 0
	total := 0
	for _, line := range lines {
		lw := width.StringWidth(line)
		if lw > maxW {
			maxW = lw
		}
	}
	if !wrap || availW <= 0 {
		return maxW, len(lines)
	}
	for _, line := range lines {
		if width.StringWidth(line) == 0 {
			total++
			continue
		}
		total += (width.StringWidth(line) + availW - 1) / availW
	}
	if maxW > availW {
		maxW = availW
	}
	return maxW, total
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func intersect(a, b layout.Rect) layout.Rect {
	x0 := maxInt(a.X, b.X)
	y0 := maxInt(a.Y, b.Y)
	x1 := minInt(a.Right(), b.Right())
	y1 := minInt(a.Bottom(), b.Bottom())
	if x1 <= x0 || y1 <= y0 {
		return layout.Rect{}
	}
	return layout.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
