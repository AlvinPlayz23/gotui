package main

import (
	"bytes"
	"testing"

	"github.com/bijim/gotui/renderables"
	"github.com/bijim/gotui/renderer"
)

func TestCrashSizes(t *testing.T) {
	for _, sz := range [][2]int{{82, 56}, {120, 40}, {40, 12}, {10, 3}, {1, 1}} {
		r := renderer.New(renderer.Options{Out: &bytes.Buffer{}, Width: sz[0], Height: sz[1]})
		root := buildRoot(func() {})
		r.Root().Add(root)
		ts := root.BaseNode().Children()[1].(*renderables.TabSelect)
		for tab := 0; tab < len(ts.Items); tab++ {
			ts.SetActive(tab)
			for i := 0; i < 50; i++ {
				r.RenderOnce()
			}
		}
		r.Resize(sz[0]+1, sz[1]+1)
		r.RenderOnce()
	}
}
