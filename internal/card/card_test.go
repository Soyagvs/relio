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

func TestParseDesignAndThemes(t *testing.T) {
	for in, want := range map[string]Design{"default": Default, "classic": Default, "minimal": Minimal, "bold": Bold, "neon": Neon} {
		got, ok := ParseDesign(in)
		if !ok || got != want {
			t.Errorf("ParseDesign(%q) = %v,%v want %v", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "loud"} {
		if _, ok := ParseDesign(in); ok {
			t.Errorf("ParseDesign(%q) ok = true, want false", in)
		}
	}
	if !ThemeForDesign(Default, "orange") || ThemeForDesign(Minimal, "orange") {
		t.Error("themes should be design-specific")
	}
}

func TestRenderDimensionsRepresentativeDesignShapeAndTheme(t *testing.T) {
	cases := []struct {
		shape  Shape
		design Design
		theme  string
		hash   bool
		w, h   int
	}{
		{Horizontal, Default, "orange", false, 1200, 630},
		{Horizontal, Minimal, "slate", false, 1200, 630},
		{Horizontal, Minimal, "sand", false, 1200, 630},
		{Horizontal, Minimal, "mint", false, 1200, 630},
		{Horizontal, Bold, "red", false, 1200, 630},
		{Horizontal, Bold, "blue", false, 1200, 630},
		{Horizontal, Bold, "gold", false, 1200, 630},
		{Horizontal, Neon, "cyan", false, 1200, 630},
		{Horizontal, Neon, "magenta", false, 1200, 630},
		{Horizontal, Neon, "lime", false, 1200, 630},
		{Vertical, Default, "green", false, 1080, 1920},
		{Square, Default, "purple", true, 1080, 1080},
	}
	for _, c := range cases {
		img := Render(sample(), c.shape, Options{Design: c.design, Theme: c.theme, ShowHash: c.hash})
		b := img.Bounds()
		if b.Dx() != c.w || b.Dy() != c.h {
			t.Errorf("%v/%v/%s hash=%v: %dx%d, want %dx%d", c.design, c.shape, c.theme, c.hash, b.Dx(), b.Dy(), c.w, c.h)
		}
	}
}

func TestRenderSupersamplesThenDownscales(t *testing.T) {
	if ssaa < 2 {
		t.Fatalf("ssaa = %d: supersampling is off, the card will render soft", ssaa)
	}
	// The card is drawn big internally but the returned image must still be the
	// exact target size — the downscale is what makes it crisp, not larger.
	img := Render(sample(), Horizontal, Options{})
	if b := img.Bounds(); b.Dx() != 1200 || b.Dy() != 630 {
		t.Fatalf("got %dx%d, want 1200x630 after downscale", b.Dx(), b.Dy())
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
