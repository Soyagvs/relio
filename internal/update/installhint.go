package update

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/soyagvs/relio/internal/i18n"
)

// executablePath resolves the path to the currently running binary; a
// package-var seam (matching internal/tokenstore's keyringBackend pattern)
// so tests can inject a fake path instead of depending on the real
// filesystem and the real relio binary.
var executablePath = os.Executable

// InstallHint returns the command a user should run to upgrade, based on how
// the currently running binary was likely installed. It never downloads or
// replaces the binary itself -- self-replacement was explicitly rejected
// because relio ships via Homebrew, and swapping the binary out from under
// `brew upgrade` would fight the package manager.
func InstallHint() string {
	path, err := executablePath()
	if err != nil {
		return i18n.T(i18n.UpdateHintManual)
	}
	// Homebrew often puts a symlink on PATH (e.g. /usr/local/bin/relio)
	// pointing into the actual Cellar; resolve it so the Cellar check below
	// sees the real target. A resolution failure just keeps the original
	// path -- still worth checking.
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	if isHomebrewPath(path) {
		return i18n.T(i18n.UpdateHintBrew)
	}
	return i18n.T(i18n.UpdateHintManual)
}

// isHomebrewPath reports whether path looks like it's managed by Homebrew or
// Linuxbrew -- the standard heuristic: macOS Homebrew installs under
// /opt/homebrew/Cellar/<formula>/... or /usr/local/Cellar/<formula>/..., and
// Linuxbrew under /home/linuxbrew/.linuxbrew/Cellar/<formula>/....
func isHomebrewPath(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "cellar") ||
		strings.Contains(lower, "homebrew") ||
		strings.Contains(lower, "linuxbrew")
}
