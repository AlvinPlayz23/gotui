package layout

import "testing"

func rect(n *Node) Rect { return Rect{n.X, n.Y, n.W, n.H} }

func TestColumnBasic(t *testing.T) {
	root := &Node{Style: DefaultStyle()}
	root.Style.Width = P(20)
	root.Style.Height = P(10)
	root.Style.Padding = [4]Value{P(1), P(1), P(1), P(1)}
	a := &Node{Style: DefaultStyle()}
	a.Style.Height = P(2)
	b := &Node{Style: DefaultStyle()}
	b.Style.Height = P(3)
	root.Add(a, b)
	root.Compute(80, 24)
	if got := rect(root); got != (Rect{0, 0, 20, 10}) {
		t.Fatalf("root %+v", got)
	}
	if got := rect(a); got != (Rect{1, 1, 18, 2}) {
		t.Fatalf("a %+v", got)
	}
	if got := rect(b); got != (Rect{1, 3, 18, 3}) {
		t.Fatalf("b %+v", got)
	}
}

func TestRowGrow(t *testing.T) {
	root := &Node{Style: DefaultStyle()}
	root.Style.Direction = Row
	root.Style.Width = P(30)
	root.Style.Height = P(5)
	a := &Node{Style: DefaultStyle()}
	a.Style.Width = P(10)
	a.Style.FlexGrow = 1
	b := &Node{Style: DefaultStyle()}
	b.Style.Width = P(10)
	root.Add(a, b)
	root.Compute(80, 24)
	if a.W != 20 || b.W != 10 {
		t.Fatalf("a.W=%d b.W=%d", a.W, b.W)
	}
}

func TestJustifyCenter(t *testing.T) {
	root := &Node{Style: DefaultStyle()}
	root.Style.Direction = Row
	root.Style.Width = P(20)
	root.Style.Height = P(4)
	root.Style.Justify = JustifyCenter
	a := &Node{Style: DefaultStyle()}
	a.Style.Width = P(4)
	a.Style.Height = P(2)
	root.Add(a)
	root.Compute(80, 24)
	if a.X != 8 {
		t.Fatalf("a.X=%d", a.X)
	}
}

func TestPercent(t *testing.T) {
	root := &Node{Style: DefaultStyle()}
	root.Style.Width = P(40)
	root.Style.Height = P(10)
	a := &Node{Style: DefaultStyle()}
	a.Style.Width = Pct(50)
	a.Style.Height = Pct(50)
	root.Add(a)
	root.Compute(80, 24)
	if a.W != 20 || a.H != 5 {
		t.Fatalf("a %+v", rect(a))
	}
}

func TestAbsolute(t *testing.T) {
	root := &Node{Style: DefaultStyle()}
	root.Style.Width = P(20)
	root.Style.Height = P(10)
	a := &Node{Style: DefaultStyle()}
	a.Style.Position = Absolute
	a.Style.Top = P(2)
	a.Style.Left = P(3)
	a.Style.Width = P(5)
	a.Style.Height = P(2)
	root.Add(a)
	root.Compute(80, 24)
	if got := rect(a); got != (Rect{3, 2, 5, 2}) {
		t.Fatalf("a %+v", got)
	}
}
