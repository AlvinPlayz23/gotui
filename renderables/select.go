package renderables

import (
	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/input"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
	"github.com/bijim/gotui/width"
)

// SelectOption is one item in a Select.
type SelectOption struct {
	Name        string
	Description string
	Value       any
}

// SelectOptions configures a Select.
type SelectOptions struct {
	ID               string
	Options          []SelectOption
	Selected         int
	FG               color.RGBA
	BG               color.RGBA
	SelectedFG       color.RGBA
	SelectedBG       color.RGBA
	DescriptionColor color.RGBA
	ShowDescription  bool
	Style            layout.Style
	OnChange         func(index int, option SelectOption)
}

// Select is a keyboard-navigable list.
type Select struct {
	*renderable.Base
	Options          []SelectOption
	Selected         int
	FG               color.RGBA
	BG               color.RGBA
	SelectedFG       color.RGBA
	SelectedBG       color.RGBA
	DescriptionColor color.RGBA
	ShowDescription  bool
	OnChange         func(index int, option SelectOption)

	filter  []rune
	scroll  int
	visible int
}

// NewSelect creates a select list.
func NewSelect(opts SelectOptions) *Select {
	if opts.Style == (layout.Style{}) {
		opts.Style = layout.DefaultStyle()
	}
	s := &Select{
		Options:          opts.Options,
		Selected:         opts.Selected,
		FG:               opts.FG,
		BG:               opts.BG,
		SelectedFG:       opts.SelectedFG,
		SelectedBG:       opts.SelectedBG,
		DescriptionColor: opts.DescriptionColor,
		ShowDescription:  opts.ShowDescription,
		OnChange:         opts.OnChange,
	}
	if s.FG == (color.RGBA{}) {
		s.FG = White
	}
	if s.SelectedFG == (color.RGBA{}) {
		s.SelectedFG = Black
	}
	if s.SelectedBG == (color.RGBA{}) {
		s.SelectedBG = White
	}
	if s.DescriptionColor == (color.RGBA{}) {
		s.DescriptionColor = Gray
	}
	if opts.Style.Direction == layout.Column && opts.Style.Height.Unit == 0 && len(opts.Options) > 0 {
		opts.Style.Height = layout.P(len(opts.Options))
	}
	s.Base = renderable.NewBase(s, &layout.Node{Style: opts.Style})
	s.Base.Focusable = true
	s.Base.Node.Measure = s.measure
	return s
}

func (s *Select) measure(availW, availH int) (int, int) {
	w := 0
	for _, o := range s.Options {
		w = maxInt(w, width.StringWidth(o.Name))
		if s.ShowDescription && o.Description != "" {
			w += 2 + width.StringWidth(o.Description)
		}
	}
	return w, len(s.Options)
}

// OptionsFiltered returns options matching the filter.
func (s *Select) ensureSelectionVisible() {
	vis := s.filtered()
	if len(vis) == 0 {
		return
	}
	for _, idx := range vis {
		if idx == s.Selected {
			return
		}
	}
	s.Selected = vis[0]
	s.ensureVisible()
}

func (s *Select) filtered() []int {
	if len(s.filter) == 0 {
		idx := make([]int, len(s.Options))
		for i := range idx {
			idx[i] = i
		}
		return idx
	}
	q := string(s.filter)
	var out []int
	for i, o := range s.Options {
		if containsFold(o.Name, q) {
			out = append(out, i)
		}
	}
	return out
}

func (s *Select) selectVisible(delta int) {
	vis := s.filtered()
	if len(vis) == 0 {
		return
	}
	pos := 0
	for i, idx := range vis {
		if idx == s.Selected {
			pos = i
			break
		}
	}
	pos = clampInt(pos+delta, 0, len(vis)-1)
	s.Selected = vis[pos]
	s.ensureVisible()
	s.RequestRender()
}

func (s *Select) ensureVisible() {
	vis := s.filtered()
	pos := 0
	for i, idx := range vis {
		if idx == s.Selected {
			pos = i
			break
		}
	}
	h := s.ContentBounds().H
	if h <= 0 {
		h = len(vis)
	}
	if pos < s.scroll {
		s.scroll = pos
	}
	if pos >= s.scroll+h {
		s.scroll = pos - h + 1
	}
	if s.scroll < 0 {
		s.scroll = 0
	}
}

