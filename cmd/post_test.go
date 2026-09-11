package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/soyagvs/relio/internal/changelog"
	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/conventional"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/semver"
	"github.com/soyagvs/relio/internal/ui"
)

func samplePlan() release.Plan {
	commits := []conventional.Commit{
		{Type: "feat", Description: "add transaction categories", Hash: "1111111111111111111111111111111111111111"},
		{Type: "feat", Scope: "dashboard", Description: "monthly summary widget", Hash: "2222222222222222222222222222222222222222"},
		{Type: "fix", Description: "crash on empty account list", Hash: "3333333333333333333333333333333333333333"},
		{Type: "fix", Scope: "kiosk", Description: "header alignment", Hash: "4444444444444444444444444444444444444444"},
		{Type: "refactor", Description: "auth flow", Hash: "5555555555555555555555555555555555555555"},
		{Type: "chore", Description: "bump deps", Hash: "6666666666666666666666666666666666666666"},
	}
	return release.Plan{
		Current: semver.Version{Major: 1, Minor: 3, Patch: 2, Prefix: "v"},
		Next:    semver.Version{Major: 1, Minor: 4, Patch: 0, Prefix: "v"},
		Bump:    semver.Minor,
		Commits: commits,
		Notes:   changelog.Build(commits),
		Now:     time.Date(2026, 9, 6, 14, 30, 0, 0, time.UTC),
	}
}

func TestMinimalPost(t *testing.T) {
	got := minimalPost("gestam-frontend", samplePlan())

	for _, w := range []string{
		"relio -- release",
		"gestam-frontend · v1.4.0",
		"2026-09-06 · 14:30 · 6 commits",
		"Added",
		"1111111  Add transaction categories",
		"Fixed",
		"3333333  Crash on empty account list",
		"4444444  kiosk: Header alignment",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("missing %q in:\n%s", w, got)
		}
	}
	if strings.Contains(got, "v1.3.2..") {
		t.Errorf("commit range should be gone:\n%s", got)
	}

	lines := strings.Split(got, "\n")
	if lines[0] != "relio -- release" {
		t.Errorf("line 1 = %q", lines[0])
	}
	if lines[2] != "gestam-frontend · v1.4.0" {
		t.Errorf("line 3 = %q", lines[2])
	}
	if lines[3] != "2026-09-06 · 14:30 · 6 commits" {
		t.Errorf("line 4 = %q", lines[3])
	}
}

func TestSocialPost(t *testing.T) {
	got := socialPost("azeink", samplePlan())
	lines := strings.Split(got, "\n")

	if lines[0] != "azeink -- Release" && lines[0] != "Azeink -- Release" {
		t.Errorf("line 1 = %q", lines[0])
	}
	if lines[2] != "v1.4.0 · 06.09.26 · 14:30" {
		t.Errorf("meta line = %q", lines[2])
	}
	hasRow := func(typ, desc string) bool {
		for _, ln := range lines {
			if strings.HasPrefix(strings.TrimSpace(ln), typ) && strings.Contains(ln, desc) {
				return true
			}
		}
		return false
	}
	if !hasRow("feat", "Add transaction categories") || !hasRow("fix", "Crash on empty account list") {
		t.Errorf("expected feat/fix rows in:\n%s", got)
	}
	// chore is dropped, hashes never shown
	if strings.Contains(got, "bump deps") || strings.Contains(got, "1111111") {
		t.Errorf("social post should be filtered and hash-free:\n%s", got)
	}
}

func TestNoHashHidesHashes(t *testing.T) {
	ui.HideHashes = true
	defer func() { ui.HideHashes = false }()

	got := minimalPost("proj", samplePlan())
	if strings.Contains(got, "1111111") || strings.Contains(got, "3333333") {
		t.Errorf("--no-hash should drop hashes:\n%s", got)
	}
	if !strings.Contains(got, "• Add transaction categories") {
		t.Errorf("expected plain bullet lines:\n%s", got)
	}
}

