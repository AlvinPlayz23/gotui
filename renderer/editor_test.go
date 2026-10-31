package renderer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bijim/gotui/input"
	"github.com/bijim/gotui/renderables"
	"github.com/bijim/gotui/text"
)

func shiftKey(name string) input.Event {
	return input.Event{Type: input.EventKey, Key: input.KeyEvent{Name: name, Shift: true}}
}

func ctrlKey(name string) input.Event {
	return input.Event{Type: input.EventKey, Key: input.KeyEvent{Name: name, Ctrl: true}}
}

func mouseEvent(x, y int, action input.MouseAction) input.Event {
	return input.Event{Type: input.EventMouse, Mouse: input.MouseEvent{X: x, Y: y, Action: action}}
}

func newTestEditor(t *testing.T, opts renderables.EditorOptions) (*Renderer, *renderables.Editor) {
	t.Helper()
	r := New(Options{Out: &bytes.Buffer{}, Width: 40, Height: 6})
	e := renderables.NewEditor(opts)
	r.Root().Add(e)
	r.RenderOnce()
	r.SetFocus(e)
	return r, e
}

func TestEditorTypingAndUndo(t *testing.T) {
	r, e := newTestEditor(t, renderables.EditorOptions{})
	for _, ch := range "hello" {
		r.HandleEvent(runeKey(ch))
	}
	if got := e.Value(); got != "hello" {
		t.Fatalf("value = %q", got)
	}
	r.HandleEvent(key("left"))
	r.HandleEvent(runeKey('X'))
	if got := e.Value(); got != "hellXo" {
		t.Fatalf("after insert = %q", got)
	}
	r.HandleEvent(key("backspace"))
	if got := e.Value(); got != "hello" {
		t.Fatalf("after backspace = %q", got)
	}
	r.HandleEvent(ctrlKey("z"))
	if got := e.Value(); got != "hellXo" {
		t.Fatalf("after undo = %q", got)
	}
	r.HandleEvent(ctrlKey("z"))
	if got := e.Value(); got != "hello" {
		t.Fatalf("after second undo = %q", got)
	}
}

func TestEditorMultilineAndPaste(t *testing.T) {
	r, e := newTestEditor(t, renderables.EditorOptions{})
	for _, ch := range "ab" {
		r.HandleEvent(runeKey(ch))
	}
	r.HandleEvent(key("return"))
	r.HandleEvent(runeKey('c'))
	if got := e.Value(); got != "ab\nc" {
		t.Fatalf("value = %q", got)
	}
	r.HandleEvent(input.Event{Type: input.EventPaste, Paste: input.PasteEvent{Text: "d\ne"}})
	if got := e.Value(); got != "ab\ncd\ne" {
		t.Fatalf("after paste = %q", got)
	}
	r.RenderOnce()
	if got := r.Buffer().Text(); !contains(got, "ab") || !contains(got, "cd") {
		t.Fatalf("buffer = %q", got)
	}
}

func TestEditorShiftSelectionReplace(t *testing.T) {
	r, e := newTestEditor(t, renderables.EditorOptions{Value: "hello world"})
	e.SetCursor(0)
	for i := 0; i < 5; i++ {
		r.HandleEvent(shiftKey("right"))
	}
	if got := e.Edit.SelectedText(); got != "hello" {
		t.Fatalf("selection = %q", got)
	}
	r.HandleEvent(runeKey('X'))
	if got := e.Value(); got != "X world" {
		t.Fatalf("value = %q", got)
	}
}

func TestEditorWordAndDocumentKeys(t *testing.T) {
	r, e := newTestEditor(t, renderables.EditorOptions{Value: "one two three"})
	e.SetCursor(0)
	r.HandleEvent(input.Event{Type: input.EventKey, Key: input.KeyEvent{Name: "right", Ctrl: true}})
	if off := e.Edit.Cursor; off != 3 {
		t.Fatalf("word right cursor = %d, want 3", off)
	}
	r.HandleEvent(ctrlKey("end"))
	if off := e.Edit.Cursor; off != len("one two three") {
		t.Fatalf("doc end cursor = %d", off)
	}
	r.HandleEvent(ctrlKey("a"))
	if got := e.Edit.SelectedText(); got != "one two three" {
		t.Fatalf("select all = %q", got)
	}
}

func TestEditorMouseClickAndDrag(t *testing.T) {
	r, e := newTestEditor(t, renderables.EditorOptions{Value: "hello\nworld"})
	r.HandleEvent(mouseEvent(2, 0, input.MousePress))
	r.HandleEvent(mouseEvent(2, 0, input.MouseRelease))
	if line, col := e.Edit.LineCol(); line != 0 || col != 2 {
		t.Fatalf("click cursor = %d,%d, want 0,2", line, col)
	}
	r.HandleEvent(mouseEvent(1, 0, input.MousePress))
	r.HandleEvent(mouseEvent(3, 1, input.MouseMove))
	if got := e.Edit.SelectedText(); got != "ello\nwor" {
		t.Fatalf("drag selection = %q", got)
	}
}

func TestEditorLineNumbers(t *testing.T) {
	r, _ := newTestEditor(t, renderables.EditorOptions{
		Value:       "alpha\nbeta",
		LineNumbers: true,
	})
	buf := r.Buffer().Text()
	if !contains(buf, "1 alpha") || !contains(buf, "2 beta") {
		t.Fatalf("buffer = %q", buf)
	}
}

func TestEditorWrapAndScroll(t *testing.T) {
	long := strings.Repeat("abcdefghij", 10)
	r, e := newTestEditor(t, renderables.EditorOptions{
		Value: long,
		Wrap:  text.WrapChar,
	})
	if got := e.View.View.LineCount(); got < 2 {
		t.Fatalf("wrapped lines = %d", got)
	}
	e.SetCursor(len(long) - 1)
	r.RenderOnce()
	if e.View.View.ScrollX != 0 {
		t.Fatalf("horizontal scroll = %d, wrap should not scroll x", e.View.View.ScrollX)
	}
}

func TestEditorPaintsCursorPosition(t *testing.T) {
	r, e := newTestEditor(t, renderables.EditorOptions{Value: "abc\nxyz"})
	e.SetCursor(6) // before z
	r.RenderOnce()
	if x, y, ok := e.View.CursorPosition(); !ok || x != 2 || y != 1 {
		t.Fatalf("cursor = %d,%d,%v", x, y, ok)
	}
}
