package main

import (
	"bytes"
	"testing"

	"github.com/bijim/gotui/renderer"
)

func TestDemoLayout(t *testing.T) {
	r := renderer.New(renderer.Options{Out: &bytes.Buffer{}, Width: 80, Height: 24})
	r.Root().Add(buildRoot(func() {}))
	r.RenderOnce()
	t.Log("\n" + r.Buffer().Text())
}
