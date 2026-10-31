package main

import (
	"bytes"
	"testing"

	"github.com/bijim/gotui/renderables"
	"github.com/bijim/gotui/renderer"
)

func BenchmarkRenderList(b *testing.B) {
	r := renderer.New(renderer.Options{Out: &bytes.Buffer{}, Width: 82, Height: 56})
	root := buildRoot(func() {})
	r.Root().Add(root)
	root.BaseNode().Children()[1].(*renderables.TabSelect).SetActive(1)
	r.RenderOnce()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.RenderOnce()
	}
}

func BenchmarkRenderForm(b *testing.B) {
	r := renderer.New(renderer.Options{Out: &bytes.Buffer{}, Width: 82, Height: 56})
	root := buildRoot(func() {})
	r.Root().Add(root)
	r.RenderOnce()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.RenderOnce()
	}
}
