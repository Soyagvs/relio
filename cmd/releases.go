package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/changelog"
	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/editor"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/ui"
)

// newReleasesCmd builds `relio releases`, a group for working with releases
// that have already been published.
func newReleasesCmd(f *releaseFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "releases",
		Short: i18n.T(i18n.ReleasesShort),
	}
	cmd.AddCommand(newReleasesEditCmd(f))
	return cmd
}

// newReleasesEditCmd builds `relio releases edit <version>`, which opens an
// already-published CHANGELOG.md section in the user's editor so a typo (or
// anything else) can be fixed without reinventing the changelog's
// extraction/removal logic.
func newReleasesEditCmd(f *releaseFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <version>",
		Short: i18n.T(i18n.ReleasesEditShort),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, cfg, err := openRepoAndConfig(f.dir)
			if err != nil {
				return err
			}
			return runReleasesEdit(cmd, repo, cfg, args[0])
		},
	}
	return cmd
}

// runReleasesEdit reads the version's changelog section, opens it in the
// user's editor, and — when it was actually changed — writes the edited
// section back in place. It touches only the changelog file: no commit, no
// tag.
func runReleasesEdit(cmd *cobra.Command, repo *gitrepo.Repo, cfg config.Config, version string) error {
	out := cmd.OutOrStdout()

	path := filepath.Join(repo.Root(), cfg.Release.ChangelogFile)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(i18n.T(i18n.ReleasesEditNoChangelog), cfg.Release.ChangelogFile)
		}
		return err
	}

	section := changelog.ExtractSection(string(content), version)
	if section == "" {
		return fmt.Errorf(i18n.T(i18n.ReleasesEditNoSuchVersion), version)
	}

	edited, err := editor.Edit(section)
	if err != nil {
		return err
	}

	switch {
	case strings.TrimSpace(edited) == strings.TrimSpace(section):
		_, err := fmt.Fprintln(out, ui.Info(i18n.T(i18n.ReleasesEditUnchanged)))
		return err
	case strings.TrimSpace(edited) == "":
		return errors.New(i18n.T(i18n.ReleasesEditBlankRefused))
	}

	newContent, _ := changelog.ReplaceSection(string(content), version, edited)
	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(out, ui.Success([]string{i18n.T(i18n.ReleasesEditDone, cfg.Release.ChangelogFile)})); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, ui.Dim.Render(i18n.T(i18n.ReleasesEditCommitHint, cfg.Release.ChangelogFile)))
	return err
}
