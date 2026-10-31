package renderables

import (
	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/input"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
)

// ScrollBarOptions configures a ScrollBar.
type ScrollBarOptions struct {
	ID          string
	Value       int // first visible unit
	ContentSize int // total units
	Viewport    int // visible units
	Vertical    bool
	ShowArrows  bool
	TrackColor  color.RGBA
	ThumbColor  color.RGBA
	FG          color.RGBA
	BG          color.RGBA
	Focusable   bool
	Style       layout.Style
	OnChange    func(value int)
}

// ScrollBar is a standalone scroll indicator and controller.
type ScrollBar struct {
	*renderable.Base
	Value       int
	ContentSize int
	Viewport    int
	Vertical    bool
	ShowArrows  bool
	TrackColor  color.RGBA
	ThumbColor  color.RGBA
	FG          color.RGBA
	BG          color.RGBA
	OnChange    func(value int)
	dragging    bool
	dragOffset  int
}

// NewScrollBar creates a scroll bar.
func NewScrollBar(opts ScrollBarOptions) *ScrollBar {
	if opts.Style == (layout.Style{}) {
		opts.Style = layout.DefaultStyle()
	}
	s := &ScrollBar{
		Value:       opts.Value,
		ContentSize: opts.ContentSize,
		Viewport:    opts.Viewport,
		Vertical:    opts.Vertical,
		ShowArrows:  opts.ShowArrows,
		TrackColor:  opts.TrackColor,
		ThumbColor:  opts.ThumbColor,
		FG:          opts.FG,
		BG:          opts.BG,
		OnChange:    opts.OnChange,
	}
	if s.TrackColor == (color.RGBA{}) {
		s.TrackColor = DarkGray
	}
	if s.ThumbColor == (color.RGBA{}) {
		s.ThumbColor = Gray
	}
	if s.FG == (color.RGBA{}) {
		s.FG = White
	}
	if s.Vertical {
		opts.Style.Width = layout.P(1)
		if opts.Style.Height.Unit == layout.Undefined {
			opts.Style.Height = layout.Pct(100)
		}
	} else {
		if opts.Style.Height.Unit == layout.Undefined {
			opts.Style.Height = layout.P(1)
		}
		if opts.Style.Width.Unit == layout.Undefined {
			opts.Style.Width = layout.Pct(100)
		}
	}
	s.Base = renderable.NewBase(s, &layout.Node{Style: opts.Style})
	s.Base.Focusable = opts.Focusable
	s.Base.Node.Measure = s.measure
	s.Base.OnMouseDown = func(ev *input.MouseEvent) bool {
		s.dragging = true
		s.setFromMouse(ev)
		return true
	}
	s.Base.OnMouseDrag = func(ev *input.MouseEvent) bool {
		if s.dragging {
			s.setFromMouse(ev)
		}
		return true
	}
	s.Base.OnMouseUp = func(*input.MouseEvent) bool {
		s.dragging = false
		return true
	}
	return s
}

func (s *ScrollBar) measure(availW, availH int) (int, int) {
	if s.Vertical {
		return 1, maxInt(1, availH)
	}
	return maxInt(1, availW), 1
}

// Max returns the largest valid scroll value.
func (s *ScrollBar) Max() int { return maxInt(0, s.ContentSize-s.Viewport) }

// SetValue sets the scroll value and notifies.
func (s *ScrollBar) SetValue(v int) {
	v = clampInt(v, 0, s.Max())
	if v == s.Value {
		return
	}
	s.Value = v
	if s.OnChange != nil {
		s.OnChange(v)
	}
	s.Invalidate()
	s.RequestRender()
}

func (s *ScrollBar) setFromMouse(ev *input.MouseEvent) {
	track := s.trackLen()
	if track <= 1 {
		return
	}
	r := s.ContentBounds()
	var pos int
	if s.Vertical {
		pos = ev.Y - r.Y
	} else {
		pos = ev.X - r.X
	}
	thumb := s.thumbLen()
	maxPos := maxInt(1, track-thumb)
	s.SetValue(int(float64(pos-thumb/2) / float64(maxPos) * float64(s.Max())))
}

func (s *ScrollBar) trackLen() int {
	content := s.ContentBounds()
	n := content.H
	if !s.Vertical {
		n = content.W
	}
	if s.ShowArrows {
		n -= 2
	}
	return maxInt(0, n)
}

func (s *ScrollBar) thumbLen() int {
	track := s.trackLen()
	if s.ContentSize <= 0 || s.Viewport <= 0 {
		return track
	}
	n := track * s.Viewport / maxInt(1, s.ContentSize)
	if n < 1 {
		n = 1
	}
	if n > track {
		n = track
	}
	return n
}

// HandleKey scrolls with the keyboard.
func (s *ScrollBar) HandleKey(ev *input.KeyEvent) bool {
	if ev.Type == input.KeyRelease {
		return false
	}
	if s.OnKeyDown != nil && s.OnKeyDown(ev) {
		return true
	}
	switch ev.Name {
	case "up", "left":
		s.SetValue(s.Value - 1)
	case "down", "right":
		s.SetValue(s.Value + 1)
	case "pageup":
		s.SetValue(s.Value - maxInt(1, s.Viewport))
	case "pagedown":
		s.SetValue(s.Value + maxInt(1, s.Viewport))
	case "home":
		s.SetValue(0)
	case "end":
		s.SetValue(s.Max())
	default:
		return false
	}
	return true
}

// Paint draws the scrollbar.
func (s *ScrollBar) Paint(ctx *renderable.Context) {
	if !s.Visible {
		return
	}
	r := s.Bounds()
	if s.BG.A > 0 {
		ctx.Fill(r.X, r.Y, r.W, r.H, s.BG)
	}
	content := s.ContentBounds()
	track := s.trackLen()
	if track <= 0 {
		return
	}
	thumb := s.thumbLen()
	start := 0
	if max := s.Max(); max > 0 {
		start = s.Value * maxInt(0, track-thumb) / max
	}
	if s.Vertical {
		y := content.Y
		if s.ShowArrows {
			ctx.SetString(content.X, y, "▲", s.FG, s.BG, 0)
			y++
			ctx.SetString(content.X, y+track-1, "▼", s.FG, s.BG, 0)
		}
		for i := 0; i < track; i++ {
			ch, fg := "│", s.TrackColor
			if i >= start && i < start+thumb {
				ch, fg = "█", s.ThumbColor
			}
			ctx.SetString(content.X, y+i, ch, fg, s.BG, 0)
		}
		return
	}
	x := content.X
	if s.ShowArrows {
		ctx.SetString(x, content.Y, "◀", s.FG, s.BG, 0)
		x++
		ctx.SetString(x+track-1, content.Y, "▶", s.FG, s.BG, 0)
	}
	for i := 0; i < track; i++ {
		ch, fg := "─", s.TrackColor
		if i >= start && i < start+thumb {
			ch, fg = "█", s.ThumbColor
		}
		ctx.SetString(x+i, content.Y, ch, fg, s.BG, 0)
	}
}
