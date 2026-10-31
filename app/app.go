// Package app provides a convenience wrapper for running a gotui application.
package app

import (
	"os"

	"github.com/bijim/gotui/ansi"
	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
	"github.com/bijim/gotui/renderer"
)

// Options configures an App.
type Options struct {
	FPS          int
	Mouse        bool
	AltScreen    bool
	Synchronized bool
	ColorMode    ansi.ColorMode
	Background   color.RGBA
}

// App owns a renderer and runs the main loop.
type App struct {
	Renderer *renderer.Renderer
	opts     Options
}

// New creates an app reading stdin and writing stdout.
func New(opts Options) *App {
	if opts.FPS <= 0 {
		opts.FPS = 60
	}
	r := renderer.New(renderer.Options{
		In:           os.Stdin,
		Out:          os.Stdout,
		FPS:          opts.FPS,
		Mouse:        opts.Mouse,
		AltScreen:    opts.AltScreen,
		Synchronized: opts.Synchronized,
		ColorMode:    opts.ColorMode,
		Background:   opts.Background,
	})
	return &App{Renderer: r, opts: opts}
}

// Run mounts root and blocks until the app stops.
func (a *App) Run(root renderable.Renderable) error {
	n := root.LayoutNode()
	if !n.Style.Width.IsDefinite() && n.Style.Width.Unit != layout.Percent {
		n.Style.Width = layout.Pct(100)
	}
	if !n.Style.Height.IsDefinite() && n.Style.Height.Unit != layout.Percent {
		n.Style.Height = layout.Pct(100)
	}
	a.Renderer.Root().Add(root)
	a.Renderer.Start()
	<-a.Renderer.Done()
	a.Renderer.Close()
	return nil
}

// Quit stops the app.
func (a *App) Quit() { a.Renderer.Stop() }
