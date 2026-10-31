// Package color implements OpenTUI-compatible RGBA colors and parsing.
package color

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// RGBA is a linear-ish 0..1 color with alpha, matching OpenTUI's RGBA model.
type RGBA struct {
	R float32
	G float32
	B float32
	A float32
}

// New returns an RGBA from 0..1 components.
func New(r, g, b, a float32) RGBA { return RGBA{r, g, b, a} }

// FromInts returns a color from 0..255 components.
func FromInts(r, g, b, a uint8) RGBA {
	return RGBA{float32(r) / 255, float32(g) / 255, float32(b) / 255, float32(a) / 255}
}

// FromUint32 builds a color from packed 0xRRGGBBAA.
func FromUint32(v uint32) RGBA {
	return FromInts(uint8(v>>24), uint8(v>>16), uint8(v>>8), uint8(v))
}

// Uint32 packs the color as 0xRRGGBBAA.
func (c RGBA) Uint32() uint32 {
	return uint32(C255(c.R))<<24 | uint32(C255(c.G))<<16 | uint32(C255(c.B))<<8 | uint32(C255(c.A))
}

// C255 clamps a 0..1 float to an 8-bit channel.
func C255(v float32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(math.Round(float64(v) * 255))
}

// Equal reports whether two colors have equal components.
func (c RGBA) Equal(o RGBA) bool {
	return c == o
}

// IsTransparent reports whether the color is fully transparent.
func (c RGBA) IsTransparent() bool { return c.A <= 0 }

// WithAlpha returns c with a new alpha.
func (c RGBA) WithAlpha(a float32) RGBA { c.A = a; return c }

// Blend alpha-composites src over dst (src is on top).
func Blend(src, dst RGBA) RGBA {
	sa := clamp01(src.A)
	if sa >= 1 {
		return src
	}
	if sa <= 0 {
		return dst
	}
	da := clamp01(dst.A)
	outA := sa + da*(1-sa)
	if outA <= 0 {
		return RGBA{}
	}
	out := RGBA{A: outA}
	out.R = (src.R*sa + dst.R*da*(1-sa)) / outA
	out.G = (src.G*sa + dst.G*da*(1-sa)) / outA
	out.B = (src.B*sa + dst.B*da*(1-sa)) / outA
	return out
}

// Lerp linearly interpolates two colors.
func Lerp(a, b RGBA, t float32) RGBA {
	t = clamp01(t)
	return RGBA{
		R: a.R + (b.R-a.R)*t,
		G: a.G + (b.G-a.G)*t,
		B: a.B + (b.B-a.B)*t,
		A: a.A + (b.A-a.A)*t,
	}
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// Parse parses a color from "#rgb", "#rgba", "#rrggbb", "#rrggbbaa",
// "rgb(r,g,b)", "rgba(r,g,b,a)", or a CSS color name. Values in rgb() are
// 0..255; rgba() alpha is 0..1.
func Parse(s string) (RGBA, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return RGBA{}, fmt.Errorf("empty color")
	}
	lower := strings.ToLower(s)
	if lower == "transparent" {
		return RGBA{}, nil
	}
	if strings.HasPrefix(lower, "#") {
		return parseHex(lower[1:])
	}
	if strings.HasPrefix(lower, "rgb(") && strings.HasSuffix(lower, ")") {
		return parseRGBArgs(lower[4:len(lower)-1], false)
	}
	if strings.HasPrefix(lower, "rgba(") && strings.HasSuffix(lower, ")") {
		return parseRGBArgs(lower[5:len(lower)-1], true)
	}
	if c, ok := named[lower]; ok {
		return c, nil
	}
	return RGBA{}, fmt.Errorf("unknown color %q", s)
}

// MustParse is Parse that panics on error.
func MustParse(s string) RGBA {
	c, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return c
}

func parseHex(h string) (RGBA, error) {
	hexVal := func(b byte) (uint8, bool) {
		switch {
		case b >= '0' && b <= '9':
			return b - '0', true
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10, true
		}
		return 0, false
	}
	pair := func(a, b byte) (uint8, bool) {
		hi, ok1 := hexVal(a)
		lo, ok2 := hexVal(b)
		return hi<<4 | lo, ok1 && ok2
	}
	switch len(h) {
	case 3, 4:
		vals := make([]uint8, len(h))
		for i := 0; i < len(h); i++ {
			v, ok := hexVal(h[i])
			if !ok {
				return RGBA{}, fmt.Errorf("invalid hex color")
			}
			vals[i] = v<<4 | v
		}
		a := uint8(255)
		if len(vals) == 4 {
			a = vals[3]
		}
		return FromInts(vals[0], vals[1], vals[2], a), nil
	case 6, 8:
		var vals [4]uint8
		vals[3] = 255
		for i := 0; i < len(h)/2; i++ {
			v, ok := pair(h[i*2], h[i*2+1])
			if !ok {
				return RGBA{}, fmt.Errorf("invalid hex color")
			}
			vals[i] = v
		}
		return FromInts(vals[0], vals[1], vals[2], vals[3]), nil
	}
	return RGBA{}, fmt.Errorf("invalid hex color length")
}

