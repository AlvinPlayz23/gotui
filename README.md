# gotui

A pure-Go port of [OpenTUI](https://github.com/anomalyco/opentui)'s core and widgets.
No native libraries, no cgo. One static binary.

This is an in-progress port of the plan in [port_plan.md](port_plan.md). The
deliverable so far is the pure-Go path: a cell renderer, Yoga-compatible flex
layout, input parsing, a text engine, and the P0 widget set.

## Status

Implemented:

- **Core**: RGBA colors and parsing, Unicode display width and grapheme
  clustering, ANSI sequences and a diffing cell encoder, optimized cell grid,
  flexbox layout, renderable tree, renderer event loop, keyboard/mouse/paste
  input, app lifecycle.
- **Text engine**: piece-table rope, text buffer with tabs and grapheme-aware
  navigation, wrapping text view with hit testing and selection geometry, edit
  buffer with cursor/selection/undo/redo/extmarks, editor view with cursor
  scrolling.
- **Widgets**: Box, Text, ScrollBox, Input, Textarea, Editor, Select,
  TabSelect, Slider, ScrollBar, ASCIIFont, ProgressBar, Spinner.
- **Testing**: golden-frame snapshots (`testing` package) and unit tests across
  color, width, ANSI, layout, text, input, and widgets.

Not implemented yet (from the plan):

- Native FFI backend (this port went straight to pure Go).
- Images, video, audio, clipboard, embedded terminal.
- Code/Markdown/Diff/TextTable widgets and syntax highlighting.
- Post effects, animation timeline, plugins, split-footer/scrollback,
  terminal capability detection, kitty keyboard protocol.

## Install

```sh
go get github.com/bijim/gotui
```

Requires Go 1.23+.

## Usage

```go
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
```

Run the demo suite:

```sh
go run ./examples/demo
```

The demo has Form, List, Widgets, and Editor tabs. `tab` cycles focus, `q` or
`ctrl+c` quits.

## Packages

| Package | Purpose |
|---|---|
| `app` | Terminal app lifecycle |
| `renderer` | Event loop, frames, mouse routing, focus |
| `renderable` | Base renderable, paint context, hit testing |
| `renderables` | Box, Text, Input, Textarea, Editor, Select, Slider, ... |
| `layout` | Yoga-compatible flexbox |
| `cell` | Cell grid and drawing primitives |
| `text` | Rope, text buffer/view, edit buffer/editor view |
| `input` | Key, mouse, paste, resize parser |
| `ansi` | Escape sequences and frame diff encoder |
| `color` | RGBA parsing, blending, palettes |
| `width` | Grapheme clustering and cell widths |
| `testing` | Golden-frame test harness |

## Testing

```sh
go test ./...
go test ./testing -update   # regenerate golden files
```

## License

MIT. OpenTUI is MIT; see [NOTICE](NOTICE) for attribution.