// SetSelected selects an index and notifies.
func (s *Select) SetSelected(i int, notify bool) {
	if i < 0 || i >= len(s.Options) {
		return
	}
	s.Selected = i
	s.ensureVisible()
	if notify && s.OnChange != nil {
		s.OnChange(i, s.Options[i])
	}
	s.Invalidate()
	s.RequestRender()
}

// HandleKey implements navigation and filtering.
func (s *Select) HandleKey(ev *input.KeyEvent) bool {
	if ev.Type == input.KeyRelease {
		return false
	}
	if s.OnKeyDown != nil && s.OnKeyDown(ev) {
		return true
	}
	switch ev.Name {
	case "up":
		s.selectVisible(-1)
	case "down":
		s.selectVisible(1)
	case "home":
		s.selectVisible(-len(s.Options))
	case "end":
		s.selectVisible(len(s.Options))
	case "pageup":
		s.selectVisible(-maxInt(1, s.ContentBounds().H))
	case "pagedown":
		s.selectVisible(maxInt(1, s.ContentBounds().H))
	case "return", "enter":
		if s.Selected >= 0 && s.Selected < len(s.Options) && s.OnChange != nil {
			s.OnChange(s.Selected, s.Options[s.Selected])
		}
	case "escape":
		if len(s.filter) > 0 {
			s.filter = nil
			s.RequestRender()
		}
	case "backspace":
		if len(s.filter) > 0 {
			s.filter = s.filter[:len(s.filter)-1]
			s.scroll = 0
			s.ensureSelectionVisible()
			s.RequestRender()
		}
	default:
		if ev.IsText() {
			s.filter = append(s.filter, ev.Rune)
			s.scroll = 0
			s.ensureSelectionVisible()
			s.RequestRender()
		} else {
			return false
		}
	}
	return true
}

// Paint draws the list.
func (s *Select) Paint(ctx *renderable.Context) {
	if !s.Visible {
		return
	}
	r := s.Bounds()
	if s.BG.A > 0 {
		ctx.Fill(r.X, r.Y, r.W, r.H, s.BG)
	}
	content := s.ContentBounds()
	vis := s.filtered()
	s.visible = content.H
	s.ensureVisible()
	for row := 0; row < content.H; row++ {
		pos := s.scroll + row
		if pos >= len(vis) {
			break
		}
		idx := vis[pos]
		o := s.Options[idx]
		y := content.Y + row
		fg, bg := s.FG, s.BG
		if idx == s.Selected && s.Focused {
			fg, bg = s.SelectedFG, s.SelectedBG
		} else if idx == s.Selected {
			fg = s.SelectedBG
		}
		if bg.A > 0 {
			ctx.Fill(content.X, y, content.W, 1, bg)
		}
		name := width.Truncate(o.Name, content.W)
		ctx.SetString(content.X, y, name, fg, bg, 0)
		if s.ShowDescription && o.Description != "" {
			nw := width.StringWidth(o.Name)
			dx := content.X + nw + 2
			if dx < content.Right() {
				dc := s.DescriptionColor
				if idx == s.Selected && s.Focused {
					dc = fg
				}
				ctx.SetString(dx, y, width.Truncate(o.Description, content.Right()-dx), dc, bg, 0)
			}
		}
	}
}

func containsFold(s, sub string) bool {
	return len(sub) == 0 || indexFold(s, sub) >= 0
}

func indexFold(s, sub string) int {
	n := len(sub)
	for i := 0; i+n <= len(s); i++ {
		if equalFold(s[i:i+n], sub) {
			return i
		}
	}
	return -1
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		x, y := a[i], b[i]
		if 'A' <= x && x <= 'Z' {
			x += 'a' - 'A'
		}
		if 'A' <= y && y <= 'Z' {
			y += 'a' - 'A'
		}
		if x != y {
			return false
		}
	}
	return true
}
