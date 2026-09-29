package update

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/soyagvs/relio/internal/i18n"
)

// withExecutablePath temporarily replaces the executablePath seam and
// restores it on test cleanup.
func withExecutablePath(t *testing.T, fn func() (string, error)) {
	t.Helper()
	orig := executablePath
	executablePath = fn
	t.Cleanup(func() { executablePath = orig })
}

func TestInstallHintDetectsHomebrewCellarPath(t *testing.T) {
	withExecutablePath(t, func() (string, error) {
		return "/opt/homebrew/Cellar/relio/1.3.0/bin/relio", nil
	})
	if got, want := InstallHint(), i18n.T(i18n.UpdateHintBrew); got != want {
		t.Errorf("InstallHint() = %q, want %q", got, want)
	}
}

func TestInstallHintDetectsLinuxbrewPath(t *testing.T) {
	withExecutablePath(t, func() (string, error) {
		return "/home/linuxbrew/.linuxbrew/Cellar/relio/1.3.0/bin/relio", nil
	})
	if got, want := InstallHint(), i18n.T(i18n.UpdateHintBrew); got != want {
		t.Errorf("InstallHint() = %q, want %q", got, want)
	}
}

func TestInstallHintDetectsGenericHomebrewSubstring(t *testing.T) {
	withExecutablePath(t, func() (string, error) {
		return "/usr/local/homebrew/bin/relio", nil
	})
	if got, want := InstallHint(), i18n.T(i18n.UpdateHintBrew); got != want {
		t.Errorf("InstallHint() = %q, want %q", got, want)
	}
}

func TestInstallHintManualInstall(t *testing.T) {
	withExecutablePath(t, func() (string, error) {
		return "/usr/local/bin/relio", nil
	})
	if got, want := InstallHint(), i18n.T(i18n.UpdateHintManual); got != want {
		t.Errorf("InstallHint() = %q, want %q", got, want)
	}
}

func TestInstallHintExecutablePathError(t *testing.T) {
	withExecutablePath(t, func() (string, error) {
		return "", errors.New("boom")
	})
	if got, want := InstallHint(), i18n.T(i18n.UpdateHintManual); got != want {
		t.Errorf("InstallHint() with executablePath error = %q, want fallback %q", got, want)
	}
}

// TestInstallHintResolvesSymlink proves the Homebrew check inspects the
// resolved target, not the symlink path itself -- matching how Homebrew
// actually lays binaries out (e.g. /usr/local/bin/relio is a symlink into
// the Cellar).
func TestInstallHintResolvesSymlink(t *testing.T) {
	dir := t.TempDir()
	cellarDir := filepath.Join(dir, "Cellar", "relio", "1.0.0", "bin")
	if err := os.MkdirAll(cellarDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	realBin := filepath.Join(cellarDir, "relio")
	if err := os.WriteFile(realBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	linkBin := filepath.Join(dir, "relio")
	if err := os.Symlink(realBin, linkBin); err != nil {
		t.Skipf("symlinks unsupported on this platform: %v", err)
	}

	withExecutablePath(t, func() (string, error) { return linkBin, nil })

	if got, want := InstallHint(), i18n.T(i18n.UpdateHintBrew); got != want {
		t.Errorf("InstallHint() via symlink = %q, want %q", got, want)
	}
}
