package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bijim/gotui/renderables"
	"github.com/bijim/gotui/renderer"
)

func TestEditorTabRenders(t *testing.T) {
	r := renderer.New(renderer.Options{Out: &bytes.Buffer{}, Width: 80, Height: 24})
	root := buildRoot(func() {})
	r.Root().Add(root)
	ts := root.BaseNode().Children()[1].(*renderables.TabSelect)
	ts.SetActive(3)
	r.RenderOnce()
	text := r.Buffer().Text()
	if !strings.Contains(text, "GoTUI editor") {
		t.Fatalf("editor content missing:\n%s", text)
	}
	if !strings.Contains(text, "1 GoTUI editor") {
		t.Fatalf("line numbers missing:\n%s", text)
	}
}
