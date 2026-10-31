# Porting OpenTUI to Go — Detailed Plan

Status: in progress — pure-Go core and text engine implemented; Phase 2 Code, Diff,
Markdown, TextTable, and LineNumber widgets plus SyntaxStyle support are implemented. Phase 3
wheel scroll chaining is implemented; acceleration, sticky scroll, viewport culling,
and child-into-view remain deferred.
Source: `github.com/anomalyco/opentui` @ `cb3d425` (2026-10-30), MIT
Target: a Go library with ~90% parity for OpenTUI **core + widgets**
Working dir: `/code/gotui`

> Implementation note: the current code follows the plan's Option A/D endpoint
> (pure Go, no native backend). See `README.md` for the implemented subset and
> the remaining phases.

---

## 1. TL;DR / recommendation

**Do a hybrid port: pure-Go API, backend behind an interface, and start by reusing the existing native core through an FFI backend; then add a pure-Go backend behind the same interface.**

Why, in one paragraph: OpenTUI is two layers. The native Zig core (~139k LOC) already does the genuinely hard, high-risk work — Yoga flexbox, the cell grid + ANSI diff encoder, the text/rope/edit engine, Unicode width/grapheme handling, input parsing, images, video, audio, clipboard, and terminal emulation. The TypeScript layer (~88k LOC core + widgets) is where the *API, renderable tree, event model, and every widget* live. Porting the TS layer to Go is unavoidable for parity. Porting the Zig layer is optional and is the riskiest, longest part. So: reimplement the TS layer in Go faithfully, and initially drive the old Zig core through its stable-ish C ABI (`opentui.h`, ~238 functions, all by-pointer, no by-value structs). That gets you to visible parity in months, not years, and keeps the door open to delete the native dependency later by implementing the same Go backend interface in pure Go.

Do **not** transpile TypeScript to Go. Do **not** try to port Yoga by hand on day one. Do **not** start with video/3D/React.

Effort to 90% parity with the FFI backend: **~6–9 months for one experienced Go systems engineer** (phased, see §11). Replacing the native backend with pure Go adds **~3–5 months** and is a separate, gated project.

The one decision I need from you before implementation: **does the deliverable need to be a single pure-Go binary with no native artifacts?** If yes, we still build the interface + FFI backend first for parity, but the pure backend is in-scope and the timeline grows. If no, ship the FFI backend and treat pure Go as a later option.

---

## 2. What OpenTUI actually is

```
@opentui/core            TypeScript API, renderer, renderable tree, widgets
├── renderer.ts  (6.0k)  CliRenderer: loop, frames, resize, mouse, selections,
│                        alt-screen, split-footer/scrollback, kitty keyboard,
│                        terminal capability detection, output capture
├── Renderable.ts (2.5k) base node: layout options, lifecycle, events, z-order
├── NativeScene.ts (0.8k) TS <-> native scene bridging (staging, paint recording)
├── NativeSession.ts     terminal session driver (stdin/stdout, raw mode, pump)
├── buffer.ts (1.0k)     OptimizedBuffer over native cell grid + lease access
├── text-buffer.ts / text-buffer-view.ts / edit-buffer.ts / editor-view.ts
├── lib/ (28k)           color, borders, width, keymaps, input parsers, selection,
│                        extmarks, syntax, tree-sitter, scroll, clipboard, pulse
├── renderables/ (30k)   all widgets (see §3)
├── post/ (1.8k)         full-frame effects/filters/matrices (blur, etc.)
├── audio / image / video / animation / plugins / console
└── platform/            FFI loader (bun:ffi / node:ffi), runtime assets

@opentui/native (Zig, 139k)  the engine
├── context-abi.zig (5.2k)  C ABI for context/scene/session/buffer
├── scene.zig (3.9k)        scene graph, staging, layout, paint recording
├── renderer.zig (3.1k)     cell renderer + ANSI diff/output, damage tracking
├── buffer.zig (4.1k)       cell grid, drawing primitives, blending, images
├── yoga.zig + yoga-bridge  Yoga flexbox + C++ bridge
├── text-buffer*.zig, rope.zig, edit-buffer.zig, editor-view.zig
├── utf8.zig, grapheme.zig, encoded-unicode.zig, ansi.zig
├── input-parser.zig, terminal.zig, session.zig, renderer-output.zig
├── image.zig (+ stb/libwebp/lcms2), video.zig (+ ffmpeg), audio.zig (+ miniaudio)
├── clipboard/*.zig (x11, wayland, macos, windows)
├── embedded-terminal/*.zig (ghostty-vt) 
└── vendor/                 yoga, ghostty, stb, libwebp, lcms2, miniaudio, ffmpeg, wuffs
```

The Zig core is exposed to JS (and to anyone else) through a C ABI in
`packages/native/include/opentui.h` (3,485 lines). It has an explicit ABI version
(`OT_CONTEXT_ABI_VERSION 2`), context-qualified handles (`{context_id, slot, generation}`),
status codes, size/version-tagged option structs, and strict thread ownership. This
header is the seam a Go FFI backend would use. It is described as experimental/version 2.

Build/packaging today: Zig produces `libopentui` for 8 targets
(linux x64/arm64 gnu+musl, macOS x64/arm64, windows x64/arm64). The npm packages
`@opentui/core-<platform>` ship those binaries. Note: the C ABI header is not published
on npm (the API doc says the C ABI is "used from a source checkout"), so a Go binding
must vendor both the header and prebuilt libs, or build the Zig source itself.

---

## 3. Scope: what "core + widgets, 90% parity" means

"90%" must be a checklist, not a feeling. Here is my proposed parity matrix. P0 = required
for 90%; P1 = strongly desirable; P2 = out of scope for the first 90% (feature-gated/later).

### 3.1 Core engine (P0)

