package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/i18n"
)

func TestQRBlockRendersAndIndents(t *testing.T) {
	out := qrBlock("https://0x0.st/abcd.png")
	if out == "" {
		t.Fatal("empty QR")
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 15 {
		t.Errorf("QR only %d lines, expected a full matrix", len(lines))
	}
	for i, ln := range lines {
		if !strings.HasPrefix(ln, "  ") {
			t.Errorf("line %d not indented: %q", i, ln)
		}
		if !strings.Contains(ln, "\x1b[4") {
			t.Errorf("line %d has no ANSI cell: %q", i, ln)
		}
	}
}

func TestImageDest(t *testing.T) {
	cases := []struct {
		name            string
		im              imageFlags
		interactive     bool
		save, link, ask bool
	}{
		{"no flags, no tty -> save only", imageFlags{}, false, true, false, false},
		{"no flags, tty -> ask", imageFlags{}, true, false, false, true},
		{"--upload -> save + link", imageFlags{upload: true}, true, true, true, false},
		{"--upload, no tty -> save + link", imageFlags{upload: true}, false, true, true, false},
		{"--link-only -> link only", imageFlags{linkOnly: true}, true, false, true, false},
		{"--link-only beats --upload", imageFlags{upload: true, linkOnly: true}, true, false, true, false},
	}
	for _, tc := range cases {
		s, l, a := imageDest(tc.im, tc.interactive)
		if s != tc.save || l != tc.link || a != tc.ask {
			t.Errorf("%s: got save=%v link=%v ask=%v, want %v/%v/%v",
				tc.name, s, l, a, tc.save, tc.link, tc.ask)
		}
	}
}

func TestDestFromAnswer(t *testing.T) {
	for _, tc := range []struct {
		ans        string
		save, link bool
	}{
		{"both", true, true},
		{"save", true, false},
		{"link", false, true},
	} {
		if s, l := destFromAnswer(tc.ans); s != tc.save || l != tc.link {
			t.Errorf("%q: got save=%v link=%v, want %v/%v", tc.ans, s, l, tc.save, tc.link)
		}
	}
}

func TestImageMeta(t *testing.T) {
	if got := imageMeta("2026-09-06 13:47", "2026-09-06", 4); got != "06.09.26 · 13:47 · 4 commits" {
		t.Errorf("imageMeta full = %q", got)
	}
	if got := imageMeta("", "2026-09-06", 2); got != "2026-09-06 · 2 commits" {
		t.Errorf("imageMeta date-only = %q", got)
	}
}

// TestNewImageCmdShortLongLocalizeAtConstructionTime proves newImageCmd()'s
// Short/Long and the --version/--hash/--upload/--link-only flag usage
// strings resolve through the active i18n catalog at construction time.
// --shape/--theme list the literal values the user types and stay
// untranslated on purpose, matching cmd/post.go's --format convention.
func TestNewImageCmdShortLongLocalizeAtConstructionTime(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	cmdEN := newImageCmd(&releaseFlags{})
	i18n.SetLanguage("es")
	cmdES := newImageCmd(&releaseFlags{})

	if cmdES.Short == cmdEN.Short {
		t.Errorf("image Short unchanged across languages: %q", cmdES.Short)
	}
	if cmdES.Long == cmdEN.Long {
		t.Error("image Long unchanged across languages")
	}
	for _, name := range []string{"version", "hash", "upload", "link-only"} {
		fEN := cmdEN.Flags().Lookup(name)
		fES := cmdES.Flags().Lookup(name)
		if fEN == nil || fES == nil {
			t.Fatalf("flag %q not found", name)
		}
		if fEN.Usage == fES.Usage {
			t.Errorf("--%s usage unchanged across languages: %q", name, fES.Usage)
		}
	}
	for _, name := range []string{"shape", "theme"} {
		fEN := cmdEN.Flags().Lookup(name)
		fES := cmdES.Flags().Lookup(name)
		if fEN == nil || fES == nil {
			t.Fatalf("flag %q not found", name)
		}
		if fEN.Usage != fES.Usage {
			t.Errorf("--%s usage should stay untranslated (literal values): en=%q es=%q", name, fEN.Usage, fES.Usage)
		}
	}
}

// TestGoldenEnglishImageCmdUnchanged pins the hardcoded English literals in
// cmd/image.go against i18n.T() under the default "en" language: converting
// them to i18n.T() calls MUST NOT change a single byte of English output.
func TestGoldenEnglishImageCmdUnchanged(t *testing.T) {
	prev := i18n.Current()
	if prev != "en" {
		i18n.SetLanguage("en")
	}
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	c := newImageCmd(&releaseFlags{})
	if c.Short != "Make a shareable image of a release" {
		t.Errorf("Short = %q", c.Short)
	}
	wantLong := "Render a dark, developer-styled release card. Everything on it comes\n" +
		"from the real release.\n\n" +
		"In a terminal, Relio asks what to do with it: save it to the current\n" +
		"directory, upload it for a QR + link, or both. With no TTY and no flags\n" +
		"it just saves to the current directory.\n\n" +
		"Flags skip the prompts:\n" +
		"  --version    release tag            --shape   horizontal|vertical|square\n" +
		"  --theme      orange|green|purple    --hash    show commit hashes\n" +
		"  --upload     also upload to a temp host (litterbox, 72h): link + QR\n" +
		"  --link-only  upload only — do not write a file to disk"
	if c.Long != wantLong {
		t.Errorf("Long = %q, want %q", c.Long, wantLong)
	}

	wantFlagUsage := map[string]string{
		"version":   "release tag to render (default: latest)",
		"shape":     "horizontal | vertical | square (default: horizontal)",
		"theme":     "orange | green | purple (default: orange)",
		"hash":      "show the commit hash on each line",
		"upload":    "also upload to a temp host (litterbox 72h) and show a link + QR",
		"link-only": "upload for a link + QR without writing a local file",
	}
	for name, want := range wantFlagUsage {
		f := c.Flags().Lookup(name)
		if f == nil || f.Usage != want {
			t.Errorf("--%s usage = %q, want %q", name, f.Usage, want)
		}
	}
}

// TestShapeItemsLocalizeLabelsAtConstructionTime proves shapeItems()'s
// Label/Desc resolve through the active catalog while Value stays the
// literal string card.ParseShape expects.
func TestShapeItemsLocalizeLabelsAtConstructionTime(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	itemsEN := shapeItems()
	i18n.SetLanguage("es")
	itemsES := shapeItems()

	if len(itemsEN) != len(itemsES) || len(itemsEN) != 3 {
		t.Fatalf("shapeItems() length mismatch: en=%d es=%d", len(itemsEN), len(itemsES))
	}
	for i := range itemsEN {
		if itemsEN[i].Value != itemsES[i].Value {
			t.Errorf("shapeItems()[%d].Value diverged across languages: en=%q es=%q", i, itemsEN[i].Value, itemsES[i].Value)
		}
		if itemsEN[i].Label == itemsES[i].Label && itemsEN[i].Desc == itemsES[i].Desc {
			t.Errorf("shapeItems()[%d] Label/Desc unchanged across languages: %q / %q", i, itemsEN[i].Label, itemsEN[i].Desc)
		}
	}
}

// TestThemeItemsLocalizeLabelsAtConstructionTime mirrors
// TestShapeItemsLocalizeLabelsAtConstructionTime for themeItems().
func TestThemeItemsLocalizeLabelsAtConstructionTime(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	itemsEN := themeItems()
	i18n.SetLanguage("es")
	itemsES := themeItems()

	if len(itemsEN) != len(itemsES) || len(itemsEN) != 3 {
		t.Fatalf("themeItems() length mismatch: en=%d es=%d", len(itemsEN), len(itemsES))
	}
	for i := range itemsEN {
		if itemsEN[i].Value != itemsES[i].Value {
			t.Errorf("themeItems()[%d].Value diverged across languages: en=%q es=%q", i, itemsEN[i].Value, itemsES[i].Value)
		}
		if itemsEN[i].Label == itemsES[i].Label && itemsEN[i].Desc == itemsES[i].Desc {
			t.Errorf("themeItems()[%d] Label/Desc unchanged across languages: %q / %q", i, itemsEN[i].Label, itemsEN[i].Desc)
		}
	}
}

// TestRunReleaseImageNoReleasesLocalizesOutput proves the no-releases-yet
// notice resolves through the active catalog.
func TestRunReleaseImageNoReleasesLocalizesOutput(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	dir, r := newStatusRepo(t)
	statusCommit(t, dir, "chore: init") // no tag: len(tags) == 0

	run := func(lang string) string {
		i18n.SetLanguage(lang)
		var buf bytes.Buffer
		c := &cobra.Command{}
		c.SetOut(&buf)
		if err := runReleaseImage(c, r, config.Default("proj"), imageFlags{}); err != nil {
			t.Fatalf("runReleaseImage: %v", err)
		}
		return buf.String()
	}

	outEN := run("en")
	outES := run("es")
	if outEN == outES {
		t.Error("no-releases-yet output unchanged across languages")
	}
	if !strings.Contains(outEN, "No releases yet") {
		t.Errorf("en output = %q, want it to mention 'No releases yet'", outEN)
	}
	if !strings.Contains(outES, "Aún no hay lanzamientos") {
		t.Errorf("es output = %q, want it to mention 'Aún no hay lanzamientos'", outES)
	}
}

// TestRunReleaseImageUnknownVersionLocalizesError proves the no-such-release
// error resolves through the active catalog and keeps the version argument.
func TestRunReleaseImageUnknownVersionLocalizesError(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	r, _ := repoWithPendingRelease(t) // tagged v1.5.0

	run := func(lang string) error {
		i18n.SetLanguage(lang)
		var buf bytes.Buffer
		c := &cobra.Command{}
		c.SetOut(&buf)
		return runReleaseImage(c, r, config.Default("proj"), imageFlags{version: "v9.9.9"})
	}

	errEN := run("en")
	errES := run("es")
	if errEN == nil || errES == nil {
		t.Fatalf("expected errors, got en=%v es=%v", errEN, errES)
	}
	if errEN.Error() == errES.Error() {
		t.Error("no-such-release error unchanged across languages")
	}
	if !strings.Contains(errEN.Error(), "v9.9.9") || !strings.Contains(errES.Error(), "v9.9.9") {
		t.Errorf("no-such-release error lost the version arg: en=%q es=%q", errEN.Error(), errES.Error())
	}
}
