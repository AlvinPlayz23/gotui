package main

import (
	"fmt"

	"github.com/bijim/gotui/ansi"
	"github.com/bijim/gotui/app"
	"github.com/bijim/gotui/cell"
	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/input"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
	"github.com/bijim/gotui/renderables"
	"github.com/bijim/gotui/text"
)

var (
	bg      = color.FromInts(30, 30, 46, 255)
	fg      = color.FromInts(205, 214, 244, 255)
	accent  = color.FromInts(137, 180, 250, 255)
	green   = color.FromInts(166, 227, 161, 255)
	surface = color.FromInts(49, 50, 68, 255)
)

func main() {
	a := app.New(app.Options{
		FPS:          60,
		Mouse:        true,
		AltScreen:    true,
		Synchronized: true,
		ColorMode:    ansi.ColorTrue,
		Background:   bg,
	})
	root := buildRoot(a.Quit)
	if err := a.Run(root); err != nil {
		fmt.Println(err)
	}
}

func buildRoot(quit func()) renderable.Renderable {
	root := renderables.NewBox(renderables.BoxOptions{
		Background: bg,
		Padding:    1,
		Style:      layout.Style{Direction: layout.Column, FlexGrow: 1, FlexShrink: 1, Width: layout.Pct(100), Height: layout.Pct(100), GapRow: layout.P(1)},
	})

	header := renderables.NewText("GoTUI", renderables.TextOptions{
		FG:    accent,
		Attr:  cell.AttrBold,
		Align: "center",
		Style: layout.Style{Direction: layout.Column, Width: layout.Pct(100)},
	})

	tabs := renderables.NewTabSelect(renderables.TabSelectOptions{
		Items: []renderables.TabItem{
			{Name: "Form", Content: formTab()},
			{Name: "List", Content: listTab()},
			{Name: "Widgets", Content: widgetsTab()},
			{Name: "Editor", Content: editorTab()},
		},
		TabBarBG: surface,
		ActiveBG: accent,
		ActiveFG: bg,
		FG:       fg,
		Style:    layout.Style{Direction: layout.Column, FlexGrow: 1, FlexShrink: 1},
		OnChange: func(i int) {},
	})

	footer := renderables.NewText("tab: focus  ·  ctrl+c / q: quit", renderables.TextOptions{
		FG:    renderables.Gray,
		Align: "center",
		Style: layout.Style{Direction: layout.Column, Width: layout.Pct(100)},
	})

	root.Add(header, tabs, footer)
	root.Base.OnKeyDown = func(ev *input.KeyEvent) bool {
		if ev.Name == "q" && !ev.Ctrl && !ev.Alt {
			quit()
			return true
		}
		return false
	}
	return root
}

func panel(title string) *renderables.Box {
	return renderables.NewBox(renderables.BoxOptions{
		Border:      true,
		Title:       title,
		BorderColor: surface,
		BorderStyle: cell.BorderRound,
		Padding:     1,
		Style:       layout.Style{Direction: layout.Column, FlexGrow: 1, FlexShrink: 1, GapRow: layout.P(1)},
	})
}

func label(s string) *renderables.Text {
	return renderables.NewText(s, renderables.TextOptions{
		FG:    fg,
		Attr:  cell.AttrBold,
		Style: layout.Style{Direction: layout.Column},
	})
}