| Area | Source | Go port notes |
|---|---|---|
| RGBA / color parsing / blending / palettes | `lib/RGBA.ts`, `ansi.ts`, `terminal-palette.ts` | Pure Go, easy. Terminal palette (256 + truecolor fallback). |
| Cell grid / OptimizedBuffer | `buffer.zig`, `buffer.ts` | Struct-of-arrays or AoS cells: rune/width, fg, bg, attrs, link id, image ref. |
| ANSI/escape encoding + diff | `ansi.zig`, `renderer-output.zig`, `renderer.zig` | Damage tracking, minimal writes, synchronized output, alt-screen. Highest-value native parity target. |
| Unicode width / grapheme / encoding | `utf8.zig`, `grapheme.zig`, `encoded-unicode.zig`, `wcwidth` tables | Use `rivo/uniseg` + generated width tables; match `wcwidth`/`unicode`/`unicode-wide` modes. |
| Flexbox layout | `yoga.zig`, vendor yoga | Use native via FFI initially. Pure backend: port Yoga or a compatible engine (see §9). |
| Renderable tree / lifecycle / events | `Renderable.ts` (2.5k) | Faithful Go port: options, hooks, focus, z-order, absolute positioning, culling. |
| Renderer / event loop / scheduler | `renderer.ts` (6k) | Frames, fps cap, resize debounce, mouse hit-testing, capture/bubble, selections, cursor. |
| Scene staging + paint recording | `NativeScene.ts`, `scene.zig` | If FFI: use native scene. Pure: implement staging, hooks, paint slots, damage. |
| Input parsing | `input-parser.zig`, `lib/parse.keypress*.ts`, `parse.mouse.ts`, `stdin-parser.ts` | Keys (legacy + kitty flags), mouse SGR/X10/urxvt, paste, OSC, focus events. |
| Terminal session / raw mode / capabilities | `terminal.zig`, `session.zig`, `lib/terminal-capability-detection.ts` | raw mode via `golang.org/x/term`, SIGWINCH, DA1/XTGETTCAP/DSR queries, theme detection. |
| Text buffer / text buffer view | `text-buffer*.zig`, `text-buffer*.ts` | Rope or piece table, wrapping cache, tabs, highlights, selection, line info. |
| Edit buffer / editor view | `edit-buffer.zig`, `editor-view.zig`, TS wrappers | Cursor, undo/redo, extmarks, word wrap, selection gestures, keybindings. |
| Selection | `lib/selection.ts`, native selection | Cell/word/line behaviors, clipboard copy. |
| Animation / timeline | `animation/Timeline.ts` | Pure Go (goroutine/ticker + easing). |
| Console overlay | `console.ts` | Pure Go. |
| Post effects | `post/*.ts` | Blur, tint, matrix, filters over RGBA buffers. Pure Go (can be slow, CPU). |
| Plugins / core slots | `plugins/*.ts` | Pure Go registry; low parity value, but cheap. |
| Testing harness | `testing/*`, `TestRenderer` | Golden frames, snapshot comparison. Critical to the whole plan. |

### 3.2 Widgets

| Tier | Widget | Source LOC | Depends on | Notes |
|---|---|---|---|---|
| P0 | `BoxRenderable` | 382 | buffer, borders, layout | Borders, titles, focus border, fill. |
| P0 | `TextRenderable` / `TextNode` / `StyledText` | 163+412 | text buffer, ANSI, color | Styled chunks, links, spans. |
| P0 | `TextBufferRenderable` | 578 | text buffer/view, selection | Base for all text widgets. |
| P0 | `ScrollBoxRenderable` (+ parts) | 805+579 | Box, native scroll, ScrollBar | Viewport/wrapper/content/bar, culling, sticky, acceleration. |
| P0 | `ScrollBarRenderable` | 518 | native arrow/slider | Vertical/horizontal, units, drag. |
| P0 | `InputRenderable` | 277 | Textarea/EditBuffer | Single-line, min/max, placeholder. |
| P0 | `TextareaRenderable` | 429 | EditBufferRenderable, keybindings | Multi-line editing. |
| P0 | `EditBufferRenderable` | 1201 | edit buffer, extmarks, syntax | Base editor widget. |
| P0 | `SelectRenderable` | 473 | ascii font, keys | List select, scroll indicator, fast scroll. |
| P0 | `TabSelectRenderable` | 437 | Select | Tabs. |
| P0 | `SliderRenderable` | 295 | native slider node | Native slider thumb. |
| P0 | `ASCIIFontRenderable` | 233 | fonts JSON, buffer | Banner text. |
| P1 | `CodeRenderable` | 654 | `text.HighlightCode`, `text.SplitLines` | Dependency-free lexical syntax highlighting. |
| P1 | `DiffRenderable` | 1360 | text buffer | Unified line diff with bounded LCS memory. |
| P1 | `MarkdownRenderable` | 2310 | text, code, table | Built-in common block and inline Markdown subset. |
| P1 | `TextTableRenderable` | 1288 | text buffer, width | Tables with wrapping/alignment. |
| P1 | `LineNumberRenderable` | 772 | scrollbox, text view | Gutter line numbers. |
| P1 | `ImageRenderable` | 309 | image codecs, kitty/sixel/blocks | PNG/JPEG/WebP/GIF, fit modes, protocols. |
| P1 | `FrameBufferRenderable` | 74 | buffer/image | Raw RGBA drawing. |
| P1 | `EmbeddedTerminalRenderable` | 520 | ghostty-vt / VT emulator | Full terminal widget; hardest media item. |
| P2 | `VideoRenderable` | 497 | ffmpeg | Frame extraction + audio; use ffmpeg CLI/cgo. |
| P2 | `TimeToFirstDrawRenderable` | 108 | — | Lower priority. |
| P2 | composition (`VRenderable`, vnode) | ~ | declarative nodes | Optional, only if a declarative API is wanted. |

### 3.3 Explicitly out of scope for the first 90%

