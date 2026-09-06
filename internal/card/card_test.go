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
		Meta:    "06.09.26 · 13:47 · 7 commits",
		Groups: []Group{
			{Kind: KindAdded, Title: "Added", Items: []Item{
				{Text: "Add facial attendance", Hash: "a967103"},
				{Text: "Add kiosk interface", Hash: "92af81e"},
			}},
			{Kind: KindChanged, Title: "Changed", Items: []Item{
				{Text: "Refactor authentication flow", Hash: "03bc911"},
			}},
			{Kind: KindFixed, Title: "Fixed", Items: []Item{
				{Text: "Fix supervisor login", Hash: "c814ab2"},
			}},
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

func TestRenderDimensionsEveryShapeAndTheme(t *testing.T) {
	cases := map[Shape][2]int{
		Horizontal: {1200, 630},
		Vertical:   {1080, 1920},
		Square:     {1080, 1080},
	}
	for s, wh := range cases {
		for _, th := range ThemeNames {
			for _, hash := range []bool{false, true} {
				img := Render(sample(), s, Options{Theme: th, ShowHash: hash})
				b := img.Bounds()
				if b.Dx() != wh[0] || b.Dy() != wh[1] {
					t.Errorf("%v/%s hash=%v: %dx%d, want %dx%d", s, th, hash, b.Dx(), b.Dy(), wh[0], wh[1])
				}
			}
		}
	}
}

func TestRenderHandlesManyItemsWithoutOverflow(t *testing.T) {
	c := sample()
	for i := 0; i < 40; i++ {
		c.Groups[0].Items = append(c.Groups[0].Items, Item{
			Text: "A very long changelog line that must be truncated so it never runs past the card edge",
			Hash: "deadbee",
		})
	}
	if Render(c, Square, Options{ShowHash: true}).Bounds().Empty() {
		t.Fatal("empty image")
	}
}

func TestRenderNoGroups(t *testing.T) {
	c := sample()
	c.Groups = nil
	if Render(c, Horizontal, Options{}).Bounds().Empty() {
		t.Fatal("empty image")
	}
}

func TestSaveWritesValidPNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.png")
	if err := Save(Render(sample(), Horizontal, Options{}), path); err != nil {
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
