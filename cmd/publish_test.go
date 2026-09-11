package cmd

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/release"
)

func TestReadYes(t *testing.T) {
	yes := []string{"y\n", "Y\n", "yes\n", "  yes  \n", "YES"}
	no := []string{"\n", "n\n", "no\n", "nope\n", "later", ""}

	for _, in := range yes {
		if !readYes(strings.NewReader(in)) {
			t.Errorf("readYes(%q) = false, want true", in)
		}
	}
	for _, in := range no {
		if readYes(strings.NewReader(in)) {
			t.Errorf("readYes(%q) = true, want false", in)
		}
	}
}

// --- i18n: publishGitHubRelease output/error localization ---

func TestPublishGitHubReleaseNoTokenLocalizesOutput(t *testing.T) {
	t.Setenv("RELIO_GITHUB_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("PATH", t.TempDir()) // no `gh` binary reachable, so ghAuthToken() fails fast

	applied := release.ApplyResult{TagName: "v1.2.0"}

	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	var bufEN bytes.Buffer
	printedEN, errEN := publishGitHubRelease(&bufEN, nil, config.Config{}, release.Plan{}, applied, true, true)
	if errEN != nil {
		t.Fatalf("publishGitHubRelease: %v", errEN)
	}
	if !printedEN {
		t.Error("expected printedNext = true on the no-token path")
	}
	if !strings.Contains(bufEN.String(), "Skipping GitHub publish: no token found.") {
		t.Errorf("english golden text missing:\n%s", bufEN.String())
	}

	i18n.SetLanguage("es")
	var bufES bytes.Buffer
	if _, err := publishGitHubRelease(&bufES, nil, config.Config{}, release.Plan{}, applied, true, true); err != nil {
		t.Fatalf("publishGitHubRelease: %v", err)
	}

	if bufEN.String() == bufES.String() {
		t.Error("publishGitHubRelease no-token output unchanged across languages")
	}
}

func TestPublishGitHubReleaseNoOriginRemoteLocalizesError(t *testing.T) {
	t.Setenv("RELIO_GITHUB_TOKEN", "dummy-token")

	dir, r := newStatusRepo(t)
	statusCommit(t, dir, "chore: init")
	applied := release.ApplyResult{TagName: "v1.0.0"}

	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	_, errEN := publishGitHubRelease(io.Discard, r, config.Config{}, release.Plan{}, applied, false, true)
	i18n.SetLanguage("es")
	_, errES := publishGitHubRelease(io.Discard, r, config.Config{}, release.Plan{}, applied, false, true)

	if errEN == nil || errES == nil {
		t.Fatal("expected no-origin-remote errors in both languages")
	}
	if !strings.Contains(errEN.Error(), "cannot publish: no `origin` remote") {
		t.Errorf("english golden error text missing: %q", errEN.Error())
	}
	if errEN.Error() == errES.Error() {
		t.Errorf("no-origin-remote error unchanged across languages: %q", errEN.Error())
	}
}
