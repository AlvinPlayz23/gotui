package main

import (
	"github.com/bijim/gotui/ansi"
	"github.com/bijim/gotui/app"
	"github.com/bijim/gotui/cell"
	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderables"
)

func main() {
	a := app.New(app.Options{
		FPS:       60,
		Mouse:     true,
		AltScreen: true,
		ColorMode: ansi.ColorTrue,
	})

	root := renderables.NewBox(renderables.BoxOptions{
		Border: true,
		Title:  "Hello",
		Style: layout.Style{
			Direction: layout.Column,
			Width:     layout.Pct(100),
			Height:    layout.Pct(100),
			Padding:   [4]layout.Value{layout.P(1), layout.P(1), layout.P(1), layout.P(1)},
		},
	})
	root.Add(renderables.NewText("Pure Go TUI", renderables.TextOptions{
		FG:   color.FromInts(255, 204, 0, 255),
		Attr: cell.AttrBold,
	}))
	root.Add(renderables.NewEditor(renderables.EditorOptions{
		Placeholder: "type here...",
		LineNumbers: true,
		Style:       layout.Style{Direction: layout.Column, FlexGrow: 1},
	}))

	_ = a.Run(root)
}
