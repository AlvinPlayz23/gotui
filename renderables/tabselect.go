package renderables

import (
	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/input"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
	"github.com/bijim/gotui/width"
)

// TabItem is one tab and its content.
type TabItem struct {
	Name    string
	Content renderable.Renderable
}

// TabSelectOptions configures a TabSelect.
type TabSelectOptions struct {
	ID       string
	Items    []TabItem
	Active   int
	FG       color.RGBA
	BG       color.RGBA
	ActiveFG color.RGBA
	ActiveBG color.RGBA
	TabBarBG color.RGBA
	Style    layout.Style
	OnChange func(index int)
}

// TabSelect shows a tab bar above the active panel.
type TabSelect struct {
	*renderable.Base
	Items    []TabItem
	Active   int
	FG       color.RGBA
	BG       color.RGBA
	ActiveFG color.RGBA
	ActiveBG color.RGBA
	TabBarBG color.RGBA
	OnChange func(index int)

	tabRects []layout.Rect
}

// NewTabSelect creates a tab select.
func NewTabSelect(opts TabSelectOptions) *TabSelect {
	if opts.Style == (layout.Style{}) {
		opts.Style = layout.DefaultStyle()
	}
	opts.Style.Direction = layout.Column
	if opts.Style.Padding[0].Unit == layout.Undefined {
		opts.Style.Padding[0] = layout.P(1)
	}
	t := &TabSelect{
		Items:    opts.Items,
		Active:   opts.Active,
		FG:       opts.FG,
		BG:       opts.BG,
		ActiveFG: opts.ActiveFG,
		ActiveBG: opts.ActiveBG,
		TabBarBG: opts.TabBarBG,
		OnChange: opts.OnChange,
	}
	if t.FG == (color.RGBA{}) {
		t.FG = Gray
	}
	if t.ActiveFG == (color.RGBA{}) {
		t.ActiveFG = White
	}
	if t.ActiveBG == (color.RGBA{}) {
		t.ActiveBG = DarkGray
	}
	t.Base = renderable.NewBase(t, &layout.Node{Style: opts.Style})
	t.Base.Focusable = true
	for _, it := range opts.Items {
		if it.Content != nil {
			t.Base.Add(it.Content)
		}
	}
	t.applyActive()
	t.Base.OnMouseDown = func(ev *input.MouseEvent) bool {
		for i, tr := range t.tabRects {
			if tr.Contains(ev.X, ev.Y) {
				t.SetActive(i)
				return true
			}
		}
		return false
	}
	return t
}

func (t *TabSelect) applyActive() {
	for i, it := range t.Items {
		if it.Content == nil {
			continue
		}
		if i == t.Active {
			it.Content.LayoutNode().Style.Display = layout.Flex
			it.Content.LayoutNode().Style.FlexGrow = 1
			it.Content.LayoutNode().Style.FlexShrink = 1
		} else {
			it.Content.LayoutNode().Style.Display = layout.None
		}
	}
}

// SetActive selects a tab.
func (t *TabSelect) SetActive(i int) {
	if i < 0 || i >= len(t.Items) || i == t.Active {
		return
	}
	t.Active = i
	t.applyActive()
	if t.OnChange != nil {
		t.OnChange(i)
	}
	t.Invalidate()
	t.RequestRender()
}

// HandleKey switches tabs with left/right and numbers.
func (t *TabSelect) HandleKey(ev *input.KeyEvent) bool {
	if ev.Type == input.KeyRelease {
		return false
	}
	if t.OnKeyDown != nil && t.OnKeyDown(ev) {
		return true
	}
	if len(t.Items) == 0 {
		return false
	}
	switch ev.Name {
	case "left":
		t.SetActive((t.Active - 1 + len(t.Items)) % len(t.Items))
	case "right":
		t.SetActive((t.Active + 1) % len(t.Items))
	case "home":
		t.SetActive(0)
	case "end":
		t.SetActive(len(t.Items) - 1)
	default:
		if ev.IsText() && ev.Rune >= '1' && ev.Rune <= '9' {
			idx := int(ev.Rune - '1')
			if idx < len(t.Items) {
				t.SetActive(idx)
				return true
			}
		}
		return false
	}
	return true
}

// Paint draws the tab bar and active content.
func (t *TabSelect) Paint(ctx *renderable.Context) {
	if !t.Visible {
		return
	}
	r := t.Bounds()
	if t.BG.A > 0 {
		ctx.Fill(r.X, r.Y, r.W, r.H, t.BG)
	}
	barY := r.Y
	if t.TabBarBG.A > 0 {
		ctx.Fill(r.X, barY, r.W, 1, t.TabBarBG)
	}
	t.tabRects = t.tabRects[:0]
	x := r.X
	for i, it := range t.Items {
		label := " " + it.Name + " "
		w := width.StringWidth(label)
		if x+w > r.Right() {
			break
		}
		fg, bg := t.FG, t.TabBarBG
		if i == t.Active {
			fg, bg = t.ActiveFG, t.ActiveBG
			if bg.A > 0 {
				ctx.Fill(x, barY, w, 1, bg)
			}
		}
		ctx.SetString(x, barY, label, fg, bg, 0)
		t.tabRects = append(t.tabRects, layout.Rect{X: x, Y: barY, W: w, H: 1})
		x += w
	}
	content := t.ContentBounds()
	t.PaintChildren(ctx.WithClip(content))
}
