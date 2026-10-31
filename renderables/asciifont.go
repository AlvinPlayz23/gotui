package renderables

import (
	"strings"

	"github.com/bijim/gotui/color"
	"github.com/bijim/gotui/layout"
	"github.com/bijim/gotui/renderable"
)

// ASCIIFontOptions configures an ASCIIFont banner.
type ASCIIFontOptions struct {
	ID      string
	Text    string
	Font    map[rune][]string
	FG      color.RGBA
	BG      color.RGBA
	Scale   int
	Spacing int
	Style   layout.Style
}

// ASCIIFont renders text as a large block font.
type ASCIIFont struct {
	*renderable.Base
	Text    string
	Font    map[rune][]string
	FG      color.RGBA
	BG      color.RGBA
	Scale   int
	Spacing int
}

// NewASCIIFont creates an ASCII font banner.
func NewASCIIFont(opts ASCIIFontOptions) *ASCIIFont {
	if opts.Style == (layout.Style{}) {
		opts.Style = layout.DefaultStyle()
	}
	a := &ASCIIFont{
		Text:    opts.Text,
		Font:    opts.Font,
		FG:      opts.FG,
		BG:      opts.BG,
		Scale:   opts.Scale,
		Spacing: opts.Spacing,
	}
	if a.Font == nil {
		a.Font = FontBlocks
	}
	if a.Scale <= 0 {
		a.Scale = 1
	}
	a.Spacing = opts.Spacing
	if a.FG == (color.RGBA{}) {
		a.FG = White
	}
	a.Base = renderable.NewBase(a, &layout.Node{Style: opts.Style})
	a.Base.Node.Measure = a.measure
	return a
}

// SetText replaces the banner text.
func (a *ASCIIFont) SetText(s string) {
	a.Text = s
	a.Invalidate()
	a.RequestRender()
}

func (a *ASCIIFont) glyphHeight() int {
	return 5 * a.Scale
}

func (a *ASCIIFont) glyphWidth() (w int) {
	for _, rows := range a.blocks() {
		if lw := len(rows[0]); lw > w {
			w = lw
		}
	}
	return w * a.Scale
}

func (a *ASCIIFont) blocks() [][]string {
	var out [][]string
	for _, line := range strings.Split(a.Text, "\n") {
		for _, r := range strings.ToUpper(line) {
			rows, ok := a.Font[r]
			if !ok {
				rows = a.Font['?']
			}
			if rows == nil {
				rows = []string{"...", "...", "...", "...", "..."}
			}
			out = append(out, rows)
		}
	}
	return out
}

func (a *ASCIIFont) measure(availW, availH int) (int, int) {
	lines := strings.Split(a.Text, "\n")
	width := 0
	for _, line := range lines {
		w := 0
		for _, r := range strings.ToUpper(line) {
			rows, ok := a.Font[r]
			if !ok {
				rows = a.Font['?']
			}
			gw := 0
			if len(rows) > 0 {
				gw = len(rows[0])
			}
			w += gw*a.Scale + a.Spacing
		}
		if w > width {
			width = w
		}
	}
	if width > 0 {
		width -= a.Spacing
	}
	return maxInt(0, width), maxInt(0, len(lines)*a.glyphHeight())
}

// Paint draws the banner.
func (a *ASCIIFont) Paint(ctx *renderable.Context) {
	if !a.Visible {
		return
	}
	r := a.Bounds()
	if a.BG.A > 0 {
		ctx.Fill(r.X, r.Y, r.W, r.H, a.BG)
	}
	content := a.ContentBounds()
	if content.W <= 0 || content.H <= 0 {
		return
	}
	block := "█"
	if a.Scale > 1 {
		block = strings.Repeat("█", a.Scale)
	}
	y := content.Y
	for _, line := range strings.Split(a.Text, "\n") {
		x := content.X
		for _, r := range strings.ToUpper(line) {
			rows, ok := a.Font[r]
			if !ok {
				rows = a.Font['?']
			}
			if rows == nil {
				rows = []string{"...", "...", "...", "...", "..."}
			}
			gw := 0
			if len(rows) > 0 {
				gw = len(rows[0])
			}
			for gy := 0; gy < 5; gy++ {
				row := ""
				if gy < len(rows) {
					row = rows[gy]
				}
				for gx := 0; gx < gw; gx++ {
					if gx < len(row) && row[gx] == '#' {
						px := x + gx*a.Scale
						for sy := 0; sy < a.Scale; sy++ {
							py := y + gy*a.Scale + sy
							if py >= content.Bottom() {
								break
							}
							ctx.SetString(px, py, block, a.FG, a.BG, 0)
						}
					}
				}
			}
			x += gw*a.Scale + a.Spacing
			if x >= content.Right() {
				break
			}
		}
		y += a.glyphHeight()
		if y >= content.Bottom() {
			break
		}
	}
}