- `@opentui/react`, `@opentui/solid` (no React/Solid in Go; optional declarative layer later).
- `@opentui/three` (WebGPU 3D).
- `@opentui/ssh` (the Go port can use `golang.org/x/crypto/ssh` later; cheap, but not core).
- `@opentui/qrcode` (cheap via `skip2/go-qrcode`, later).
- Tree-sitter WASM worker model: use the built-in lightweight lexer/parser; consider
  optional grammar dependencies later if exact language coverage is needed.
- Exact pixel/GPU post effects: CPU effects only, may not match 3D.

---

## 4. Backend options (the real decision)

There are four credible strategies. I compare them on the things that matter: time to parity,
distribution, correctness risk, and long-term maintenance.

### Option A — Pure Go from scratch (reimplement Zig + TS)

- Rewrite all ~139k Zig + ~40k TS (non-test) logic in Go, replacing Yoga, ghostty-vt,
  stb/libwebp/lcms2, miniaudio, ffmpeg with Go libraries or new code.
- **Pros:** single static binary, cross-compiles, no native artifacts, no experimental ABI,
  no cgo, easiest install (`go install`), full control, best long-term story, idiomatic Go.
- **Cons:** Yoga-compatible flexbox is a huge, subtle task; terminal emulation (ghostty-vt)
  is enormous; video/audio are hard without cgo; text/rope/wrap/selection parity is fiddly;
  ANSI diff parity and damage tracking are easy to get *almost* right and hard to get exact.
- **Time to 90%:** ~12–18 months solo. **Risk:** high.

### Option B — cgo bindings to the existing Zig core

- Vendor `opentui.h` + prebuilt `libopentui` per platform (or build Zig in CI), call via cgo.
- Go reimplements the TS layer (renderable tree, renderer loop, widgets); Zig does layout,
  scene paint, buffer, text, input, image/video/audio/clipboard/terminal.
- **Pros:** fastest path to real visual parity; reuses battle-tested native code; Go code is
  mostly the TS port you must write anyway; callbacks are straightforward with cgo; all
  args are pointers (verified: no by-value structs), so ABI is simple.
- **Cons:** requires a C toolchain + per-platform native lib; `go install` alone won't work;
  ABI is marked experimental v2 (pin a commit, vendor header, generate bindings, check
  `ot_context_abi_version` at startup); callback/lifetime discipline across the boundary;
  cgo disables easy cross-compilation; cgo call overhead (mitigate by batching buffer ops).
- **Time to 90%:** ~6–9 months. **Risk:** medium-low.

### Option C — purego dynamic FFI to the same Zig core (no cgo)

- Same as B, but load `libopentui` at runtime with `github.com/ebitengine/purego`.
- **Pros:** no C toolchain, still a normal Go build; works on linux/macOS/windows amd64/arm64;
  callbacks via `purego.NewCallback`; can embed/discover binaries.
- **Cons:** slower than cgo for many tiny calls (batch!), callback setup is more constrained;
  still ships a native artifact per platform; same experimental-ABI risk; less familiar to
  contributors; purego's struct-by-value support would matter, but the ABI uses pointers.
- **Time to 90%:** ~7–10 months. **Risk:** medium.

### Option D — Hybrid, interface-first, FFI first then pure Go (recommended)

- Define a small Go `Backend` interface (layout, scene paint, buffer leases, text buffer,
  image, input, terminal). Implement it twice:
  1. `backend/native` — purego (or cgo behind a build tag) to `libopentui`.
  2. `backend/pure` — pure Go, built incrementally, starting with the easy pieces.
- All renderables/widgets/renderer code target the interface, never the ABI directly.
- **Pros:** fastest parity, de-risked; every pure-Go milestone immediately usable by users
  who opt in; lets you swap backends per subsystem; the interface is the real long-term asset;
  parity tests run against both backends.
- **Cons:** interface design cost up front; some native features (video/audio/terminal) may be
  hard to express identically in pure Go; risk of "leaky" interface if not disciplined.
- **Time to 90%:** ~6–9 months (native backend), +3–5 months to make pure the default.
  **Risk:** medium, with a clean fallback.

### Recommendation

**Option D, starting with the purego native backend.** Concretely:

- Keep a single public Go package that looks like `@opentui/core`.
- Put every native touch behind `internal/backend`.
- Build `backend/native` first; it is the parity yardstick.
- Build `backend/pure` subsystem-by-subsystem, each gated by the same golden tests.
- Offer build tags: default = native (fast parity), `-tags gotui_pure` = pure Go.

Why purego over cgo for the native backend: the ABI is pointer-only, the binary is already
shipped per platform, and purego keeps `go build` clean and cross-platform-ish without a C
toolchain. Keep a cgo implementation behind `-tags gotui_cgo` as a correctness fallback and for
platforms where purego callback/ABI behavior is suspect.

Why not "just reimplement everything in Go": the 139k Zig LOC is not the deliverable the user
asked for (they asked for core + widgets at 90%); rewriting ghostty-vt and Yoga would dominate
the schedule and add the most risk for the least visible parity. Reuse them until the widget
layer is done, then replace.

---

## 5. Target Go architecture

### 5.1 Repository layout

```
gotui/
  go.mod                       module github.com/<you>/gotui
  tui.go / tui/                public facade: CreateRenderer, Renderable, widgets
  options.go                   RendererOptions, LayoutOptions, RenderableOptions
  color/                       RGBA, parse, blend, terminal palette
  width/                       grapheme + display width (uniseg + tables)
  ansi/                        sequence constants, encoder, diff, synchronized output
  cell/                        Cell, Grid, Buffer (optimized, stack, lease)
  layout/                      Yoga-compatible Node/Style/Measure types
  renderable/                  Base, Renderable, Root, hooks, focus, z-order, culling
  renderer/                    Renderer, scheduler, loop, resize, mouse, selection, cursor
  input/                       key/mouse/paste parser, kitty protocol, key events, keymap
  term/                        raw mode, capabilities, queries, signals, OSC, theme
  text/                        Rope, TextBuffer, TextView, EditBuffer, EditorView,
                               Selection, Extmarks, SyntaxStyle, wrap cache
  scene/                       staging, paint recorder, damage, paint slots
  backend/
    backend.go                 the interface
    native/                    purego bindings + structs generated from opentui.h
    native/cgo/                optional cgo parity fallback
    pure/                      pure Go implementations
  renderables/                 Box, Text, ScrollBox, Input, Textarea, Select, ...
  image/                       decode, resize, ICC, kitty/sixel/blocks encoders
  audio/                       optional
  video/                       optional
  clipboard/                   optional/platform
  post/                        effects, filters, matrices
  animation/                   timeline, easing, ticker
  plugins/                     registry, core slots
  testing/                     TestRenderer, golden harness, recorder
  examples/                    ported demos, one per widget
  internal/abigen/             generate Go structs/offsets from opentui.h
  internal/parity/             golden frame corpus + runner
```

