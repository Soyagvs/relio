package ui_test

import (
	"testing"
	"time"

	"github.com/soyagvs/relio/internal/changelog"
	"github.com/soyagvs/relio/internal/conventional"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/semver"
	"github.com/soyagvs/relio/internal/ui"
)

// fixturePlan is a fixed, self-contained release.Plan covering every
// changelog group, so every renderer under test has real content to render.
func fixturePlan() release.Plan {
	notes := changelog.Notes{Groups: map[changelog.Group][]changelog.Item{
		changelog.Added:      {{Text: "New export flag", Hash: "abc1234"}},
		changelog.Changed:    {{Text: "Faster startup", Hash: "def5678"}},
		changelog.Fixed:      {{Text: "Crash on empty list", Hash: "ghi9012"}},
		changelog.Removed:    {{Text: "Legacy config loader", Hash: "jkl3456"}},
		changelog.Deprecated: {{Text: "The --old-flag switch", Hash: "mno7890"}},
		changelog.Security:   {{Text: "Patched dependency X", Hash: "pqr1234"}},
	}}
	return release.Plan{
		Current: semver.Version{Major: 1, Minor: 5, Prefix: "v"},
		Next:    semver.Version{Major: 1, Minor: 6, Prefix: "v"},
		Bump:    semver.Minor,
		Notes:   notes,
		Now:     time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Commits: []conventional.Commit{{}, {}, {}},
		Footer:  "Thanks to @octocat for contributing!",
	}
}

// TestArtifactInvarianceAcrossLanguages proves the written-artifact boundary
// the spec requires: changelog and release-notes rendering must stay
// byte-identical across every registered language, while the interactive
// plan preview must actually change. The negative control (PlanBox/PlanView)
// is the real guard — without it, a silently no-op'ing SetLanguage would
// make every "invariant" assertion trivially pass while proving nothing.
func TestArtifactInvarianceAcrossLanguages(t *testing.T) {
	t.Cleanup(func() { i18n.SetLanguage("en") })

	langs := i18n.Languages()
	if len(langs) < 2 {
		t.Fatalf("language registry has fewer than 2 entries (%d); invariance proves nothing", len(langs))
	}

	fixture := fixturePlan()

	invariant := map[string]func() string{
		"changelog.RenderSection": func() string {
			return changelog.RenderSection(fixture.Next.String(), fixture.Now, fixture.Notes)
		},
		"release.Plan.Section":     func() string { return fixture.Section() },
		"release.Plan.ReleaseBody": func() string { return fixture.ReleaseBody() },
		"ui.Notes":                 func() string { return ui.Notes(fixture.Notes) },
		"ui.ReleaseText": func() string {
			return ui.ReleaseText("proj", fixture.Next.String(), "meta", fixture.Notes)
		},
	}
	localized := map[string]func() string{
		"ui.PlanBox":  func() string { return ui.PlanBox(fixture) },
		"ui.PlanView": func() string { return ui.PlanView(fixture) },
	}

	if _, ok := i18n.SetLanguage("en"); !ok {
		t.Fatal("SetLanguage(en) rejected the reference language")
	}
	baseline := map[string]string{}
	for name, render := range invariant {
		baseline[name] = render()
	}
	baseLocalized := map[string]string{}
	for name, render := range localized {
		baseLocalized[name] = render()
	}

	for _, lang := range langs {
		if _, ok := i18n.SetLanguage(lang.ID); !ok {
			t.Fatalf("SetLanguage(%q) rejected a registered language", lang.ID)
		}

		for name, render := range invariant {
			if got := render(); got != baseline[name] {
				t.Errorf("%s diverged under language %q — this surface must stay byte-identical across languages\n--- baseline (en) ---\n%s\n--- got (%s) ---\n%s",
					name, lang.ID, baseline[name], lang.ID, got)
			}
		}

		if lang.ID == "en" {
			continue
		}
		for name, render := range localized {
			if got := render(); got == baseLocalized[name] {
				t.Errorf("%s did NOT change under language %q — negative control failed: either SetLanguage silently no-op'd, or %s stopped routing through i18n.T",
					name, lang.ID, name)
			}
		}
	}
}
