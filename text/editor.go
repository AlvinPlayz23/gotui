package text

// EditorView combines an EditBuffer with a TextView and keeps the cursor in
// view. It is the geometry half of an editor widget; key bindings live in the
// widget.
type EditorView struct {
	Edit *EditBuffer
	View *TextView
	// ScrollMargin is the number of lines kept around the cursor.
	ScrollMargin int
}

// NewEditorView creates a view over buf.
func NewEditorView(buf *EditBuffer, width, height int) *EditorView {
	ev := &EditorView{Edit: buf, View: NewTextView(buf.Text), ScrollMargin: 0}
	ev.View.Resize(width, height)
	return ev
}

// Resize sets the viewport size.
func (e *EditorView) Resize(w, h int) {
	e.View.Resize(w, h)
	e.EnsureCursorVisible()
}

// EnsureCursorVisible scrolls the view so the cursor is visible.
func (e *EditorView) EnsureCursorVisible() {
	line, col := e.Edit.LineCol()
	e.View.EnsureVisible(line, col, e.ScrollMargin)
}

// CursorPosition returns the cursor's viewport cell and whether it is inside
// the viewport.
func (e *EditorView) CursorPosition() (x, y int, visible bool) {
	line, col := e.Edit.LineCol()
	idx, cellX := e.View.VisualIndex(line, col)
	x = cellX - e.View.ScrollX
	y = idx - e.View.ScrollY
	visible = x >= 0 && x < e.View.Width && y >= 0 && y < e.View.Height
	return x, y, visible
}

// Invalidate recomputes the line layout.
func (e *EditorView) Invalidate() {
	e.View.Invalidate()
	e.EnsureCursorVisible()
}

// SelectionCells returns the selection rectangles for painting.
func (e *EditorView) SelectionCells() []SelectionCell {
	start, end, ok := e.Edit.Selection()
	if !ok {
		return nil
	}
	return e.View.SelectionCells(start, end)
}
