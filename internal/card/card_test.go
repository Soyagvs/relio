package card

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func sample() Card {
	return Card{
		Project: "Azeink",
		Version: "v1.4.0",
		Meta:    "06.09.26 · 13:47 · 4 commits",
		Rows: []Row{
			{"feat", "Add facial attendance"},
			{"feat", "Add new kiosk interface"},
			{"refactor", "Authentication flow"},
			{"fix", "Supervisor login"},
		},
		Author: "SOYAGVS",
	}
}

func TestParseShape(t *testing.T) {
	for in, want := range map[string]Shape{
		"horizontal": Horizontal, "wide": Horizontal,
		"vertical": Vertical, "story": Vertical,
		"square": Square, "SQ": Square,
	} {
		got, ok := ParseShape(in)
		if !ok || got != want {
			t.Errorf("ParseShape(%q) = %v,%v want %v", in, got, ok, want)
		}
	}
	if _, ok := ParseShape("triangle"); ok {
		t.Error("expected invalid shape to report ok=false")
	}
}

func TestRenderDimensions(t *testing.T) {
	cases := map[Shape][2]int{
		Horizontal: {1200, 630},
		Vertical:   {1080, 1350},
		Square:     {1080, 1080},
	}
	for s, wh := range cases {
		img := Render(sample(), s)
		b := img.Bounds()
		if b.Dx() != wh[0] || b.Dy() != wh[1] {
			t.Errorf("%v: got %dx%d, want %dx%d", s, b.Dx(), b.Dy(), wh[0], wh[1])
		}
	}
}

func TestRenderHandlesManyRowsAndLongText(t *testing.T) {
	c := sample()
	for i := 0; i < 40; i++ {
		c.Rows = append(c.Rows, Row{"feat", "A very long description that should be truncated with an ellipsis so it never overflows the card width"})
	}
	img := Render(c, Square) // must not panic
	if img.Bounds().Empty() {
		t.Fatal("empty image")
	}
}

func TestSaveWritesValidPNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.png")
	if err := Save(Render(sample(), Horizontal), path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		t.Fatalf("not a valid PNG: %v", err)
	}
}