Keep it flatter than the TS tree where Go conventions favor it; the public API should feel
like Go, not like a 1:1 TS transliteration.

### 5.2 The backend interface (sketch)

```go
// backend/backend.go
type Backend interface {
    NewContext(opts ContextOptions) (Context, error)
    ABI() ABIInfo // version, features
}

type Context interface {
    NewSession(opts SessionOptions) (Session, error)
    NewBuffer(opts BufferOptions) (Buffer, error)
    NewTextBuffer(opts TextBufferOptions) (TextBuffer, error)
    NewEditBuffer(opts EditBufferOptions) (EditBuffer, error)
    NewEditorView(opts ...) (EditorView, error)
    DecodeImage(data []byte, opts ImageOptions) (Image, error)
    Close() error
}

type Session interface {
    // terminal I/O
    SetupTerminal(opts TerminalOptions) error
    Suspend() error; Resume() error
    Input([]byte) error
    Pump() (PumpResult, error)
    Render(frame FrameRequest) (RenderStatus, error)
    Resize(w, h int) error
    Output() ([]byte, error)
    Close() error
}

type Buffer interface {
    Resize(w, h int)
    AcquireLease() (Lease, error) // direct cell access: char/fg/bg/attrs
    Draw(...)                     // batched draw records
    Destroy()
}

type Scene interface {
    CreateNode(parent Node, kind NodeKind, opts NodeOptions) (Node, error)
    MoveNode(node, parent Node, index int) error
    DestroyNode(node Node) error
    SetStyle(node Node, key StyleKey, value StyleValue) error
    SetText(node Node, chunks []StyledChunk, opts TextOptions) error
    SetPaint(node Node, p PaintOptions) error
    SetMeasure(node Node, fn MeasureFunc) error
    SetScroll(node Node, opts ScrollOptions) error
    Frame(req FrameRequest) (Frame, error) // paint slots, geometry, damage
    Commit(frame Frame) (Status, error)
    HitTest(x, y int) (Node, error)
}
```

Rules:
- Widgets never call the ABI. They call `scene`, `text`, `buffer`.
- Batch draw calls: build a `DrawList` (clear/fill/text/box/image/alpha/compose) and submit once
  per node, exactly like the native `ot_buffer_draw_*` record stream.
- Callbacks (measure, hooks) are registered once and dispatched on the owner goroutine.
- The renderer owns exactly one goroutine per context; the backend contract says "one OS
  thread owns a context" (same as the native ABI). Do not spread calls across goroutines.

### 5.3 Go API style

Mirror the *concepts* but make them Go-idiomatic:

```go
r, _ := tui.NewRenderer(tui.WithFPS(60), tui.WithMouse(true))
defer r.Close()

root := renderables.NewBox(renderables.BoxOptions{
    ID: "root", FlexDirection: "column", Padding: tui.All(1),
})
title := renderables.NewText("Hello, OpenTUI", renderables.TextOptions{FG: "#ffcc00"})
input := renderables.NewInput(renderables.InputOptions{Placeholder: "type..."})
root.Add(title, input)
r.Root().Add(root)
r.Start()
```

Events: Go interfaces / callbacks instead of EventEmitter:
```go
input.OnChange(func(v string) { ... })
input.OnKeyDown(func(e *input.KeyEvent) bool { ... })
```
or `r.On(tui.EventKey, handler)` if an emitter is preferred. Pick one and keep it consistent.

---

## 6. Subsystem port plans

### 6.1 Color, width, ANSI (start here)

- Port `RGBA.ts`: float32/uint16 channels, `fromInts`, `fromHex`, `parseColor` (named, hex,
  `rgb()`, `hsl()` if present), `blend`, `withAlpha`, equality.
- Terminal palette: 16 ANSI + 256 xterm + truecolor detection and downsampling.
- Width: `rivo/uniseg` for grapheme clusters; generate wcwidth tables from the same Unicode
  version the Zig core uses; implement `wcwidth`, `unicode`, `unicode-wide` modes with the
  same ambiguous-width policy; handle ZWJ, emoji, combining marks, tabs, control chars.
- ANSI: move/clear/SGR builders, OSC 8 hyperlinks, kitty graphics, synchronized update,
  cursor shape, bracketed paste, mouse modes, alt/main screen.
- Diff encoder: implement the same damage model as `renderer-output.zig` (compare previous
  frame, emit only changed runs, reuse SGR state, handle wide cells and trailing spaces).
  **This is the single most important parity target; test with recorded golden streams.**

### 6.2 Cell grid + buffer

- Cell layout: `rune`/glyph id + width, fg RGBA, bg RGBA, attrs uint32, link id,
  image layer ref. Use parallel slices (`[]uint32 char`, `[]uint16 fg`, ...) like the JS
  `BufferAccess` for cheap snapshots and diffing; or AoS for cache locality. Benchmark both.
- Drawing primitives: text (styled runs), box borders, fill, alpha blend, grayscale,
  color matrix, supersample, image, grid, compose/draw-buffer, editor/text-view draws.
- Buffer stack and clear/scroll operations.
- Lease API for zero-copy reads from the renderer and for test snapshots.

