package renderables

import (
	"time"

	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
	"github.com/bijim/gotui/width"
)

// DefaultSpinnerFrames is the classic braille spinner.
var DefaultSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// SpinnerOptions configures a Spinner.
type SpinnerOptions struct {
	ID       string
	Frames   []string
	Interval time.Duration
	FG       color.RGBA
	BG       color.RGBA
	Text     string
	Style    layout.Style
}

// Spinner is an animated progress indicator.
type Spinner struct {
	*renderable.Base
	Frames   []string
	Interval time.Duration
	FG       color.RGBA
	BG       color.RGBA
	Text     string

	frame int
	acc   time.Duration
}

// NewSpinner creates a spinner.
func NewSpinner(opts SpinnerOptions) *Spinner {
	if opts.Style == (layout.Style{}) {
		opts.Style = layout.DefaultStyle()
	}
	s := &Spinner{
		Frames:   opts.Frames,
		Interval: opts.Interval,
		FG:       opts.FG,
		BG:       opts.BG,
		Text:     opts.Text,
	}
	if len(s.Frames) == 0 {
		s.Frames = DefaultSpinnerFrames
	}
	if s.Interval <= 0 {
		s.Interval = 80 * time.Millisecond
	}
	if s.FG == (color.RGBA{}) {
		s.FG = White
	}
	s.Base = renderable.NewBase(s, &layout.Node{Style: opts.Style})
	s.Base.Node.Measure = func(availW, availH int) (int, int) {
		w := 0
		if len(s.Frames) > 0 {
			w = width.StringWidth(s.Frames[0])
		}
		if s.Text != "" {
			w += 1 + width.StringWidth(s.Text)
		}
		return w, 1
	}
	s.Base.OnUpdate = func(dt float64) bool {
		s.acc += time.Duration(dt * float64(time.Second))
		if s.acc < s.Interval {
			return false
		}
		s.acc = 0
		s.frame = (s.frame + 1) % len(s.Frames)
		return true
	}
	return s
}

// Paint draws the spinner.
func (s *Spinner) Paint(ctx *renderable.Context) {
	if !s.Visible {
		return
	}
	content := s.ContentBounds()
	if content.W <= 0 || content.H <= 0 {
		return
	}
	frame := s.Frames[s.frame]
	ctx.SetString(content.X, content.Y, frame, s.FG, s.BG, 0)
	if s.Text != "" {
		ctx.SetString(content.X+width.StringWidth(frame)+1, content.Y, s.Text, s.FG, s.BG, 0)
	}
}

// ProgressBarOptions configures a ProgressBar.
type ProgressBarOptions struct {
	ID          string
	Progress    float64
	BarColor    color.RGBA
	TrackColor  color.RGBA
	BG          color.RGBA
	BarChar     rune
	TrackChar   rune
	ShowPercent bool
	PercentFG   color.RGBA
	Style       layout.Style
}

// ProgressBar shows 0..1 progress.
type ProgressBar struct {
	*renderable.Base
	Progress    float64
	BarColor    color.RGBA
	TrackColor  color.RGBA
	BG          color.RGBA
	BarChar     rune
	TrackChar   rune
	ShowPercent bool
	PercentFG   color.RGBA
}

// NewProgressBar creates a progress bar.
func NewProgressBar(opts ProgressBarOptions) *ProgressBar {
	if opts.Style == (layout.Style{}) {
		opts.Style = layout.DefaultStyle()
	}
	p := &ProgressBar{
		Progress:    opts.Progress,
		BarColor:    opts.BarColor,
		TrackColor:  opts.TrackColor,
		BG:          opts.BG,
		BarChar:     opts.BarChar,
		TrackChar:   opts.TrackChar,
		ShowPercent: opts.ShowPercent,
		PercentFG:   opts.PercentFG,
	}
	if p.BarColor == (color.RGBA{}) {
		p.BarColor = White
	}
	if p.TrackColor == (color.RGBA{}) {
		p.TrackColor = DarkGray
	}
	if p.PercentFG == (color.RGBA{}) {
		p.PercentFG = White
	}
	if p.BarChar == 0 {
		p.BarChar = '█'
	}
	if p.TrackChar == 0 {
		p.TrackChar = '░'
	}
	opts.Style.Height = layout.P(1)
	p.Base = renderable.NewBase(p, &layout.Node{Style: opts.Style})
	return p
}

// SetProgress updates the value.
func (p *ProgressBar) SetProgress(v float64) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	p.Progress = v
	p.Invalidate()
	p.RequestRender()
}

// Paint draws the bar.
func (p *ProgressBar) Paint(ctx *renderable.Context) {
	if !p.Visible {
		return
	}
	r := p.Bounds()
	if p.BG.A > 0 {
		ctx.Fill(r.X, r.Y, r.W, r.H, p.BG)
	}
	content := p.ContentBounds()
	w := content.W
	pctText := ""
	if p.ShowPercent {
		pctText = " " + itoa(int(p.Progress*100)) + "%"
		w -= width.StringWidth(pctText)
	}
	if w <= 0 {
		return
	}
	filled := int(p.Progress*float64(w) + 0.5)
	for i := 0; i < w; i++ {
		ch := p.TrackChar
		fg := p.TrackColor
		if i < filled {
			ch = p.BarChar
			fg = p.BarColor
		}
		ctx.SetString(content.X+i, content.Y, string(ch), fg, p.BG, 0)
	}
	if pctText != "" {
		ctx.SetString(content.Right()-width.StringWidth(pctText), content.Y, pctText, p.PercentFG, p.BG, 0)
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
