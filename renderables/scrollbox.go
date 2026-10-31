package renderables

import (
	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/input"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
)

type container struct{ *renderable.Base }

// ScrollBoxOptions configures a ScrollBox.
type ScrollBoxOptions struct {
	ID          string
	Style       layout.Style
	Background  color.RGBA
	ScrollY     int
	ScrollX     int
	Focusable   bool
	Scrollbar   bool
	ScrollbarFG color.RGBA
	ScrollbarBG color.RGBA
	OnScroll    func(y int)
}

// ScrollBox clips and scrolls a single content child.
type ScrollBox struct {
	*renderable.Base
	Content     *container
	ScrollX     int
	ScrollY     int
	Background  color.RGBA
	Scrollbar   bool
	ScrollbarFG color.RGBA
	ScrollbarBG color.RGBA
	OnScroll    func(y int)

	contentH  int
	viewportH int
	dragging  bool
}

// NewScrollBox creates a scroll box.
func NewScrollBox(opts ScrollBoxOptions) *ScrollBox {
	if opts.Style == (layout.Style{}) {
		opts.Style = layout.DefaultStyle()
	}
	opts.Style.Direction = layout.Column
	sb := &ScrollBox{
		ScrollX:     opts.ScrollX,
		ScrollY:     opts.ScrollY,
		Background:  opts.Background,
		Scrollbar:   opts.Scrollbar,
		ScrollbarFG: opts.ScrollbarFG,
		ScrollbarBG: opts.ScrollbarBG,
		OnScroll:    opts.OnScroll,
	}
	if sb.ScrollbarFG == (color.RGBA{}) {
		sb.ScrollbarFG = Gray
	}
	sb.Base = renderable.NewBase(sb, &layout.Node{Style: opts.Style})
	sb.Base.Focusable = opts.Focusable

	c := &container{}
	c.Base = renderable.NewBase(c, &layout.Node{Style: layout.DefaultStyle()})
	c.Node.Style.Direction = layout.Column
	c.Node.Style.FlexShrink = 0
	c.Node.Style.Width = layout.Pct(100)
	c.Node.BeforeLayout = func(n *layout.Node) {
		n.OffsetY = -sb.ScrollY
		n.OffsetX = -sb.ScrollX
	}
	c.Node.AfterLayout = func(n *layout.Node) {
		sb.contentH = n.H
		sb.viewportH = sb.ContentBounds().H
		if sb.clampScroll() {
			sb.RequestRender()
		}
	}
	sb.Content = c
	sb.Base.Add(c)

	sb.Base.OnMouseDown = func(ev *input.MouseEvent) bool {
		if sb.Scrollbar && ev.X >= sb.Bounds().Right()-1 {
			sb.dragging = true
			sb.scrollToBar(ev.Y)
			return true
		}
		return false
	}
	sb.Base.OnMouseDrag = func(ev *input.MouseEvent) bool {
		if sb.dragging {
			sb.scrollToBar(ev.Y)
		}
		return true
	}
	sb.Base.OnScroll = func(ev *input.MouseEvent) bool {
		if ev.Action == input.MouseWheel {
			step := 3
			if ev.ScrollX != 0 || ev.Shift {
				sb.ScrollXBy(step * ev.ScrollX)
			}
			if ev.Scroll != 0 {
				sb.ScrollBy(step * ev.Scroll)
			}
			return true
		}
		return false
	}
	return sb
}

// HitTest clips child hits to the viewport.
func (sb *ScrollBox) HitTest(x, y int) renderable.Renderable {
	if !sb.Visible || !sb.Bounds().Contains(x, y) {
		return nil
	}
	if !sb.ContentBounds().Contains(x, y) {
		return sb.Self()
	}
	return sb.Base.HitTest(x, y)
}

// ScrollBy scrolls vertically by delta lines.
func (sb *ScrollBox) ScrollBy(dy int) {
	sb.ScrollY += dy
	if sb.clampScroll() {
		sb.onScroll()
	}
}