### 6.3 Layout (Yoga)

Native path: use `ot_scene_set_style` / `ot_scene_get_layout` / measure callbacks; the Go
`layout` package only holds option structs and conversion.
Pure path: the cheapest credible route is a **Go port of Yoga's algorithm** (or a compatible
subset). Do not invent a new flexbox; parity with Yoga's rounding, percentages, `auto` margins,
baseline, caching, and pixel-grid rounding is where bugs live. Evaluate:
1. Port Yoga C++ (~15k) to Go with the algorithm files (CalculateLayout, FlexLine, Cache,
   AbsoluteLayout, PixelGrid, Baseline). MIT license. Highest fidelity, most work.
2. Link Yoga via cgo and keep pure-Go-only for a later phase (defeats pure builds).
3. Use an existing Go flexbox (few are Yoga-compatible; expect parity gaps).
Recommendation: for the pure backend, port Yoga; budget 6–10 weeks and a big golden test
suite (`packages/core/src/tests/yoga-upstream` is a ready-made corpus).

### 6.4 Renderable tree

Faithful port of `Renderable.ts`:
- Options: layout (all flex/margin/padding/position/size), z-index, visibility, opacity,
  buffered/live, animation fps.
- Lifecycle: construct → attach → layout → paint hooks (`renderBefore`/`renderAfter`) → resize →
  destroy; rollback construction on error (the TS does this; replicate).
- Hooks: `onMouse*`, `onKeyDown`, `onPaste`, `onSizeChange`, focus/blur events.
- Composition: `buffered` nodes render to their own buffer then composite (needed for
  opacity/transforms); `live` nodes animate.
- Culling: viewport culling for ScrollBox; subtree culling flags.
- Event dispatch: capture then bubble, `stopPropagation`/`preventDefault` semantics,
  `MouseEvent` with target/source/isDragging, click repeat, drag/drop.
- Focus management: focusable nodes, focus chain, focus restore, auto-focus.

### 6.5 Renderer / event loop

Port `renderer.ts` behavior, not its every internal:
- `NewRenderer(stdin, stdout, w, h, config)`, `Start`, `Stop`, `Destroy`, `Pause`, `Resume`.
- Frame scheduler: target fps, min frame time, idle detection, immediate re-render, output
  backpressure, synchronized output, frame callbacks, `RequestAnimationFrame`.
- Terminal setup: raw mode, alt-screen vs main-screen vs split-footer, mouse enable/disable,
  kitty keyboard flags, focus events, bracketed paste, cursor hide/show/shape/color.
- Resize: SIGWINCH (Unix) / console events (Windows), debounce, pixel resolution queries.
- Capability detection: DA1, XTGETTCAP, DECRQM, kitty graphics query, theme mode (OSC 11),
  multiplexer detection (tmux/zellij/screen).
- Selection: cell/word/line, drag, double/triple click, copy to clipboard.
- Split-footer/scrollback: the hardest renderer feature. Port last; gate behind a flag.
- Output capture: `capture-stdout`, passthrough, external writes.
- Death/restore: restore terminal on every exit path, including panics.

### 6.6 Input

- Port `stdin-parser.ts` + native parser behavior: UTF-8 incremental decoding, escape
  sequences split across reads, legacy vs kitty keyboard (flags 1/2/4/8/16), modifiers,
  alternate keys, release events, key aliases, focus in/out, mouse (X10, SGR, urxvt, SGR-Pixels),
  wheel, drag, paste (bracketed), OSC pass-through.
- `KeyHandler` map/aliases/actions (used by widgets), `parse.keypress.ts`, `parse.mouse.ts`.
- Keymap package: a compact action router (not full `@opentui/keymap` parity) initially.
- Golden vector tests: port `stdin-parser.conformance.json` and the parser tests; feed byte
  sequences and compare event structs.

### 6.7 Text engine

This is the second-biggest Go implementation after widgets.
- `Rope`: port `rope.zig` (piece tree / rope with UTF-8, grapheme-aware, line index, edit ops,
  fuzz-tested). Go alternatives exist but parity with wrap/selection is easier if the same
  model is used.
- `TextBuffer`: text + line starts + wrap cache + highlights + tab expansion + width modes +
  styled chunks + links.
- `TextBufferView`: viewport, scroll X/Y, wrap modes (none/char/word), alignment, truncate,
  tab indicators, line info, selection conversion, hit-testing, object-in-viewport.
- `EditBuffer`: cursor (logical/visual), insert/delete, atomic edits, history, extmarks,
  highlights, syntax styles, tab width, placeholder.
- `EditorView`: viewport, scroll margins, selection, replacements, commands, line numbers,
  measure.
- Port the extensive Zig test suite semantics (`text-buffer*_test.zig`, `rope*_test.zig`,
  `edit-buffer*_test.zig`, `editor-view*_test.zig`) as Go table tests.

### 6.8 Widgets

Port in dependency order (see §7). Each widget plan:

1. Read the TS implementation and its tests/snapshots.
2. Port options struct + defaults + validation (errors as Go errors).
3. Port layout/paint logic; use scene nodes where the TS uses native integration.
4. Port keybindings/actions and mouse behavior.
5. Port events/callbacks.
6. Port snapshot tests verbatim (frame comparison), then add parity corpus cases.

Watch for:
- The TS `nativeIntegration` static: some widgets draw natively (Box border/title, Slider,
  Arrow/ScrollBar, Image, Video, TextView, EditorView). The Go port should either use the
  same native nodes (FFI backend) or draw equivalents (pure backend) — keep that choice in
  `backend`, not in widget code.
- `ScrollBox` is really four renderables (`Wrapper/Viewport/Content/Bar`) with native scroll
  state; port those parts faithfully or you will fight scroll math forever.
- `Markdown`/`Diff`/`Code`/`TextTable` are pure logic over text buffers; they are large but
  low-risk. Prefer Go libraries for parsing (Goldmark, diff) but replicate OpenTUI's exact
  rendering and options.

