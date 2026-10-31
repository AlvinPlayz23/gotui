package renderables

import (
	"fmt"

	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/input"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
)

// SliderOptions configures a Slider.
type SliderOptions struct {
	ID         string
	Value      float64 // 0..1
	Min        float64
	Max        float64
	Step       float64
	Vertical   bool
	TrackColor color.RGBA
	ThumbColor color.RGBA
	FG         color.RGBA
	BG         color.RGBA
	ShowValue  bool
	Format     func(float64) string
	Focusable  bool
	Style      layout.Style
	OnChange   func(float64)
}

// Slider is a horizontal or vertical value slider.
type Slider struct {
	*renderable.Base
	Value      float64
	Min, Max   float64
	Step       float64
	Vertical   bool
	TrackColor color.RGBA
	ThumbColor color.RGBA
	FG         color.RGBA
	BG         color.RGBA
	ShowValue  bool
	Format     func(float64) string
	OnChange   func(float64)
	dragging   bool
}

// NewSlider creates a slider.
func NewSlider(opts SliderOptions) *Slider {
	if opts.Style == (layout.Style{}) {
		opts.Style = layout.DefaultStyle()
	}
	s := &Slider{
		Value:      clamp01(opts.Value),
		Min:        opts.Min,
		Max:        opts.Max,
		Step:       opts.Step,
		Vertical:   opts.Vertical,
		TrackColor: opts.TrackColor,
		ThumbColor: opts.ThumbColor,
		FG:         opts.FG,
		BG:         opts.BG,
		ShowValue:  opts.ShowValue,
		Format:     opts.Format,
		OnChange:   opts.OnChange,
	}
	if s.Max <= s.Min {
		s.Min, s.Max = 0, 1
	}
	if s.Step <= 0 {
		s.Step = 0.05
	}
	if s.TrackColor == (color.RGBA{}) {
		s.TrackColor = DarkGray
	}
	if s.ThumbColor == (color.RGBA{}) {
		s.ThumbColor = White
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

func (s *Slider) measure(availW, availH int) (int, int) {
	if s.Vertical {
		return 1, maxInt(1, availH)
	}
	return maxInt(1, availW), 1
}

// SetValue sets the normalized value, clamps it, and notifies.
func (s *Slider) SetValue(v float64) {
	v = clamp01(v)
	if v == s.Value {
		return
	}
	s.Value = v
	if s.OnChange != nil {
		s.OnChange(s.Absolute())
	}
	s.Invalidate()
	s.RequestRender()
}

// Absolute returns the value mapped into Min..Max.
func (s *Slider) Absolute() float64 { return s.Min + s.Value*(s.Max-s.Min) }

// SetAbsolute sets the slider from a value in Min..Max.
func (s *Slider) SetAbsolute(v float64) {
	if s.Max <= s.Min {
		return
	}
	s.SetValue((v - s.Min) / (s.Max - s.Min))
}

func (s *Slider) setFromMouse(ev *input.MouseEvent) {
	r := s.ContentBounds()
	if s.Vertical {
		h := maxInt(1, r.H-1)
		s.SetValue(1 - float64(ev.Y-r.Y)/float64(h))
		return
	}
	w := maxInt(1, r.W-1)
	s.SetValue(float64(ev.X-r.X) / float64(w))
}

// HandleKey moves the slider.
func (s *Slider) HandleKey(ev *input.KeyEvent) bool {
	if ev.Type == input.KeyRelease {
		return false
	}
	if s.OnKeyDown != nil && s.OnKeyDown(ev) {
		return true
	}
	switch ev.Name {
	case "left", "down":
		s.SetValue(s.Value - s.Step)
	case "right", "up":
		s.SetValue(s.Value + s.Step)
	case "home":
		s.SetValue(0)
	case "end":
		s.SetValue(1)
	default:
		return false
	}
	return true
}

func (s *Slider) label() string {
	if s.Format != nil {
		return s.Format(s.Absolute())
	}
	return fmt.Sprintf("%.0f", s.Value*100)
}

// Paint draws the slider.
func (s *Slider) Paint(ctx *renderable.Context) {
	if !s.Visible {
		return
	}
	r := s.Bounds()
	if s.BG.A > 0 {
		ctx.Fill(r.X, r.Y, r.W, r.H, s.BG)
	}
	content := s.ContentBounds()
	if content.W <= 0 || content.H <= 0 {
		return
	}
	if s.Vertical {
		h := content.H
		thumb := h - 1 - int(float64(h-1)*s.Value)
		for i := 0; i < h; i++ {
			ch := "│"
			fg := s.TrackColor
			if i == thumb {
				ch = "●"
				fg = s.ThumbColor
				if s.Focused {
					fg = s.FG
				}
			}
			ctx.SetString(content.X, content.Y+i, ch, fg, s.BG, 0)
		}
		return
	}
	w := content.W
	if s.ShowValue {
		label := s.label()
		lw := len(label)
		w = maxInt(1, content.W-lw-1)
		ctx.SetString(content.Right()-lw, content.Y, label, s.FG, s.BG, 0)
	}
	thumb := int(float64(w-1) * s.Value)
	if w <= 1 {
		thumb = 0
	}
	for i := 0; i < w; i++ {
		ch := "─"
		fg := s.TrackColor
		if i == thumb {
			ch = "●"
			fg = s.ThumbColor
			if s.Focused {
				fg = s.FG
			}
		}
		ctx.SetString(content.X+i, content.Y, ch, fg, s.BG, 0)
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
