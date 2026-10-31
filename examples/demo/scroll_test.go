package main

import (
	"bytes"
	"testing"

	"github.com/bijim/gotui/input"
	"github.com/bijim/gotui/renderables"
	"github.com/bijim/gotui/renderer"
)

func TestScrollFocus(t *testing.T) {
	r := renderer.New(renderer.Options{Out: &bytes.Buffer{}, Width: 80, Height: 24})
	root := buildRoot(func() {})
	r.Root().Add(root)
	ts := root.BaseNode().Children()[1].(*renderables.TabSelect)
	ts.SetActive(1)
	r.RenderOnce()
	r.HandleEvent(input.Event{Type: input.EventMouse, Mouse: input.MouseEvent{X: 50, Y: 10, Action: input.MousePress}})
	r.HandleEvent(input.Event{Type: input.EventMouse, Mouse: input.MouseEvent{X: 50, Y: 10, Action: input.MouseRelease}})
	t.Logf("focused %T", r.Focused())
	sb := ts.Items[1].Content.(*renderables.ScrollBox)
	r.HandleEvent(input.Event{Type: input.EventKey, Key: input.KeyEvent{Name: "pagedown"}})
	t.Logf("scrollY=%d max=%d", sb.ScrollY, sb.MaxScroll())
}
