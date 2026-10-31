package color

import "testing"

func TestParseHex(t *testing.T) {
	cases := map[string]RGBA{
		"#f00":      FromInts(255, 0, 0, 255),
		"#ff0000":   FromInts(255, 0, 0, 255),
		"#00ff0080": FromInts(0, 255, 0, 128),
		"#0f08":     FromInts(0, 255, 0, 136),
	}
	for in, want := range cases {
		got, err := Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", in, err)
		}
		if !got.Equal(want) {
			t.Errorf("Parse(%q) = %+v, want %+v", in, got, want)
		}
	}
}

func TestParseRGB(t *testing.T) {
	c, err := Parse("rgb(255, 128, 0)")
	if err != nil {
		t.Fatal(err)
	}
	if C255(c.R) != 255 || C255(c.G) != 128 || C255(c.B) != 0 || C255(c.A) != 255 {
		t.Fatalf("rgb = %+v", c)
	}
	c, err = Parse("rgba(255, 0, 0, 0.5)")
	if err != nil {
		t.Fatal(err)
	}
	if c.A != 0.5 {
		t.Fatalf("alpha = %v", c.A)
	}
	c, err = Parse("rgba(100%, 0%, 0%, 25%)")
	if err != nil {
		t.Fatal(err)
	}
	if c.R != 1 || c.A != 0.25 {
		t.Fatalf("percent = %+v", c)
	}
}

func TestParseNamedAndErrors(t *testing.T) {
	c, err := Parse("SteelBlue")
	if err != nil {
		t.Fatal(err)
	}
	if c != named["steelblue"] {
		t.Fatalf("named = %+v", c)
	}
	if _, err := Parse("nonsense"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := Parse("#12"); err == nil {
		t.Fatal("expected hex length error")
	}
	if _, err := Parse("rgb(1,2)"); err == nil {
		t.Fatal("expected arg error")
	}
}

func TestBlend(t *testing.T) {
	red := FromInts(255, 0, 0, 255)
	blue := FromInts(0, 0, 255, 255)
	half := red.WithAlpha(0.5)
	got := Blend(half, blue)
	if C255(got.R) != 128 || C255(got.B) != 128 {
		t.Fatalf("blend = %+v", got)
	}
	if got := Blend(red, blue); !got.Equal(red) {
		t.Fatalf("opaque blend = %+v", got)
	}
	if got := Blend(RGBA{}, blue); !got.Equal(blue) {
		t.Fatalf("transparent blend = %+v", got)
	}
}

func TestUint32RoundTrip(t *testing.T) {
	c := FromInts(12, 34, 56, 78)
	if got := FromUint32(c.Uint32()); !got.Equal(c) {
		t.Fatalf("round trip = %+v, want %+v", got, c)
	}
}
