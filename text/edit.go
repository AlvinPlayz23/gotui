package text

// Extmark is a position that moves with edits.
type Extmark struct {
	ID     int
	Offset int
	// Bias decides where the mark lands when text is inserted exactly at its
	// offset. BiasRight keeps it after the inserted text.
	BiasRight bool
	Data      any
}

// EditOp is one undoable change.
type EditOp struct {
	start        int
	inserted     string
	deleted      string
	cursorBefore int
	cursorAfter  int
	anchorBefore int
	anchorAfter  int
}

// EditBuffer is a text buffer with a cursor, selection, undo/redo, and
// extmarks. Offsets are byte offsets; the cursor always sits on a grapheme
// boundary.
type EditBuffer struct {
	Text    *TextBuffer
	Cursor  int
	anchor  int // -1 when there is no selection
	goalCol int // desired column for vertical movement, -1 when unset

	undo []EditOp
	redo []EditOp
	// MaxUndo limits the undo history. Zero means unlimited.
	MaxUndo int

	marks    []Extmark
	nextMark int
	OnChange func()
}

// NewEditBuffer creates an empty edit buffer.
func NewEditBuffer() *EditBuffer {
	return &EditBuffer{Text: NewTextBuffer(""), anchor: -1, goalCol: -1}
}

// NewEditBufferText creates an edit buffer containing s.
func NewEditBufferText(s string) *EditBuffer {
	eb := NewEditBuffer()
	eb.Text.SetText(s)
	eb.Cursor = eb.Text.Len()
	return eb
}

// Value returns the buffer text.
func (e *EditBuffer) Value() string { return e.Text.Text() }

// Len returns the byte length.
func (e *EditBuffer) Len() int { return e.Text.Len() }

// SetValue replaces the content, clears history, and moves the cursor to the
// end.
func (e *EditBuffer) SetValue(s string) {
	e.Text.SetText(s)
	e.Cursor = e.Text.Len()
	e.anchor = -1
	e.goalCol = -1
	e.undo = nil
	e.redo = nil
	e.marks = nil
	e.changed()
}

// SetCursor moves the cursor and clears the selection.
func (e *EditBuffer) SetCursor(off int) {
	e.Cursor = clampInt(off, 0, e.Text.Len())
	e.anchor = -1
	e.goalCol = -1
}

// Select sets an explicit selection.
func (e *EditBuffer) Select(start, end int) {
	start, end = clampRange(start, end, e.Text.Len())
	e.anchor = start
	e.Cursor = end
	e.goalCol = -1
}

// SelectAll selects the whole buffer.
func (e *EditBuffer) SelectAll() {
	e.Select(0, e.Text.Len())
}

// Selection returns the selected byte range.
func (e *EditBuffer) Selection() (start, end int, ok bool) {
	if e.anchor < 0 || e.anchor == e.Cursor {
		return 0, 0, false
	}
	start, end = e.anchor, e.Cursor
	if start > end {
		start, end = end, start
	}
	return start, end, true
}

// ClearSelection removes the selection without moving the cursor.
func (e *EditBuffer) ClearSelection() { e.anchor = -1 }

// SelectedText returns the selected text.
func (e *EditBuffer) SelectedText() string {
	start, end, ok := e.Selection()
	if !ok {
		return ""
	}
	return e.Text.Slice(start, end)
}

// LineCol returns the cursor's line and byte column.
func (e *EditBuffer) LineCol() (line, col int) { return e.Text.Position(e.Cursor) }

// Insert inserts s at the cursor, replacing the selection.
func (e *EditBuffer) Insert(s string) {
	if s == "" {
		return
	}
	start, end := e.Cursor, e.Cursor
	if a, b, ok := e.Selection(); ok {
		start, end = a, b
	}
	e.apply(start, end, s)
}

// DeleteSelection removes the selection. It reports whether anything was
// removed.
func (e *EditBuffer) DeleteSelection() bool {
	start, end, ok := e.Selection()
	if !ok {
		return false
	}
	e.apply(start, end, "")
	return true
}

