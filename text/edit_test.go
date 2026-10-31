package text

import "testing"

func TestEditInsertDelete(t *testing.T) {
	eb := NewEditBufferText("ab")
	eb.Insert("X")
	if got := eb.Value(); got != "abX" {
		t.Fatalf("Value = %q", got)
	}
	eb.Backspace()
	if got := eb.Value(); got != "ab" {
		t.Fatalf("after backspace = %q", got)
	}
	eb.SetCursor(1)
	eb.Insert("[]")
	if got := eb.Value(); got != "a[]b" {
		t.Fatalf("Value = %q", got)
	}
	if eb.Cursor != 3 {
		t.Fatalf("Cursor = %d, want 3", eb.Cursor)
	}
	eb.DeleteForward()
	if got := eb.Value(); got != "a[]" {
		t.Fatalf("after delete = %q", got)
	}
}

func TestEditSelectionReplace(t *testing.T) {
	eb := NewEditBufferText("hello world")
	eb.Select(6, 11)
	if got := eb.SelectedText(); got != "world" {
		t.Fatalf("SelectedText = %q", got)
	}
	eb.Insert("there")
	if got := eb.Value(); got != "hello there" {
		t.Fatalf("Value = %q", got)
	}
	if s, e, ok := eb.Selection(); ok {
		t.Fatalf("selection remains %d..%d", s, e)
	}
}

func TestEditBackspaceAcrossLines(t *testing.T) {
	eb := NewEditBufferText("ab\ncd")
	eb.SetCursor(3)
	eb.Backspace()
	if got := eb.Value(); got != "abcd" {
		t.Fatalf("Value = %q", got)
	}
	if eb.Cursor != 2 {
		t.Fatalf("Cursor = %d, want 2", eb.Cursor)
	}
}

func TestEditDeleteForwardAcrossLines(t *testing.T) {
	eb := NewEditBufferText("ab\ncd")
	eb.SetCursor(2)
	eb.DeleteForward()
	if got := eb.Value(); got != "abcd" {
		t.Fatalf("Value = %q", got)
	}
}

func TestEditUndoRedo(t *testing.T) {
	eb := NewEditBufferText("abc")
	eb.Insert("d")
	eb.Insert("e")
	if got := eb.Value(); got != "abcde" {
		t.Fatalf("Value = %q", got)
	}
	if !eb.Undo() || eb.Value() != "abcd" {
		t.Fatalf("undo = %q", eb.Value())
	}
	if eb.Cursor != 4 {
		t.Fatalf("cursor after undo = %d, want 4", eb.Cursor)
	}
	if !eb.Undo() || eb.Value() != "abc" {
		t.Fatalf("undo = %q", eb.Value())
	}
	if !eb.Redo() || eb.Value() != "abcd" {
		t.Fatalf("redo = %q", eb.Value())
	}
	if !eb.Redo() || eb.Value() != "abcde" {
		t.Fatalf("redo = %q", eb.Value())
	}
	if eb.Redo() {
		t.Fatal("unexpected extra redo")
	}
}

func TestEditUndoDelete(t *testing.T) {
	eb := NewEditBufferText("hello")
	eb.SetCursor(2)
	eb.DeleteForward()
	if eb.Value() != "helo" {
		t.Fatalf("Value = %q", eb.Value())
	}
	eb.Undo()
	if eb.Value() != "hello" || eb.Cursor != 2 {
		t.Fatalf("after undo Value=%q Cursor=%d", eb.Value(), eb.Cursor)
	}
}

func TestEditWordOperations(t *testing.T) {
	eb := NewEditBufferText("foo bar baz")
	eb.SetCursor(11)
	eb.DeleteWordBackward()
	if got := eb.Value(); got != "foo bar " {
		t.Fatalf("Value = %q", got)
	}
	eb.SetCursor(0)
	eb.DeleteWordForward()
	if got := eb.Value(); got != " bar " {
		t.Fatalf("Value = %q", got)
	}
}

func TestEditMovementGoalColumn(t *testing.T) {
	eb := NewEditBufferText("abcdef\nx\nabcdef")
	eb.SetCursor(5)
	eb.MoveDown(false)
	line, col := eb.LineCol()
	if line != 1 || col != 1 {
		t.Fatalf("after first down = %d,%d, want 1,1", line, col)
	}
	eb.MoveDown(false)
	line, col = eb.LineCol()
	if line != 2 || col != 5 {
		t.Fatalf("goal column not preserved: %d,%d, want 2,5", line, col)
	}
}

func TestEditMovementUnicode(t *testing.T) {
	eb := NewEditBufferText("a👍🏽b")
	eb.SetCursor(0)
	eb.MoveRight(false)
	if eb.Cursor != 1 {
		t.Fatalf("Cursor = %d, want 1", eb.Cursor)
	}
	eb.MoveRight(false)
	if eb.Cursor != 9 {
		t.Fatalf("Cursor = %d, want 9", eb.Cursor)
	}
	eb.MoveLeft(false)
	if eb.Cursor != 1 {
		t.Fatalf("Cursor = %d, want 1", eb.Cursor)
	}
}

func TestEditSelectionMovement(t *testing.T) {
	eb := NewEditBufferText("hello")
	eb.SetCursor(1)
	eb.MoveRight(true)
	eb.MoveRight(true)
	s, e, ok := eb.Selection()
	if !ok || s != 1 || e != 3 {
		t.Fatalf("selection = %d..%d ok=%v", s, e, ok)
	}
	if got := eb.SelectedText(); got != "el" {
		t.Fatalf("SelectedText = %q", got)
	}
	eb.MoveLeft(true)
	s, e, _ = eb.Selection()
	if s != 1 || e != 2 {
		t.Fatalf("shrunk selection = %d..%d", s, e)
	}
}

func TestEditExtmarks(t *testing.T) {
	eb := NewEditBufferText("hello world")
	id := eb.AddExtmark(5, false) // before the space
	eb.SetCursor(0)
	eb.Insert(">> ")
	m, ok := eb.Extmark(id)
	if !ok || m.Offset != 8 {
		t.Fatalf("mark = %+v ok=%v", m, ok)
	}
	// A left-biased mark inserted at its offset stays before the text.
	id2 := eb.AddExtmark(0, false)
	eb.SetCursor(0)
	eb.Insert("a")
	m2, _ := eb.Extmark(id2)
	if m2.Offset != 0 {
		t.Fatalf("left-biased mark moved to %d", m2.Offset)
	}
	eb.RemoveExtmark(id2)
	if _, ok := eb.Extmark(id2); ok {
		t.Fatal("mark not removed")
	}
}

func TestEditMaxUndo(t *testing.T) {
	eb := NewEditBufferText("")
	eb.MaxUndo = 2
	eb.Insert("a")
	eb.Insert("b")
	eb.Insert("c")
	eb.Undo()
	eb.Undo()
	if eb.CanUndo() {
		t.Fatal("history should be capped at 2")
	}
	if got := eb.Value(); got != "a" {
		t.Fatalf("Value = %q, want a", got)
	}
}

func TestEditorViewCursor(t *testing.T) {
	eb := NewEditBufferText("abc\ndefgh")
	ev := NewEditorView(eb, 10, 2)
	eb.SetCursor(6) // line 1, col 2
	ev.EnsureCursorVisible()
	x, y, ok := ev.CursorPosition()
	if !ok || x != 2 || y != 1 {
		t.Fatalf("CursorPosition = %d,%d,%v", x, y, ok)
	}
}