### 6.9 Images, video, audio, terminal, clipboard

- Images: decode PNG/JPEG/GIF/WebP (`image/*` + `golang.org/x/image/webp` or pure Go webp),
  resize (`golang.org/x/image/draw` with premultiplied alpha), ICC (optional, `lcms2` via cgo
  or skip), then encode to kitty (raw/zlib/file), sixel, or blocks. Protocol negotiation and
  transport polling mirror `renderer.ts`.
- Video: P2. Use FFmpeg via cgo (`github.com/asticode/go-astiav`) or the `ffmpeg` CLI; extract
  frames to RGBA, feed the image pipeline; audio via a Go audio lib. Parity will be partial.
- Audio: P2. `github.com/ebitengine/oto/v3` or `github.com/gen2brain/malgo` (cgo). Streaming
  + mixer + groups + tap. Or delegate to native backend.
- Embedded terminal: P2/P1. Native backend reuses ghostty-vt. Pure Go would need a VT emulator
  (e.g. `github.com/hinshun/vt10x` is limited; parity with ghostty is unrealistic). Gate it as
  native-only until proven otherwise.
- Clipboard: native backend reuses it; pure Go uses `golang.design/x/clipboard` (cgo) or
  OSC 52 fallback + platform tools.

### 6.10 Post effects / animation / plugins / console

Mostly straightforward pure Go:
- Blur (box blur or separable Gaussian), tint, grayscale, color matrix, invert, contrast,
  saturation, custom filter chains. Match the TS formula and pass order.
- Timeline: `time.Ticker` or renderer tick; easing functions; `Animate`/`CreateTimeline`.
- Console overlay: ring buffer of lines drawn as a Box+Text overlay; toggle key.
- Plugins: registry with slot names; low priority.

---

## 7. Widget roadmap and phases

### Phase 0 — Foundations + parity harness (2–3 weeks)

Deliverables:
- Go module, CI (linux/macOS/windows amd64+arm64), lint, license/NOTICE.
- `internal/abigen`: parse `opentui.h` → Go structs with exact offsets + function tables.
- `backend/native` loads `libopentui` via purego; `ABI()` checks `ot_context_abi_version()==2`.
- `testing` harness: create a renderer with an in-memory sink at WxH, present frames, snapshot cell
  grids, compare to golden files.
- Golden corpus generator: run the *upstream TS examples* with the same sizes and dump their
  cell grids/ANSI (or use `TestRenderer`), commit them as testdata. This is the objective
  parity oracle.
- Port `color`, `width`, and ANSI constants; conformance tests.

Exit criteria: a Go program can create a native context/session/buffer, draw a box+text, commit
a frame, and match a committed golden snapshot byte-for-byte.

### Phase 1 — Core rendering + Box/Text (4–6 weeks)

- Renderable tree, Root, Box, Text, TextNode, StyledText.
- Renderer loop, resize, alt-screen, cursor, basic mouse/key events.
- Scene staging/paint via native backend; buffer leases.
- Border styles, titles, padding/margin/flex basics.
- Port `Box`, `Text`, `ASCIIFont`, `TimeToFirstDraw` snapshots.

Exit: `box`, `text`, `ascii-font`, `layout` demos at parity across sizes.

### Phase 2 — Text engine + editors (6–8 weeks)

- Rope, TextBuffer, TextBufferView, EditBuffer, EditorView, Selection, Extmarks, SyntaxStyle.
- Textarea, Input, EditBufferRenderable.
- Code, Diff, TextTable, Markdown, LineNumber (dependency-free implementations).
- Port text/edit/rope/editor golden tests.

Exit: `editor`, `input`, `textarea`, `code`, `diff`, `markdown`, `text-table` demos at parity.

### Phase 3 — Scroll + selection widgets (4–6 weeks)

- ScrollBox, ScrollBar, nested wheel scroll chaining. Scroll acceleration, sticky scroll,
  viewport culling, and child-into-view remain deferred.
- Select, TabSelect, Slider.
- Port scroll/select/slider snapshots; mouse drag tests.

Exit: `scrollbox`, `select`, `slider`, `line-number` demos at parity.

### Phase 4 — Media + terminal (6–10 weeks)

- Image (decode/resize/protocols), FrameBuffer, ImageRenderable.
- EmbeddedTerminal (native-only first), Clipboard.
- Audio, Video (P2, may be partial).
- Kitty graphics transport polling, sixel, blocks.

Exit: `image`, `framebuffer`, `embedded-terminal`, `clipboard` demos at parity (minus video
where declared partial).

### Phase 5 — Renderer advanced features (2–4 weeks)

- Split-footer/scrollback, external output capture/passthrough, kitty keyboard flags, focus
  events, theme detection, capability detection, synchronized output, debug overlay/stats.
- Port renderer/session tests.

Exit: `console`, `split-footer`, `scrollback`, `kitty` demos at parity.

### Phase 6 — Pure-Go backend (10–16 weeks, gated)

- `backend/pure` implementing layout (Yoga port), cell buffer, ANSI diff, scene/paint, text
  buffers, input parsing, unicode width, image codecs/protocols.
- Swap per subsystem behind the same interface; keep native available.
- Terminal emulation and video stay native-only until separately approved.

Exit: `-tags gotui_pure` builds a static binary; Phase 1–3 demos pass golden tests without
native libs. Declare pure parity level honestly (e.g. "core+widgets excluding video/terminal").

### Phase 7 — Ecosystem (optional)

- SSH server example, QR code, declarative layer, extra demos, docs.
- 3D/React/Solid: not planned.

---

## 8. Native FFI binding plan (details)

1. Vendor `packages/native/include/opentui.h` at the pinned commit; record the commit in
   `internal/abigen/VERSION`.
2. Vendor prebuilt `libopentui` for supported targets under
   `backend/native/lib/<os>-<arch>/` (or download on demand with checksums). Prefer
   `go:embed` for small libs? No — embed only the loader metadata; ship libs as separate
   module files or resolve from `@opentui/core-<platform>` if present.