// Backspace deletes the grapheme before the cursor, or the selection.
func (e *EditBuffer) Backspace() bool {
	if e.DeleteSelection() {
		return true
	}
	if e.Cursor <= 0 {
		return false
	}
	from := e.Text.PrevGrapheme(e.Cursor)
	e.apply(from, e.Cursor, "")
	return true
}

// DeleteForward deletes the grapheme after the cursor, or the selection.
func (e *EditBuffer) DeleteForward() bool {
	if e.DeleteSelection() {
		return true
	}
	if e.Cursor >= e.Text.Len() {
		return false
	}
	to := e.Text.NextGrapheme(e.Cursor)
	e.apply(e.Cursor, to, "")
	return true
}

// DeleteWordBackward deletes the previous word.
func (e *EditBuffer) DeleteWordBackward() bool {
	if e.DeleteSelection() {
		return true
	}
	from := e.Text.WordBoundaryLeft(e.Cursor)
	if from == e.Cursor {
		return false
	}
	e.apply(from, e.Cursor, "")
	return true
}

// DeleteWordForward deletes the next word.
func (e *EditBuffer) DeleteWordForward() bool {
	if e.DeleteSelection() {
		return true
	}
	to := e.Text.WordBoundaryRight(e.Cursor)
	if to == e.Cursor {
		return false
	}
	e.apply(e.Cursor, to, "")
	return true
}

// MoveLeft moves one grapheme left, extending the selection when extend is
// true.
func (e *EditBuffer) MoveLeft(extend bool) {
	e.moveTo(e.Text.PrevGrapheme(e.Cursor), extend)
	e.goalCol = -1
}

// MoveRight moves one grapheme right.
func (e *EditBuffer) MoveRight(extend bool) {
	e.moveTo(e.Text.NextGrapheme(e.Cursor), extend)
	e.goalCol = -1
}

// MoveWordLeft moves to the start of the previous word.
func (e *EditBuffer) MoveWordLeft(extend bool) {
	e.moveTo(e.Text.WordBoundaryLeft(e.Cursor), extend)
	e.goalCol = -1
}

// MoveWordRight moves to the end of the next word.
func (e *EditBuffer) MoveWordRight(extend bool) {
	e.moveTo(e.Text.WordBoundaryRight(e.Cursor), extend)
	e.goalCol = -1
}

// MoveUp moves up one logical line, preserving the desired column.
func (e *EditBuffer) MoveUp(extend bool) {
	line, col := e.LineCol()
	if line == 0 {
		e.moveTo(0, extend)
		return
	}
	if e.goalCol < 0 {
		e.goalCol = col
	}
	target := e.Text.Offset(line-1, e.goalCol)
	e.moveTo(target, extend)
}

// MoveDown moves down one logical line, preserving the desired column.
func (e *EditBuffer) MoveDown(extend bool) {
	line, col := e.LineCol()
	if line >= e.Text.LineCount()-1 {
		e.moveTo(e.Text.Len(), extend)
		return
	}
	if e.goalCol < 0 {
		e.goalCol = col
	}
	target := e.Text.Offset(line+1, e.goalCol)
	e.moveTo(target, extend)
}

// MoveHome moves to the start of the logical line.
func (e *EditBuffer) MoveHome(extend bool) {
	line, _ := e.LineCol()
	e.moveTo(e.Text.LineStart(line), extend)
	e.goalCol = -1
}

// MoveEnd moves to the end of the logical line.
func (e *EditBuffer) MoveEnd(extend bool) {
	line, _ := e.LineCol()
	e.moveTo(e.Text.LineEnd(line), extend)
	e.goalCol = -1
}

// MoveDocStart moves to the start of the buffer.
func (e *EditBuffer) MoveDocStart(extend bool) {
	e.moveTo(0, extend)
	e.goalCol = -1
}

// MoveDocEnd moves to the end of the buffer.
func (e *EditBuffer) MoveDocEnd(extend bool) {
	e.moveTo(e.Text.Len(), extend)
	e.goalCol = -1
}

// MoveTo moves the cursor to off, extending the selection when extend is true.
func (e *EditBuffer) MoveTo(off int, extend bool) { e.moveTo(off, extend) }

