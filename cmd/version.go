package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/ui"
	"github.com/soyagvs/relio/internal/update"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: i18n.T(i18n.VersionShort),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			if _, err := fmt.Fprintf(w, i18n.T(i18n.VersionInfoLine), ui.AppName, version, commit, date); err != nil {
				return err
			}

			// Only nudge on a real terminal, so scripts parsing `relio version`
			// keep getting a single clean line.
			if stdoutIsTTY() {
				if v := update.Available(version); v != "" {
					if _, err := fmt.Fprint(w, renderUpdateNotice(v, update.InstallHint())); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

// renderUpdateNotice renders the "new version available" line plus the
// install-method-specific upgrade hint (see internal/update.InstallHint).
// Kept as a pure function of its already-resolved inputs -- rather than
// calling update.Available/update.InstallHint itself -- so it's testable
// without a real TTY or a real running binary.
func renderUpdateNotice(latest, hint string) string {
	return ui.Key.Render(i18n.T(i18n.VersionUpdateAvailable, latest)) + "\n" +
		ui.Dim.Render(hint) + "\n"
}