3. `internal/abigen` parses the header into:
   - Go structs with explicit fields and `unsafe.Offsetof` assertions (fail at init on drift).
   - `var` function pointers + `RegisterLibFunc` calls.
   - constants (status codes, enums, limits).
   Compare generated offsets against `packages/core/src/native-abi.generated.ts` (same source).
4. Loader:
   - purego: `purego.Dlopen` in platform-specific files; `purego.RegisterLibFunc` for each.
   - cgo fallback: `//go:build gotui_cgo` with `#include "opentui.h"`.
   - Search order: env `GOTUI_NATIVE_LIB`, local `lib/`, npm package path, `LD_LIBRARY_PATH`.
5. Callbacks: measure, scene hooks (render before/after/resize/scroll/update), edit events,
   clipboard results, audio callbacks. All callbacks must be invoked on the owning goroutine
   (the native ABI says callbacks are synchronous, same-thread).
6. Lifetimes: every handle is context-qualified `{context_id, slot, generation}`. Wrap in Go
   types that check context identity and generation; never let a Go pointer escape into C
   beyond the call (except explicit callbacks kept alive in a registry).
7. Threading: one goroutine owns a context; terminate it explicitly; no finalizers that call
   into native from arbitrary goroutines. `runtime.KeepAlive` around borrowed buffers.
8. Startup guard: `ot_context_abi_version()` must equal 2; refuse otherwise with a clear error.
9. Error mapping: `ot_status` → Go errors with symbolic names; drain diagnostics per frame
   (`ot_context_drain_diagnostics`).
10. Batching: use staged scene properties + `ot_buffer_draw_*` record streams; never per-cell
    FFI calls. Acquire a buffer lease once per frame and write cells directly.

Risks: experimental ABI changes → pin + vendor; purego ABI quirks → keep cgo fallback;
platform lib availability → CI builds Zig from source (`zig` 0.16) and caches artifacts.

---

## 9. Pure-Go backend plan (details)

Order by value/risk:

1. **Cell buffer + ANSI diff** (low risk, high value). Port `buffer.zig` primitives and
   `renderer-output.zig` damage model. Golden-test against native output for random scenes.
2. **Unicode width/grapheme** (low risk). `uniseg` + generated tables; compare with native
   `encoded-unicode` on a large corpus.
3. **Text engine** (medium risk). Port rope/view/edit/editor. This is mostly deterministic and
   testable; reuse the Zig tests as specs.
4. **Layout** (high risk). Port Yoga algorithm files; run the upstream Yoga corpus and compare
   node rects to native `ot_scene_get_layout`. Budget the most time here.
5. **Scene staging/paint/hooks** (medium). Pure Go version of the scene contract: nodes, style,
   measure callbacks, paint slots, damage, culling, scroll.
6. **Input parser** (medium). Pure Go port of the native parser + kitty; conformance vectors.
7. **Images** (medium). stdlib + x/image; kitty/sixel encoders; compare encoded bytes.
8. **Terminal emulation / video / audio** (high). Keep native or defer.

Gate: pure backend is "shipped" only when Phase 1–3 golden tests pass without native libs.
Do not claim full parity until video/terminal are resolved.

---

## 10. Testing and parity strategy

Parity is the whole game. Build the oracle first.

1. **Golden frame corpus**
   - For every upstream demo, render at several sizes (e.g. 80x24, 120x40, 200x60) and capture:
     - cell grid (char/fg/bg/attrs) after each committed frame,
     - the emitted ANSI byte stream (for diff-encoder parity),
     - layout rects per node,
     - key/mouse event structs for scripted input.
   - Store as compact binary/JSON testdata.
2. **Differential tests**
   - Run the same scripted scenario through the Go port and the native backend; diff snapshots.
   - For the pure backend, diff against the native backend.
3. **Unit/spec tests**
   - Port Zig test files as Go table tests: buffer, rope, text-buffer, editor-view, utf8,
     grapheme, input-parser, yoga, scene, renderer-output, image.
   - Port TS tests where they encode behavior (selection, scroll, widgets).
4. **Fuzz**
   - Rope edits, UTF-8 decode, ANSI parse, input parser, wrap. The Zig suite already fuzzes rope;
     mirror it.
5. **Interactive smoke tests**
   - Scripted PTY runs (`creack/pty`) of each demo; capture output; assert no panic and stable
     frames after N ticks.
6. **CI matrix**
   - linux/macOS/windows, amd64/arm64; native lib download/build cache; pure build tag job.

Definition of done for a widget: its demo renders at parity across the corpus and its
keyboard/mouse interactions produce identical event traces.

---

## 11. Milestones and effort

Assumes one experienced Go systems engineer, full-time. Ranges are honest, not marketing.

| Phase | Scope | Effort | Cumulative |
|---|---|---|---|
| 0 | Foundations, abigen, loader, golden harness | 2–3 w | 2–3 w |
| 1 | Renderer + Renderable + Box/Text | 4–6 w | 6–9 w |
| 2 | Text engine + Input/Textarea/Code/Diff/Markdown/Table | 6–8 w | 12–17 w |
| 3 | ScrollBox/ScrollBar/Select/TabSelect/Slider | 4–6 w | 16–23 w |
| 4 | Image/FrameBuffer/EmbeddedTerminal/Clipboard (+audio/video partial) | 6–10 w | 22–33 w |
| 5 | Split-footer, scrollback, capabilities, kitty, capture | 2–4 w | 24–37 w |
| 6 | Pure-Go backend (layout/text/buffer/input/images) | 10–16 w | 34–53 w |
| 7 | Ecosystem (SSH, QR, declarative, docs) | 3–6 w | 37–59 w |

So ~24–37 engineer-weeks (~6–9 months) to 90% parity on the native backend, and ~34–53
(~8–12 months) with a pure-Go backend. Add 30–50% for a less experienced team, more platforms,
or strict byte-for-byte ANSI parity.