func (e *EditBuffer) moveTo(off int, extend bool) {
	off = clampInt(off, 0, e.Text.Len())
	if extend {
		if e.anchor < 0 {
			e.anchor = e.Cursor
		}
	} else {
		e.anchor = -1
	}
	if e.Cursor != off {
		e.Cursor = off
	}
}

// CanUndo reports whether an undo is available.
func (e *EditBuffer) CanUndo() bool { return len(e.undo) > 0 }

// CanRedo reports whether a redo is available.
func (e *EditBuffer) CanRedo() bool { return len(e.redo) > 0 }

// Undo reverts the last edit.
func (e *EditBuffer) Undo() bool {
	if len(e.undo) == 0 {
		return false
	}
	op := e.undo[len(e.undo)-1]
	e.undo = e.undo[:len(e.undo)-1]
	e.Text.Replace(op.start, op.start+len(op.inserted), op.deleted)
	e.Cursor = op.cursorBefore
	e.anchor = op.anchorBefore
	e.goalCol = -1
	e.redo = append(e.redo, op)
	e.changed()
	return true
}

// Redo reapplies the last undone edit.
func (e *EditBuffer) Redo() bool {
	if len(e.redo) == 0 {
		return false
	}
	op := e.redo[len(e.redo)-1]
	e.redo = e.redo[:len(e.redo)-1]
	e.Text.Replace(op.start, op.start+len(op.deleted), op.inserted)
	e.Cursor = op.cursorAfter
	e.anchor = op.anchorAfter
	e.goalCol = -1
	e.undo = append(e.undo, op)
	e.changed()
	return true
}

// AddExtmark registers a mark at off and returns its id.
func (e *EditBuffer) AddExtmark(off int, biasRight bool) int {
	id := e.nextMark
	e.nextMark++
	e.marks = append(e.marks, Extmark{ID: id, Offset: clampInt(off, 0, e.Text.Len()), BiasRight: biasRight})
	return id
}

// RemoveExtmark deletes a mark by id.
func (e *EditBuffer) RemoveExtmark(id int) {
	for i := range e.marks {
		if e.marks[i].ID == id {
			e.marks = append(e.marks[:i], e.marks[i+1:]...)
			return
		}
	}
}

// Extmark returns a mark by id.
func (e *EditBuffer) Extmark(id int) (Extmark, bool) {
	for _, m := range e.marks {
		if m.ID == id {
			return m, true
		}
	}
	return Extmark{}, false
}

// Extmarks returns all marks.
func (e *EditBuffer) Extmarks() []Extmark { return append([]Extmark(nil), e.marks...) }

func (e *EditBuffer) apply(start, end int, inserted string) {
	start, end = clampRange(start, end, e.Text.Len())
	deleted := e.Text.Slice(start, end)
	if deleted == inserted {
		return
	}
	op := EditOp{
		start:        start,
		inserted:     inserted,
		deleted:      deleted,
		cursorBefore: e.Cursor,
		cursorAfter:  start + len(inserted),
		anchorBefore: e.anchor,
		anchorAfter:  -1,
	}
	e.Text.Replace(start, end, inserted)
	e.shiftMarks(start, end, len(inserted))
	e.Cursor = op.cursorAfter
	e.anchor = -1
	e.goalCol = -1
	if e.MaxUndo > 0 && len(e.undo) >= e.MaxUndo {
		e.undo = append([]EditOp(nil), e.undo[len(e.undo)-e.MaxUndo+1:]...)
	}
	e.undo = append(e.undo, op)
	e.redo = nil
	e.changed()
}

func (e *EditBuffer) shiftMarks(start, end, inserted int) {
	removed := end - start
	for i := range e.marks {
		m := &e.marks[i]
		switch {
		case m.Offset > end || (m.Offset == end && m.BiasRight):
			m.Offset += inserted - removed
		case m.Offset > start || (m.Offset == start && m.BiasRight):
			m.Offset = start + inserted
		}
		if m.Offset < 0 {
			m.Offset = 0
		}
	}
}

func (e *EditBuffer) changed() {
	if e.OnChange != nil {
		e.OnChange()
	}
}
