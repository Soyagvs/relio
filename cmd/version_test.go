package cmd

import (
	"strings"
	"testing"

	"github.com/soyagvs/relio/internal/i18n"
)

// TestRenderUpdateNoticeIncludesVersionAndHint proves the TTY-only update
// notice printed by `relio version` carries both the "new version" line and
// the install-method-specific upgrade hint (internal/update.InstallHint),
// instead of the old hardcoded "brew upgrade relio" text that was wrong for
// non-Homebrew installs.
func TestRenderUpdateNoticeIncludesVersionAndHint(t *testing.T) {
	out := renderUpdateNotice("1.3.0", "Run `brew upgrade relio` to update.")

	if !strings.Contains(out, i18n.T(i18n.VersionUpdateAvailable, "1.3.0")) {
		t.Errorf("renderUpdateNotice output missing version line:\n%s", out)
	}
	if !strings.Contains(out, "Run `brew upgrade relio` to update.") {
		t.Errorf("renderUpdateNotice output missing hint line:\n%s", out)
	}
}

func TestRenderUpdateNoticeWithManualHint(t *testing.T) {
	hint := "See https://github.com/soyagvs/relio/blob/main/docs/install.md to update."
	out := renderUpdateNotice("2.0.0", hint)

	if !strings.Contains(out, hint) {
		t.Errorf("renderUpdateNotice output missing manual hint:\n%s", out)
	}
}
