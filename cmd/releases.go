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
	"github.com/soyagvs/relio/internal/releases"
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
	cmd.AddCommand(newReleasesListCmd(f))
	cmd.AddCommand(newReleasesDeleteCmd(f))
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

// newReleasesListCmd builds `relio releases list`, a non-interactive dump of
// every local tag — the same data source the interactive browser
// (`internal/releases`, reachable from the main menu) uses, with no GitHub
// API call.
func newReleasesListCmd(f *releaseFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: i18n.T(i18n.ReleasesListShort),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, _, err := openRepoAndConfig(f.dir)
			if err != nil {
				return err
			}
			return runReleasesList(cmd, repo)
		},
	}
	return cmd
}

// runReleasesList prints every local tag, newest first, one per line.
func runReleasesList(cmd *cobra.Command, repo *gitrepo.Repo) error {
	out := cmd.OutOrStdout()

	tags, err := repo.Tags()
	if err != nil {
		return err
	}
	if len(tags) == 0 {
		_, err := fmt.Fprintln(out, ui.Info(i18n.T(i18n.ReleasesListEmpty)))
		return err
	}
	for _, t := range tags {
		if _, err := fmt.Fprintf(out, "%-12s  %s  %s\n", t.Name, t.Date, t.Subject); err != nil {
			return err
		}
	}
	return nil
}

// newReleasesDeleteCmd builds `relio releases delete <version>`, which
// mirrors exactly what the interactive browser's own delete does: the local
// git tag and its matching CHANGELOG.md section, and nothing else. The
// GitHub Release (if one was published) and anything pushed to a remote are
// deliberately untouched — see ReleasesDeleteLong.
func newReleasesDeleteCmd(f *releaseFlags) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <version>",
		Short: i18n.T(i18n.ReleasesDeleteShort),
		Long:  i18n.T(i18n.ReleasesDeleteLong),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, cfg, err := openRepoAndConfig(f.dir)
			if err != nil {
				return err
			}
			return runReleasesDelete(cmd, repo, cfg, args[0], yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, i18n.T(i18n.ReleasesDeleteFlagYesUsage))
	return cmd
}

// resolveTagName finds the tag repo.Tags() actually has for version,
// accepting it with or without a leading "v" (matching how
// internal/changelog's own section lookup treats the version).
func resolveTagName(repo *gitrepo.Repo, version string) (string, error) {
	tags, err := repo.Tags()
	if err != nil {
		return "", err
	}
	want := strings.TrimPrefix(version, "v")
	for _, t := range tags {
		if strings.TrimPrefix(t.Name, "v") == want {
			return t.Name, nil
		}
	}
	return "", fmt.Errorf(i18n.T(i18n.ReleasesDeleteNoSuchTag), version)
}

// runReleasesDelete deletes the local tag and its matching changelog
// section, via the same internal/releases.Delete the interactive browser
// uses — see that package for the shared two-step implementation. It never
// touches the remote GitHub Release.
//
// If version has no matching tag but the changelog still carries a section
// for it, this is treated as an interrupted prior delete (the tag was
// already removed, but a subsequent changelog failure left the section
// behind) and recovered by finishing just that step — see
// recoverStaleChangelogSection. Without this, a retry after such a failure
// has nothing left for resolveTagName to find and permanently gets "no such
// version" instead of a way to finish the cleanup.
func runReleasesDelete(cmd *cobra.Command, repo *gitrepo.Repo, cfg config.Config, version string, yes bool) error {
	out := cmd.OutOrStdout()
	writeLine := func(a ...any) error {
		_, err := fmt.Fprintln(out, a...)
		return err
	}

	changelogPath := filepath.Join(repo.Root(), cfg.Release.ChangelogFile)

	tagName, err := resolveTagName(repo, version)
	if err != nil {
		if releases.HasStaleChangelogSection(changelogPath, version) {
			return recoverStaleChangelogSection(cmd, changelogPath, version, yes)
		}
		return err
	}

	if !yes {
		if _, err := fmt.Fprint(out, i18n.T(i18n.ReleasesDeleteConfirmPrompt, tagName)); err != nil {
			return err
		}
		if !readYes(cmd.InOrStdin()) {
			return writeLine(ui.Info(i18n.T(i18n.ReleasesDeleteCancelled)))
		}
	}

	changelogRemoved, err := releases.Delete(repo, changelogPath, tagName)
	if err != nil {
		var cErr *releases.ChangelogError
		if errors.As(err, &cErr) {
			return fmt.Errorf(i18n.T(i18n.ReleasesChangelogWriteError), cErr.Err)
		}
		return err
	}

	removed := i18n.T(i18n.ReleasesRemovedTag)
	if changelogRemoved {
		removed = i18n.T(i18n.ReleasesRemovedTagAndChangelog)
	}
	return writeLine(ui.Success([]string{i18n.T(i18n.ReleasesDeleted, tagName, removed)}))
}

// recoverStaleChangelogSection finishes an interrupted `releases delete`:
// version has no matching git tag (it's already gone), but the changelog
// still has a section for it. It confirms (unless yes) then removes just
// that section — the tag side of the original delete already succeeded, so
// there is nothing left to do there.
func recoverStaleChangelogSection(cmd *cobra.Command, changelogPath, version string, yes bool) error {
	out := cmd.OutOrStdout()
	writeLine := func(a ...any) error {
		_, err := fmt.Fprintln(out, a...)
		return err
	}

	if !yes {
		if _, err := fmt.Fprint(out, i18n.T(i18n.ReleasesDeleteRecoverConfirmPrompt, version)); err != nil {
			return err
		}
		if !readYes(cmd.InOrStdin()) {
			return writeLine(ui.Info(i18n.T(i18n.ReleasesDeleteCancelled)))
		}
	}

	if _, err := releases.RemoveChangelogSection(changelogPath, version); err != nil {
		var cErr *releases.ChangelogError
		if errors.As(err, &cErr) {
			return fmt.Errorf(i18n.T(i18n.ReleasesChangelogWriteError), cErr.Err)
		}
		return err
	}

	return writeLine(ui.Success([]string{i18n.T(i18n.ReleasesRecoveredStaleChangelog, version)}))
}