What to cut if time is tight: video, audio, embedded terminal, split-footer/scrollback, exact
theme/capability detection, React/Solid/three. That still leaves a very usable core+widgets
library.

---

## 12. Risk register

| Risk | Impact | Likelihood | Mitigation |
|---|---|---|---|
| Native ABI is experimental and changes | Breaks bindings | Medium | Pin commit, vendor header+libs, generate offsets, ABI version guard, CI diff vs upstream |
| Yoga parity in pure Go | Layout drift | High | Port Yoga algorithm, use upstream corpus, defer pure backend |
| ANSI diff parity (wide chars, trailing cells, SGR state) | Visual glitches/perf | High | Golden byte-stream tests, property tests, compare against native |
| Text/rope/wrap/selection parity | Editor bugs | High | Port Zig tests as specs, fuzz, golden snapshots |
| Callback/lifetime bugs across FFI | Crashes | Medium | One-goroutine ownership, generation-checked handles, registries, no finalizers, cgo fallback |
| cgo/purego platform issues | Build friction | Medium | purego default; cgo opt-in; CI on all targets; prebuilt libs |
| Distribution/native artifact burden | Install pain | Medium | Pure backend as endgame; bundle per-platform modules; clear docs |
| Scope creep (video/3D/React) | Slips 90% | High | Explicit out-of-scope list, feature gates, parity matrix |
| Upstream divergence | Stale port | Medium | Track releases, port changelog, keep parity corpus regenerable |

---

## 13. Decisions needed from you

1. **Backend strategy.** Confirm Option D (hybrid, native-first) or choose A/B/C. This is the
   only decision that changes the architecture.
2. **Pure-Go requirement.** Must the final artifact be a single static Go binary (no native
   libs)? If yes, pure backend is in-scope from the start; if no, it becomes Phase 6 optional.
3. **Widget cutoff for "90%".** My P0+P1 list (Box, Text, ScrollBox/Bar, Input, Textarea,
   Select, TabSelect, Slider, ASCIIFont, Code, Diff, Markdown, TextTable, LineNumber, Image,
   FrameBuffer, EmbeddedTerminal) — confirm or trim. Video/audio/terminal can be excluded.
4. **Declarative layer.** Do you want a React-like/Elm-like builder API in Go, or is imperative
   widget composition enough for v1? (I recommend imperative first.)
5. **Public API naming.** Suggested module `github.com/<you>/gotui` with package `tui`; confirm
   the import path and whether you want a `cmd/` demo suite.
6. **Target platforms.** Confirm linux/macOS/windows, amd64+arm64, and whether musl matters.

Once 1–3 are answered, the next concrete step is Phase 0: scaffold the repo, generate the ABI
bindings, load the native lib with purego, and stand up the golden-parity harness against the
upstream demos.

---

## Appendix A — C ABI surface groups (from `opentui.h`)

- Context/lifecycle: `ot_context_create/destroy/abi_version`, diagnostics, last error.
- Buffers: `ot_buffer_create/destroy/resize/stack`, draw ops (text, unicode, box, grid, packed,
  scene body, text view, editor view, image, alpha, grayscale, supersample, color matrix),
  leases (acquire/release/validate/copy).
- Scene: create/destroy/move node, style get/set (enum/float/value/border/dimension), paint,
  box details, text (plain/styled/options/lines/info), slider, arrow, scroll, viewport,
  editor view, text view, image, video, surface, measure, hit test, layout, cursor, selection,
  stats, hooks, frame step/get slots/commit/cancel/copy/draw buffer, scroll to/request.
- Session: create/destroy/close/resume/cancel, attach renderer, terminal setup, input feed/reset/
  expect, pump/render/render split, output drain/complete, stdout, resize, capabilities,
  buffer lease, clipboard, kitty image transport, control, notifications, links, hit grid,
  terminal state/cursor.
- Text/edit/editor: text buffer create/set/get/insert/delete/measure/lines/info/selection;
  edit buffer create/set/get/insert/delete/range/history/position/highlights/syntax/tab width/
  cursor/measure/defaults/commands; editor view create/destroy/command/info/lines/position/
  selected text/selection/viewport/measure/replace/select/placeholder/scroll margin/tab color.
- Unicode: `ot_unicode_create/get/destroy`.
- Image: create pixels/decode/clone/retain/destroy/resize/extend/extract/composite/transform/
  copy pixels/png/info/inspect/update/take/release pixels.
- Video: info/state (via scene video node).
- Audio: engine start/stop/mixer, devices, capture, streams, load/play/voices/groups/volume/tap.
- Clipboard: service create/destroy/poll/drain/shutdown, read/write ops, results.
- Embedded terminal: create/destroy/write/resize/command/compose/cursor/responses/encode key/
  mouse/paste/focus/selection.

## Appendix B — Upstream test corpora to reuse as specs

- Native: `packages/native/src/tests/*_test.zig` (renderer, buffer, rope, text-buffer,
  text-buffer-view, edit-buffer, editor-view, utf8, grapheme, input-parser, yoga, scene*,
  image, terminal, session*).
- JS: `packages/core/src/**/*.test.ts` plus `__snapshots__` for widgets.
- Conformance: `lib/stdin-parser.conformance.json`, `tests/fixtures/text-corpus.json`,
  `tests/unicode-width-map.zon`.
- Yoga: `packages/core/src/tests/yoga-upstream` (MeasureCounter, utils) and `yoga_test.zig`.

## Appendix C — Licensing

OpenTUI is MIT. Vendored deps: Yoga MIT, ghostty MIT, stb public-domain/MIT, libwebp BSD,
lcms2 MIT, miniaudio public-domain/MIT, ffmpeg LGPL/ZLIB, wuffs Apache/MIT. A Go port must keep
the MIT copyright notice and the licenses of any vendored binaries or ported code. If we port
Yoga to Go, keep its MIT notice. If we link ffmpeg, respect LGPL (dynamic linking, notices).
