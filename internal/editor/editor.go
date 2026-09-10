// Package editor opens the user's text editor on a scratch file. The --edit
// release flow uses it to let the generated notes be hand-edited before they
// are written to the changelog and the GitHub Release.
package editor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Edit opens text in the user's editor and returns the edited contents. The
// editor command is resolved from $RELIO_EDITOR, then $VISUAL, then $EDITOR,
// then a platform default ("vi", or "notepad" on Windows). The command string
// is split on whitespace so values like "code --wait" work. stdin/stdout/stderr
// are wired to the process's own, so the editor takes over the terminal.
func Edit(text string) (string, error) {
	f, err := os.CreateTemp("", "relio-notes-*.md")
	if err != nil {
		return "", err
	}
	path := f.Name()
	defer os.Remove(path)

	if _, err := f.WriteString(text); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}

	parts := strings.Fields(resolveEditor())
	cmd := exec.Command(parts[0], append(parts[1:], path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor %q failed: %w", strings.Join(parts, " "), err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

// resolveEditor returns the first non-empty of $RELIO_EDITOR, $VISUAL, and
// $EDITOR, falling back to a platform default.
func resolveEditor() string {
	for _, key := range []string{"RELIO_EDITOR", "VISUAL", "EDITOR"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}
