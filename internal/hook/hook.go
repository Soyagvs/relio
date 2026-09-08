// Package hook runs the user's before/after release commands from .release.yaml
// through the platform shell, streaming their output and stopping at the first
// failure.
package hook

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

// Env is the set of RELIO_* variables exported to every hook command.
type Env struct {
	Version     string // "1.6.0"
	Tag         string // "v1.6.0"
	PreviousTag string // "v1.5.0", or ""
}

// Run executes each command in order through the platform shell (`sh -c` on
// Unix, `cmd /c` on Windows), streaming combined output to w. dir is the working
// directory. It stops at the first non-zero exit and returns an error naming the
// failed command and its exit status. An empty list is a no-op.
func Run(w io.Writer, dir string, commands []string, env Env) error {
	for _, c := range commands {
		fmt.Fprintf(w, "› %s\n", c)

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", c)
		} else {
			cmd = exec.Command("sh", "-c", c)
		}
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"RELIO_VERSION="+env.Version,
			"RELIO_TAG="+env.Tag,
			"RELIO_PREVIOUS_TAG="+env.PreviousTag,
		)
		cmd.Stdout = w
		cmd.Stderr = w

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("hook %q failed: %w", c, err)
		}
	}
	return nil
}
