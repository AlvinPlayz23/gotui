// Package renderer implements the terminal renderer and event loop.
package renderer

import (
	"io"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/bijim/gotui/ansi"
	"github.com/bijim/gotui/cell"
	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/input"
	"github.com/bijim/gotui/renderable"
	"golang.org/x/term"
)

// Options configures a Renderer.
type Options struct {
	In           io.Reader
	Out          io.Writer
	Width        int
	Height       int
	FPS          int
	Mouse        bool
	AltScreen    bool
	Synchronized bool
	ColorMode    ansi.ColorMode
	Background   color.RGBA
}

// Renderer drives layout, painting, and terminal input/output.
type Renderer struct {
	opts Options
	root *renderable.Root
	buf  *cell.Buffer
	enc  *ansi.Encoder
	par  *input.Parser

	width, height int
	mu            sync.Mutex
	frameMu       sync.Mutex
	parseMu       sync.Mutex
	flushMu       sync.Mutex
	flushTimer    *time.Timer

	renderCh  chan struct{}
	stopCh    chan struct{}
	started   bool
	stopped   atomic.Bool
	wg        sync.WaitGroup
	closeOnce sync.Once

	focused     renderable.Renderable
	hovered     renderable.Renderable
	pressed     renderable.Renderable
	pressX      int
	pressY      int
	lastClickAt time.Time

	cursorX, cursorY int
	cursorVisible    bool

	inFile   *os.File
	outFile  *os.File
	oldState *term.State
	cleanup  []func()
}

// New creates a renderer. Width/Height are required unless In is a terminal.
func New(opts Options) *Renderer {
	if opts.FPS <= 0 {
		opts.FPS = 60
	}
	if opts.Background == (color.RGBA{}) {
		opts.Background = color.RGBA{}
	}
	width, height := opts.Width, opts.Height
	if f, ok := opts.In.(*os.File); ok {
		if w, h, err := term.GetSize(int(f.Fd())); err == nil && w > 0 {
			if width == 0 {
				width = w
			}
			if height == 0 {
				height = h
			}
		}
	}
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	r := &Renderer{
		opts:  optionsWithDefaults(opts),
		width: width, height: height,
		renderCh: make(chan struct{}, 1),
		stopCh:   make(chan struct{}),
		par:      input.NewParser(),
	}
	r.root = renderable.NewRoot()
	r.root.Base.Renderer = r
	r.buf = cell.New(width, height, r.opts.Background)
	r.enc = ansi.NewEncoder(r.opts.Out, ansi.EncoderOptions{ColorMode: r.opts.ColorMode, Synchronized: r.opts.Synchronized})
	return r
}

func optionsWithDefaults(o Options) Options {
	if o.Out == nil {
		o.Out = io.Discard
	}
	if o.In == nil {
		o.In = nil
	}
	return o
}

// Root returns the root renderable.
func (r *Renderer) Root() *renderable.Root { return r.root }

// Width returns the current terminal width.
func (r *Renderer) Width() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.width
}

// Height returns the current terminal height.
func (r *Renderer) Height() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.height
}

// RequestRender schedules a frame.
func (r *Renderer) RequestRender() {
	select {
	case r.renderCh <- struct{}{}:
	default:
	}
}

// SetFocus focuses a renderable, blurring the previous one.
func (r *Renderer) SetFocus(target renderable.Renderable) {
	r.mu.Lock()
	old := r.focused
	r.focused = target
	r.mu.Unlock()
	if old == target {
		return
	}
	if old != nil {
		old.BaseNode().SetFocused(false)
	}
	if target != nil {
		target.BaseNode().SetFocused(true)
	}
	r.RequestRender()
}

// Focused returns the focused renderable.
func (r *Renderer) Focused() renderable.Renderable {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.focused
}

// SetCursor sets the terminal cursor for the next frame.
func (r *Renderer) SetCursor(x, y int, visible bool) {
	r.cursorX, r.cursorY, r.cursorVisible = x, y, visible
}