func formTab() renderable.Renderable {
	p := panel("Profile")

	name := renderables.NewInput(renderables.InputOptions{
		Placeholder:      "Ada Lovelace",
		FG:               fg,
		BG:               surface,
		CursorFG:         bg,
		CursorBG:         accent,
		PlaceholderColor: renderables.Gray,
		Style:            layout.Style{Direction: layout.Column, FlexGrow: 1},
	})
	pass := renderables.NewInput(renderables.InputOptions{
		Placeholder:      "secret",
		Password:         true,
		FG:               fg,
		BG:               surface,
		CursorFG:         bg,
		CursorBG:         accent,
		PlaceholderColor: renderables.Gray,
		Style:            layout.Style{Direction: layout.Column, FlexGrow: 1},
	})
	sel := renderables.NewSelect(renderables.SelectOptions{
		Options: []renderables.SelectOption{
			{Name: "Catppuccin", Description: "mocha"},
			{Name: "Nord", Description: "polar"},
			{Name: "Dracula", Description: "vampire"},
			{Name: "Gruvbox", Description: "retro"},
		},
		Selected:        0,
		FG:              fg,
		BG:              bg,
		SelectedFG:      bg,
		SelectedBG:      accent,
		ShowDescription: true,
	})

	bar := renderables.NewProgressBar(renderables.ProgressBarOptions{
		Progress:    0.42,
		BarColor:    green,
		TrackColor:  surface,
		ShowPercent: true,
		Style:       layout.Style{Direction: layout.Column, Width: layout.Pct(100)},
	})

	p.Add(label("Name"), name, label("Password"), pass, label("Theme"), sel, bar)
	return scrollable(p)
}

func scrollable(r renderable.Renderable) renderable.Renderable {
	sb := renderables.NewScrollBox(renderables.ScrollBoxOptions{
		Scrollbar:   true,
		ScrollbarFG: accent,
		Style:       layout.Style{Direction: layout.Column, FlexGrow: 1, FlexShrink: 1},
	})
	sb.Content.Add(r)
	return sb
}

func listTab() renderable.Renderable {
	sb := renderables.NewScrollBox(renderables.ScrollBoxOptions{
		Scrollbar:   true,
		ScrollbarFG: accent,
		Style:       layout.Style{Direction: layout.Column, FlexGrow: 1},
		Focusable:   true,
	})
	for i := 1; i <= 200; i++ {
		sb.Content.Add(renderables.NewText(fmt.Sprintf("%3d  row number %d", i, i), renderables.TextOptions{
			FG:    fg,
			Style: layout.Style{Direction: layout.Column},
		}))
	}
	return sb
}

func widgetsTab() renderable.Renderable {
	p := panel("Widgets")

	spin := renderables.NewSpinner(renderables.SpinnerOptions{
		Text:  "loading…",
		FG:    accent,
		Style: layout.Style{Direction: layout.Column},
	})

	bold := renderables.NewText("Bold", renderables.TextOptions{FG: fg, Attr: cell.AttrBold})
	italic := renderables.NewText("Italic", renderables.TextOptions{FG: fg, Attr: cell.AttrItalic})
	underline := renderables.NewText("Underline", renderables.TextOptions{FG: fg, Attr: cell.AttrUnderline})
	strike := renderables.NewText("Strikethrough", renderables.TextOptions{FG: fg, Attr: cell.AttrStrikethrough})
	wrap := renderables.NewText(
		"The quick brown fox jumps over the lazy dog. This sentence is long enough to wrap across several lines in the panel, demonstrating word wrapping.",
		renderables.TextOptions{
			FG:    renderables.Gray,
			Wrap:  text.WrapWord,
			Style: layout.Style{Direction: layout.Column, Width: layout.Pct(100)},
		})

	p.Add(spin, bold, italic, underline, strike, wrap)
	return scrollable(p)
}

func editorTab() renderable.Renderable {
	p := panel("Editor")

	ed := renderables.NewEditor(renderables.EditorOptions{
		Value:        "GoTUI editor\n\nType, select, undo and redo.\n\nUse ctrl+z to undo, ctrl+shift+z to redo.\nShift+arrows select text. Drag with the mouse.\n",
		FG:           fg,
		BG:           surface,
		CursorFG:     bg,
		CursorBG:     accent,
		SelectionBG:  color.FromInts(69, 71, 90, 255),
		LineNumberFG: renderables.Gray,
		LineNumberBG: surface,
		LineNumbers:  true,
		Wrap:         text.WrapWord,
		Style: layout.Style{
			Direction:  layout.Column,
			FlexGrow:   1,
			FlexShrink: 1,
			Width:      layout.Pct(100),
		},
	})

	p.Add(ed)
	return p
}