// FontBlocks is a built-in 3x5 block font supporting A-Z, 0-9, and common
// punctuation.
var FontBlocks = map[rune][]string{
	' ': {"...", "...", "...", "...", "..."},
	'A': {".#.", "#.#", "###", "#.#", "#.#"},
	'B': {"##.", "#.#", "##.", "#.#", "##."},
	'C': {".##", "#..", "#..", "#..", ".##"},
	'D': {"##.", "#.#", "#.#", "#.#", "##."},
	'E': {"###", "#..", "##.", "#..", "###"},
	'F': {"###", "#..", "##.", "#..", "#.."},
	'G': {".##", "#..", "#.#", "#.#", ".##"},
	'H': {"#.#", "#.#", "###", "#.#", "#.#"},
	'I': {"###", ".#.", ".#.", ".#.", "###"},
	'J': {"..#", "..#", "..#", "#.#", ".#."},
	'K': {"#.#", "#.#", "##.", "#.#", "#.#"},
	'L': {"#..", "#..", "#..", "#..", "###"},
	'M': {"#.#", "###", "###", "#.#", "#.#"},
	'N': {"#.#", "##.", "###", ".##", "#.#"},
	'O': {".#.", "#.#", "#.#", "#.#", ".#."},
	'P': {"##.", "#.#", "##.", "#..", "#.."},
	'Q': {".#.", "#.#", "#.#", "##.", ".##"},
	'R': {"##.", "#.#", "##.", "#.#", "#.#"},
	'S': {".##", "#..", ".#.", "..#", "##."},
	'T': {"###", ".#.", ".#.", ".#.", ".#."},
	'U': {"#.#", "#.#", "#.#", "#.#", "###"},
	'V': {"#.#", "#.#", "#.#", "#.#", ".#."},
	'W': {"#.#", "#.#", "###", "###", "#.#"},
	'X': {"#.#", "#.#", ".#.", "#.#", "#.#"},
	'Y': {"#.#", "#.#", ".#.", ".#.", ".#."},
	'Z': {"###", "..#", ".#.", "#..", "###"},
	'0': {"###", "#.#", "#.#", "#.#", "###"},
	'1': {".#.", "##.", ".#.", ".#.", "###"},
	'2': {"##.", "..#", ".#.", "#..", "###"},
	'3': {"##.", "..#", ".#.", "..#", "##."},
	'4': {"#.#", "#.#", "###", "..#", "..#"},
	'5': {"###", "#..", "##.", "..#", "##."},
	'6': {".##", "#..", "###", "#.#", "###"},
	'7': {"###", "..#", ".#.", ".#.", ".#."},
	'8': {"###", "#.#", "###", "#.#", "###"},
	'9': {"###", "#.#", "###", "..#", "##."},
	'!': {".#.", ".#.", ".#.", "...", ".#."},
	'?': {"##.", "..#", ".#.", "...", ".#."},
	'.': {"...", "...", "...", "...", ".#."},
	',': {"...", "...", "...", ".#.", "#.."},
	'-': {"...", "...", "###", "...", "..."},
	':': {"...", ".#.", "...", ".#.", "..."},
	'+': {"...", ".#.", "###", ".#.", "..."},
	'/': {"..#", "..#", ".#.", "#..", "#.."},
	'_': {"...", "...", "...", "...", "###"},
}