// ScrollXBy scrolls horizontally by delta cells.
func (sb *ScrollBox) ScrollXBy(dx int) {
	content := sb.Content.Node
	maxX := maxInt(0, content.W-sb.ContentBounds().W)
	sb.ScrollX = clampInt(sb.ScrollX+dx, 0, maxX)
	sb.RequestRender()
}

// ScrollTo scrolls to an absolute vertical offset.
func (sb *ScrollBox) ScrollTo(y int) {
	sb.ScrollY = y
	if sb.clampScroll() {
		sb.onScroll()
	}
}

func (sb *ScrollBox) onScroll() {
	if sb.OnScroll != nil {
		sb.OnScroll(sb.ScrollY)
	}
	sb.RequestRender()
}

// MaxScroll returns the maximum vertical offset.
func (sb *ScrollBox) MaxScroll() int { return sb.maxScroll() }

func (sb *ScrollBox) maxScroll() int {
	return maxInt(0, sb.contentH-sb.viewportH)
}

func (sb *ScrollBox) clampScroll() bool {
	maxY := sb.maxScroll()
	old := sb.ScrollY
	sb.ScrollY = clampInt(sb.ScrollY, 0, maxY)
	if sb.ScrollY != old {
		sb.onScroll()
		return true
	}
	return false
}

func (sb *ScrollBox) scrollToBar(y int) {
	b := sb.Bounds()
	if b.H <= 0 {
		return
	}
	ratio := float64(y-b.Y) / float64(b.H)
	sb.ScrollTo(int(ratio * float64(sb.maxScroll())))
	sb.RequestRender()
}

// HandleKey scrolls with the keyboard.
func (sb *ScrollBox) HandleKey(ev *input.KeyEvent) bool {
	if ev.Type == input.KeyRelease {
		return false
	}
	if sb.OnKeyDown != nil && sb.OnKeyDown(ev) {
		return true
	}
	switch ev.Name {
	case "up":
		sb.ScrollBy(-1)
	case "down":
		sb.ScrollBy(1)
	case "pageup":
		sb.ScrollBy(-maxInt(1, sb.viewportH-1))
	case "pagedown":
		sb.ScrollBy(maxInt(1, sb.viewportH-1))
	case "home":
		sb.ScrollTo(0)
	case "end":
		sb.ScrollTo(sb.maxScroll())
	default:
		return false
	}
	return true
}

// OnLayout runs after the scroll box is arranged.
func (sb *ScrollBox) OnLayout() {
	sb.viewportH = sb.ContentBounds().H
}

// Paint draws the scrolled content and scrollbar.
func (sb *ScrollBox) Paint(ctx *renderable.Context) {
	if !sb.Visible {
		return
	}
	r := sb.Bounds()
	if sb.Background.A > 0 {
		ctx.Fill(r.X, r.Y, r.W, r.H, sb.Background)
	}
	content := sb.ContentBounds()
	sb.viewportH = content.H
	sb.PaintChildren(ctx.WithClip(content))

	max := sb.maxScroll()
	if sb.Scrollbar && max > 0 && content.H > 1 {
		barX := r.Right() - 1
		thumbH := maxInt(1, content.H*content.H/(sb.contentH))
		if thumbH > content.H {
			thumbH = content.H
		}
		track := content.H - thumbH
		pos := 0
		if max > 0 {
			pos = sb.ScrollY * track / max
		}
		for i := 0; i < content.H; i++ {
			ch := "│"
			fg := sb.ScrollbarFG
			if i >= pos && i < pos+thumbH {
				ch = "┃"
			}
			ctx.SetString(barX, content.Y+i, ch, fg, sb.ScrollbarBG, 0)
		}
	}
}

// Resize updates the viewport.
func (sb *ScrollBox) Resize(w, h int) {
	sb.Base.Resize(w, h)
	sb.viewportH = sb.ContentBounds().H
}
