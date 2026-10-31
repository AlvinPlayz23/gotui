package renderable

import "github.com/bijim/gotui/layout"

// Root is the top-level renderable owned by a renderer.
type Root struct {
	*Base
}

// NewRoot creates an empty root.
func NewRoot() *Root {
	r := &Root{}
	r.Base = NewBase(r, &layout.Node{Style: layout.DefaultStyle()})
	r.Base.Node.Style.Width = layout.Pct(100)
	r.Base.Node.Style.Height = layout.Pct(100)
	return r
}