func parseRGBArgs(s string, withAlpha bool) (RGBA, error) {
	parts := strings.Split(s, ",")
	if len(parts) < 3 || len(parts) > 4 {
		return RGBA{}, fmt.Errorf("invalid rgb() args")
	}
	ch := func(i int) (float32, error) {
		p := strings.TrimSpace(parts[i])
		if strings.HasSuffix(p, "%") {
			v, err := strconv.ParseFloat(strings.TrimSuffix(p, "%"), 64)
			if err != nil {
				return 0, err
			}
			return float32(v / 100), nil
		}
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return 0, err
		}
		return float32(v / 255), nil
	}
	r, err := ch(0)
	if err != nil {
		return RGBA{}, err
	}
	g, err := ch(1)
	if err != nil {
		return RGBA{}, err
	}
	b, err := ch(2)
	if err != nil {
		return RGBA{}, err
	}
	a := float32(1)
	if len(parts) == 4 {
		p := strings.TrimSpace(parts[3])
		if strings.HasSuffix(p, "%") {
			v, err := strconv.ParseFloat(strings.TrimSuffix(p, "%"), 64)
			if err != nil {
				return RGBA{}, err
			}
			a = float32(v / 100)
		} else {
			v, err := strconv.ParseFloat(p, 64)
			if err != nil {
				return RGBA{}, err
			}
			if !withAlpha && v > 1 {
				v = v / 255
			}
			a = float32(v)
		}
	}
	return RGBA{clamp01(r), clamp01(g), clamp01(b), clamp01(a)}, nil
}

var named = map[string]RGBA{
	"black":         FromInts(0, 0, 0, 255),
	"white":         FromInts(255, 255, 255, 255),
	"red":           FromInts(255, 0, 0, 255),
	"green":         FromInts(0, 128, 0, 255),
	"blue":          FromInts(0, 0, 255, 255),
	"yellow":        FromInts(255, 255, 0, 255),
	"cyan":          FromInts(0, 255, 255, 255),
	"magenta":       FromInts(255, 0, 255, 255),
	"gray":          FromInts(128, 128, 128, 255),
	"grey":          FromInts(128, 128, 128, 255),
	"orange":        FromInts(255, 165, 0, 255),
	"purple":        FromInts(128, 0, 128, 255),
	"pink":          FromInts(255, 192, 203, 255),
	"brown":         FromInts(165, 42, 42, 255),
	"lime":          FromInts(0, 255, 0, 255),
	"navy":          FromInts(0, 0, 128, 255),
	"teal":          FromInts(0, 128, 128, 255),
	"olive":         FromInts(128, 128, 0, 255),
	"maroon":        FromInts(128, 0, 0, 255),
	"silver":        FromInts(192, 192, 192, 255),
	"aqua":          FromInts(0, 255, 255, 255),
	"fuchsia":       FromInts(255, 0, 255, 255),
	"gold":          FromInts(255, 215, 0, 255),
	"violet":        FromInts(238, 130, 238, 255),
	"indigo":        FromInts(75, 0, 130, 255),
	"coral":         FromInts(255, 127, 80, 255),
	"salmon":        FromInts(250, 128, 114, 255),
	"tomato":        FromInts(255, 99, 71, 255),
	"khaki":         FromInts(240, 230, 140, 255),
	"beige":         FromInts(245, 245, 220, 255),
	"ivory":         FromInts(255, 255, 240, 255),
	"lavender":      FromInts(230, 230, 250, 255),
	"skyblue":       FromInts(135, 206, 235, 255),
	"steelblue":     FromInts(70, 130, 180, 255),
	"royalblue":     FromInts(65, 105, 225, 255),
	"slateblue":     FromInts(106, 90, 205, 255),
	"darkred":       FromInts(139, 0, 0, 255),
	"darkgreen":     FromInts(0, 100, 0, 255),
	"darkblue":      FromInts(0, 0, 139, 255),
	"darkgray":      FromInts(169, 169, 169, 255),
	"darkgrey":      FromInts(169, 169, 169, 255),
	"lightgray":     FromInts(211, 211, 211, 255),
	"lightgrey":     FromInts(211, 211, 211, 255),
	"whitesmoke":    FromInts(245, 245, 245, 255),
	"rebeccapurple": FromInts(102, 51, 153, 255),
}