// minimalPost must be byte-for-byte the releases-browser output for the same
// project, version, meta and notes.
func TestMinimalPostMatchesReleasesBrowser(t *testing.T) {
	p := samplePlan()
	got := minimalPost("gestam-frontend", p)
	want := ui.ReleaseText("gestam-frontend", p.Next.String(), commitMeta(p), p.Notes)
	if got != want {
		t.Errorf("minimalPost diverged from ui.ReleaseText:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestMinimalPostNoNotableChanges(t *testing.T) {
	commits := []conventional.Commit{{Type: "chore", Description: "bump deps"}}
	p := release.Plan{
		Next:    semver.Version{Minor: 1, Prefix: "v"},
		Commits: commits,
		Notes:   changelog.Build(commits),
		Now:     time.Date(2026, 9, 6, 14, 30, 0, 0, time.UTC),
	}
	got := minimalPost("proj", p)
	if !strings.Contains(got, "1 commits") {
		t.Errorf("commit count missing:\n%s", got)
	}
	if !strings.Contains(got, "(no user-facing changes)") {
		t.Errorf("expected the no-changes note:\n%s", got)
	}
}

// --- i18n: construction-time Short/Long/flag usage + runtime output localization ---

func TestNewPostCmdShortLongLocalizeAtConstructionTime(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	cEN := newPostCmd(&releaseFlags{})
	shortEN, longEN := cEN.Short, cEN.Long
	formatUsageEN := cEN.Flags().Lookup("format").Usage

	i18n.SetLanguage("es")
	cES := newPostCmd(&releaseFlags{})
	shortES, longES := cES.Short, cES.Long
	formatUsageES := cES.Flags().Lookup("format").Usage

	if shortEN != "Generate copy-paste release text for social posts" {
		t.Errorf("newPostCmd().Short (en) = %q", shortEN)
	}
	if shortES == shortEN || shortES == "" {
		t.Errorf("newPostCmd().Short unchanged across languages: %q", shortES)
	}
	if longES == longEN || longES == "" {
		t.Errorf("newPostCmd().Long unchanged across languages: %q", longES)
	}
	// --format's usage lists literal values the user types verbatim
	// (minimal|social|technical|casual|changelog); it stays untranslated
	// on purpose, matching how command/flag syntax stays literal elsewhere
	// (cmd/help.go's row names, PR6a).
	if formatUsageES != formatUsageEN {
		t.Errorf("--format usage should stay literal across languages: en=%q es=%q", formatUsageEN, formatUsageES)
	}
}

func TestPostFormatItemsLocalizeLabelsAtConstructionTime(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	itemsEN := postFormatItems()

	i18n.SetLanguage("es")
	itemsES := postFormatItems()

	if len(itemsEN) != len(itemsES) {
		t.Fatalf("postFormatItems length mismatch: en=%d es=%d", len(itemsEN), len(itemsES))
	}
	if itemsEN[0].Label != "Minimal" {
		t.Errorf("postFormatItems()[0].Label (en) = %q", itemsEN[0].Label)
	}
	for i := range itemsEN {
		if itemsEN[i].Value != itemsES[i].Value {
			t.Errorf("item %d Value changed across languages: %q vs %q", i, itemsEN[i].Value, itemsES[i].Value)
		}
		if itemsEN[i].Label == itemsES[i].Label {
			t.Errorf("item %d Label unchanged across languages: %q", i, itemsEN[i].Label)
		}
	}
}

func TestRenderPostUnknownFormatLocalizesError(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	_, errEN := renderPost("proj", samplePlan(), "bogus")
	i18n.SetLanguage("es")
	_, errES := renderPost("proj", samplePlan(), "bogus")

	if errEN == nil || errES == nil {
		t.Fatal("expected unknown-format errors in both languages")
	}
	if !strings.Contains(errEN.Error(), `unknown format "bogus"`) {
		t.Errorf("english golden error text missing: %q", errEN.Error())
	}
	if errEN.Error() == errES.Error() {
		t.Errorf("unknown-format error unchanged across languages: %q", errEN.Error())
	}
}

func TestSocialPostLocalizesOutput(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	gotEN := socialPost("azeink", samplePlan())
	if !strings.Contains(gotEN, "Azeink -- Release") && !strings.Contains(gotEN, "azeink -- Release") {
		t.Errorf("english golden header missing:\n%s", gotEN)
	}

	i18n.SetLanguage("es")
	gotES := socialPost("azeink", samplePlan())

	if gotEN == gotES {
		t.Error("socialPost output unchanged across languages")
	}
}

func TestSocialPostNoNotableChangesLocalizesOutput(t *testing.T) {
	p := release.Plan{Next: samplePlan().Next, Commits: nil}
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	gotEN := socialPost("proj", p)
	if !strings.Contains(gotEN, "(no notable changes)") {
		t.Errorf("english golden text missing:\n%s", gotEN)
	}

	i18n.SetLanguage("es")
	gotES := socialPost("proj", p)

	if gotEN == gotES {
		t.Error("socialPost no-notable-changes output unchanged across languages")
	}
}

func TestCasualPostLocalizesOutput(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	gotEN := casualPost("proj", samplePlan())
	if !strings.Contains(gotEN, "is out.") {
		t.Errorf("english golden text missing:\n%s", gotEN)
	}

	i18n.SetLanguage("es")
	gotES := casualPost("proj", samplePlan())

	if gotEN == gotES {
		t.Error("casualPost output unchanged across languages")
	}
}

func TestTechnicalPostLocalizesOutput(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	gotEN := technicalPost("proj", samplePlan())
	if !strings.Contains(gotEN, "6 commits · v1.4.0") {
		t.Errorf("english golden text missing:\n%s", gotEN)
	}

	i18n.SetLanguage("es")
	gotES := technicalPost("proj", samplePlan())

	if gotEN == gotES {
		t.Error("technicalPost output unchanged across languages")
	}
}

func postRepoWithTag(t *testing.T) string {
	t.Helper()
	dir, _ := newStatusRepo(t)
	statusCommit(t, dir, "chore: init")
	statusTag(t, dir, "v1.0.0")
	if err := config.Default("proj").Save(dir); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRunPostNoCommitsLocalizesOutput(t *testing.T) {
	dir := postRepoWithTag(t)
	f := &releaseFlags{dir: dir}

	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	cEN := newPostCmd(f)
	var errBufEN bytes.Buffer
	cEN.SetErr(&errBufEN)
	if err := cEN.RunE(cEN, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errBufEN.String(), "No commits since the last tag — nothing to announce.") {
		t.Errorf("english golden text missing:\n%s", errBufEN.String())
	}

	i18n.SetLanguage("es")
	cES := newPostCmd(f)
	var errBufES bytes.Buffer
	cES.SetErr(&errBufES)
	if err := cES.RunE(cES, nil); err != nil {
		t.Fatal(err)
	}

	if errBufEN.String() == errBufES.String() {
		t.Error("post no-commits output unchanged across languages")
	}
}