// Start enters the terminal and starts the event loop.
func (r *Renderer) Start() {
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return
	}
	r.started = true
	r.mu.Unlock()

	r.setupTerminal()
	r.RenderOnce()

	r.wg.Add(1)
	go r.loop()

	if r.opts.In != nil {
		go r.readInput()
	}
}

func (r *Renderer) setupTerminal() {
	if f, ok := r.opts.In.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		r.inFile = f
		if st, err := term.MakeRaw(int(f.Fd())); err == nil {
			r.oldState = st
			r.cleanup = append(r.cleanup, func() { _ = term.Restore(int(f.Fd()), st) })
		}
	}
	if f, ok := r.opts.Out.(*os.File); ok {
		r.outFile = f
	}
	var seq string
	if r.opts.AltScreen {
		seq += ansi.EnterAltScreen
	}
	seq += ansi.HideCursor + ansi.EnableBracketedPaste + ansi.EnableFocusEvents
	if r.opts.Mouse {
		seq += ansi.EnableMouse
	}
	_, _ = io.WriteString(r.opts.Out, seq)
	if r.opts.Mouse {
		r.cleanup = append(r.cleanup, func() { _, _ = io.WriteString(r.opts.Out, ansi.DisableMouse) })
	}
	r.cleanup = append(r.cleanup, func() {
		_, _ = io.WriteString(r.opts.Out, ansi.DisableFocusEvents+ansi.DisableBracketedPaste+ansi.ShowCursor)
	})
	if r.opts.AltScreen {
		r.cleanup = append(r.cleanup, func() { _, _ = io.WriteString(r.opts.Out, ansi.ExitAltScreen) })
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGWINCH, syscall.SIGINT, syscall.SIGTERM)
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for {
			select {
			case <-r.stopCh:
				return
			case s := <-sig:
				if s == syscall.SIGWINCH {
					r.querySize()
				} else {
					r.Stop()
					return
				}
			}
		}
	}()
	r.cleanup = append(r.cleanup, func() { signal.Stop(sig) })
}

func (r *Renderer) querySize() {
	if r.inFile == nil {
		return
	}
	if w, h, err := term.GetSize(int(r.inFile.Fd())); err == nil && w > 0 && h > 0 {
		r.Resize(w, h)
	}
}

// Resize updates the renderer size.
func (r *Renderer) Resize(w, h int) {
	r.frameMu.Lock()
	defer r.frameMu.Unlock()
	r.resize(w, h)
}

func (r *Renderer) resize(w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	r.mu.Lock()
	if w == r.width && h == r.height {
		r.mu.Unlock()
		return
	}
	r.width, r.height = w, h
	r.buf.Resize(w, h)
	r.mu.Unlock()
	r.enc.Invalidate()
	r.RequestRender()
}

// Done is closed when the renderer stops.
func (r *Renderer) Done() <-chan struct{} { return r.stopCh }

// Stop stops the event loop but does not restore the terminal.
func (r *Renderer) Stop() {
	r.stopped.Store(true)
	select {
	case <-r.stopCh:
	default:
		close(r.stopCh)
	}
	r.RequestRender()
}

// Close stops the renderer and restores the terminal. Safe to call twice.
func (r *Renderer) Close() {
	r.closeOnce.Do(func() {
		r.Stop()
		r.wg.Wait()
		for _, f := range r.cleanup {
			f()
		}
	})
}

