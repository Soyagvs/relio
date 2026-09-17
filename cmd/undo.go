package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/ui"
)

// newUndoCmd builds `relio undo`, which reverses the most recent local release
// while it is still local.
func newUndoCmd(f *releaseFlags) *cobra.Command {
	var yes, force bool
	cmd := &cobra.Command{
		Use:   "undo",
		Short: i18n.T(i18n.UndoShort),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, cfg, err := openRepoAndConfig(f.dir)
			if err != nil {
				return err
			}
			return runUndo(cmd, repo, cfg, yes, force)
		},
	}
	lf := cmd.Flags()
	lf.BoolVarP(&yes, "yes", "y", false, i18n.T(i18n.UndoFlagYesUsage))
	lf.BoolVar(&force, "force", false, i18n.T(i18n.UndoFlagForceUsage))
	return cmd
}

// runUndo deletes the latest tag and, when HEAD is its `chore(release)` commit,
// drops that commit with `git reset --hard HEAD~1`. It refuses once the release
// has been pushed or is no longer the current commit.
func runUndo(cmd *cobra.Command, repo *gitrepo.Repo, cfg config.Config, yes, force bool) error {
	out := cmd.OutOrStdout()
	writeLine := func(a ...any) error {
		_, err := fmt.Fprintln(out, a...)
		return err
	}

	tag, ok, err := repo.LatestTag()
	if err != nil {
		return err
	}
	if !ok {
		return writeLine(ui.Info(i18n.T(i18n.UndoNoTags)))
	}

	atHead, err := repo.TagPointsAtHead(tag)
	if err != nil {
		return err
	}
	if !atHead {
		return fmt.Errorf(i18n.T(i18n.UndoTagNotAtHead), tag)
	}

	pushed, err := repo.RemoteContainsHead()
	if err != nil {
		return err
	}
	if pushed {
		return fmt.Errorf(i18n.T(i18n.UndoAlreadyPushed), tag, tag)
	}

	subject, err := repo.HeadSubject()
	if err != nil {
		return err
	}
	isReleaseCommit := subject == "chore(release): "+tag

	if isReleaseCommit && !force {
		clean, cerr := repo.IsClean()
		if cerr != nil {
			return cerr
		}
		if !clean {
			return errors.New(i18n.T(i18n.UndoDirtyTree))
		}
	}

	if err := writeLine(ui.Key.Render(i18n.T(i18n.UndoHeader, tag))); err != nil {
		return err
	}
	if err := writeLine(ui.Dim.Render(i18n.T(i18n.UndoStepDeleteTag, tag))); err != nil {
		return err
	}
	if isReleaseCommit {
		if err := writeLine(ui.Dim.Render(i18n.T(i18n.UndoStepRemoveCommit, tag))); err != nil {
			return err
		}
		if err := writeLine(ui.Dim.Render(i18n.T(i18n.UndoStepFilesRevert, cfg.Release.ChangelogFile))); err != nil {
			return err
		}
	}
	if err := writeLine(); err != nil {
		return err
	}

	if !yes {
		if _, err := fmt.Fprint(out, i18n.T(i18n.UndoProceedPrompt)); err != nil {
			return err
		}
		if !readYes(cmd.InOrStdin()) {
			return writeLine(ui.Info(i18n.T(i18n.UndoCancelled)))
		}
	}

	if err := repo.DeleteTag(tag); err != nil {
		return err
	}

	done := []string{i18n.T(i18n.UndoDoneDeletedTag, tag)}
	if isReleaseCommit {
		if err := repo.ResetHardPrevious(); err != nil {
			return fmt.Errorf(i18n.T(i18n.UndoResetFailed), tag, err)
		}
		done = append(done, i18n.T(i18n.UndoDoneRemovedCommit))
	}

	return writeLine(ui.Success(done))
}
