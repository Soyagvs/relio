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
			{Type: "feat", Text: "Add facial attendance", Hash: "a967103"},
			{Type: "feat", Text: "Add new kiosk interface", Hash: "92af81e"},
			{Type: "refactor", Text: "Authentication flow", Hash: "03bc911"},
			{Type: "fix", Text: "Supervisor login", Hash: "c814ab2"},
		},
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
		for _, hash := range []bool{false, true} {
			img := Render(sample(), s, hash)
			b := img.Bounds()
			if b.Dx() != wh[0] || b.Dy() != wh[1] {
				t.Errorf("%v hash=%v: got %dx%d, want %dx%d", s, hash, b.Dx(), b.Dy(), wh[0], wh[1])
			}
		}
	}
}

func TestRenderHandlesManyRowsAndLongText(t *testing.T) {
	c := sample()
	for i := 0; i < 40; i++ {
		c.Rows = append(c.Rows, Row{Type: "feat", Hash: "deadbee", Text: "A very long description that should be truncated with an ellipsis so it never overflows the card width"})
	}
	img := Render(c, Square, true) // must not panic
	if img.Bounds().Empty() {
		t.Fatal("empty image")
	}
}

func TestSaveWritesValidPNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.png")
	if err := Save(Render(sample(), Horizontal, false), path); err != nil {
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