func (r *Renderer) loop() {
	defer r.wg.Done()
	interval := time.Second / time.Duration(r.opts.FPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	dirty := true
	for {
		select {
		case <-r.stopCh:
			return
		case <-r.renderCh:
			r.RenderOnce()
			dirty = false
		case <-ticker.C:
			if r.root.Base.Update(interval.Seconds()) {
				dirty = true
			}
			if dirty {
				r.RenderOnce()
				dirty = false
			}
		}
	}
}

func (r *Renderer) readInput() {
	buf := make([]byte, 4096)
	for {
		n, err := r.opts.In.Read(buf)
		if n > 0 {
			r.parseMu.Lock()
			evs := r.par.Feed(buf[:n])
			pending := r.par.Pending()
			r.parseMu.Unlock()
			for _, ev := range evs {
				r.HandleEvent(ev)
			}
			if pending > 0 {
				r.scheduleFlush()
			}
		}
		if err != nil {
			return
		}
		select {
		case <-r.stopCh:
			return
		default:
		}
	}
}

// RenderOnce lays out and paints a single frame.
func (r *Renderer) RenderOnce() {
	r.frameMu.Lock()
	defer r.frameMu.Unlock()
	r.mu.Lock()
	w, h := r.width, r.height
	buf := r.buf
	r.mu.Unlock()

	// Fill background.
	buf.Clear(r.opts.Background)
	r.cursorVisible = false
	r.root.Node.Compute(w, h)
	ctx := renderable.NewContext(buf, r)
	ctx.Width, ctx.Height = w, h
	r.root.Paint(ctx)
	r.root.Base.Resize(w, h)

	r.mu.Lock()
	_ = r.enc.Render(buf)
	if r.cursorVisible {
		_, _ = io.WriteString(r.opts.Out, ansi.CursorTo(r.cursorX+1, r.cursorY+1)+ansi.ShowCursor)
	} else {
		_, _ = io.WriteString(r.opts.Out, ansi.HideCursor)
	}
	r.mu.Unlock()
}

// HandleEvent dispatches a parsed event.
func (r *Renderer) HandleEvent(ev input.Event) {
	r.frameMu.Lock()
	defer r.frameMu.Unlock()
	switch ev.Type {
	case input.EventResize:
		r.resize(ev.Width, ev.Height)
	case input.EventKey:
		r.handleKey(ev.Key)
	case input.EventPaste:
		r.handlePaste(ev.Paste)
	case input.EventMouse:
		r.handleMouse(ev.Mouse)
	}
}

func (r *Renderer) handleKey(ev input.KeyEvent) {
	focused := r.Focused()
	if focused != nil {
		if dispatchKey(focused, &ev) {
			r.RequestRender()
			return
		}
	}
	// Default focus traversal.
	if ev.Name == "tab" && !ev.Ctrl && !ev.Alt {
		if ev.Shift {
			r.focusPrev()
		} else {
			r.focusNext()
		}
		r.RequestRender()
		return
	}
	if ev.Ctrl && ev.Name == "c" && ev.Type != input.KeyRelease {
		r.Stop()
		return
	}
}

func dispatchKey(target renderable.Renderable, ev *input.KeyEvent) bool {
	for n := target; n != nil; n = n.BaseNode().Parent() {
		if ev.Type == input.KeyRelease {
			if n.HandleKeyUp(ev) {
				return true
			}
		} else if n.HandleKey(ev) {
			return true
		}
	}
	return false
}

func (r *Renderer) handlePaste(ev input.PasteEvent) {
	focused := r.Focused()
	if focused == nil {
		return
	}
	if dispatchPaste(focused, &ev) {
		r.RequestRender()
	}
}

func dispatchPaste(target renderable.Renderable, ev *input.PasteEvent) bool {
	for n := target; n != nil; n = n.BaseNode().Parent() {
		if n.HandlePaste(ev) {
			return true
		}
	}
	return false
}

func (r *Renderer) handleMouse(ev input.MouseEvent) {
	hit := r.root.HitTest(ev.X, ev.Y)
	if hit != r.hovered {
		if r.hovered != nil && r.hovered.BaseNode().OnMouseOut != nil {
			r.hovered.BaseNode().OnMouseOut(&ev)
		}
		r.hovered = hit
		if hit != nil && hit.BaseNode().OnMouseOver != nil {
			hit.BaseNode().OnMouseOver(&ev)
		}
		r.RequestRender()
	}
	switch ev.Action {
	case input.MousePress:
		if hit != nil {
			for n := hit; n != nil; n = n.BaseNode().Parent() {
				if n.BaseNode().Focusable {
					r.SetFocus(n)
					break
				}
			}
		}
		r.pressed = hit
		r.pressX, r.pressY = ev.X, ev.Y
		if hit != nil {
			dispatchMouse(hit, &ev)
		}
		r.RequestRender()
	case input.MouseRelease:
		if hit != nil {
			dispatchMouse(hit, &ev)
			if hit == r.pressed {
				now := time.Now()
				double := now.Sub(r.lastClickAt) < 400*time.Millisecond
				r.lastClickAt = now
				if double && hit.BaseNode().OnDoubleClick != nil {
					hit.BaseNode().OnDoubleClick(&ev)
				} else if hit.BaseNode().OnClick != nil {
					hit.BaseNode().OnClick(&ev)
				}
			}
		}
		r.pressed = nil
		r.RequestRender()
	case input.MouseMove:
		handled := false
		if r.pressed != nil && (ev.X != r.pressX || ev.Y != r.pressY) {
			handled = dispatchDrag(r.pressed, &ev)
		}
		if !handled && hit != nil {
			dispatchMouse(hit, &ev)
		}
		r.RequestRender()
	case input.MouseWheel:
		if hit != nil {
			dispatchMouse(hit, &ev)
		}
		r.RequestRender()
	}
}

func dispatchDrag(target renderable.Renderable, ev *input.MouseEvent) bool {
	for n := target; n != nil; n = n.BaseNode().Parent() {
		if n.BaseNode().OnMouseDrag != nil && n.BaseNode().OnMouseDrag(ev) {
			return true
		}
	}
	return false
}

func dispatchMouse(target renderable.Renderable, ev *input.MouseEvent) bool {
	for n := target; n != nil; n = n.BaseNode().Parent() {
		if n.HandleMouse(ev) {
			return true
		}
	}
	return false
}

func (r *Renderer) focusables() []renderable.Renderable {
	var out []renderable.Renderable
	var walk func(n renderable.Renderable)
	walk = func(n renderable.Renderable) {
		b := n.BaseNode()
		if !b.Visible {
			return
		}
		if b.Focusable {
			out = append(out, n)
		}
		for _, c := range b.Children() {
			walk(c)
		}
	}
	walk(r.root)
	return out
}

// FocusNext focuses the next focusable renderable.
func (r *Renderer) FocusNext() { r.focusNext() }

// FocusPrev focuses the previous focusable renderable.
func (r *Renderer) FocusPrev() { r.focusPrev() }

func (r *Renderer) focusNext() {
	list := r.focusables()
	if len(list) == 0 {
		return
	}
	cur := r.Focused()
	idx := -1
	for i, n := range list {
		if n == cur {
			idx = i
			break
		}
	}
	r.SetFocus(list[(idx+1)%len(list)])
}

func (r *Renderer) focusPrev() {
	list := r.focusables()
	if len(list) == 0 {
		return
	}
	cur := r.Focused()
	idx := 0
	for i, n := range list {
		if n == cur {
			idx = i
			break
		}
	}
	if idx == 0 {
		idx = len(list)
	}
	r.SetFocus(list[idx-1])
}

func (r *Renderer) scheduleFlush() {
	r.flushMu.Lock()
	if r.flushTimer == nil {
		r.flushTimer = time.AfterFunc(25*time.Millisecond, r.flushPending)
	}
	r.flushMu.Unlock()
}

func (r *Renderer) flushPending() {
	r.flushMu.Lock()
	r.flushTimer = nil
	r.flushMu.Unlock()

	r.parseMu.Lock()
	evs := r.par.Flush()
	pending := r.par.Pending()
	r.parseMu.Unlock()
	for _, ev := range evs {
		r.HandleEvent(ev)
	}
	if pending > 0 {
		r.scheduleFlush()
	}
}

// Buffer returns the current frame buffer (for tests).
func (r *Renderer) Buffer() *cell.Buffer { return r.buf }

// Parser returns the input parser.
func (r *Renderer) Parser() *input.Parser { return r.par }
