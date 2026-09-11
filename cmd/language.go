package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/ui"
	"github.com/soyagvs/relio/internal/userconfig"
)

// prescanDir extracts the -C/--dir value from raw CLI args, mirroring the
// releaseFlags.dir flag. It runs before the cobra command tree exists:
// cobra evaluates Short/Long and flag usage strings at NewRootCmd()
// construction time, and --help returns before any PersistentPreRun hook
// fires, so the language must be resolved before that tree is built.
func prescanDir(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-C" || a == "--dir":
			if i+1 < len(args) {
				return args[i+1]
			}
			return "."
		case strings.HasPrefix(a, "--dir="):
			return strings.TrimPrefix(a, "--dir=")
		case strings.HasPrefix(a, "-C="):
			return strings.TrimPrefix(a, "-C=")
		}
	}
	return "."
}

// resolveLanguage applies the merge order default "en" < global user
// preference < project .release.yaml override, sets the active i18n
// language, and returns the id actually applied. A dir that is not a git
// repository (or has no .release.yaml) simply contributes no project-level
// override — Settings and language resolution both work outside a repo. An
// unknown language id anywhere in the chain warns once on w (hardcoded
// English: no language is resolved yet) and falls back to "en".
func resolveLanguage(dir string, w io.Writer) string {
	id := "en"
	if g := userconfig.Load().Language; g != "" {
		id = g
	}
	if repo, err := gitrepo.Open(dir); err == nil {
		if p := config.Language(repo.Root()); p != "" {
			id = p
		}
	}

	resolved, ok := i18n.SetLanguage(id)
	if !ok {
		fmt.Fprintln(w, ui.Warn.Render("✗ ")+fmt.Sprintf("unknown language %q, falling back to English", id))
	}
	return resolved
}
