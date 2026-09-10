package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/ui"
)

// newUndoCmd builds `relio undo`, which reverses the most recent local release
// while it is still local.
func newUndoCmd(f *releaseFlags) *cobra.Command {
	var yes, force bool
	cmd := &cobra.Command{
		Use:   "undo",
		Short: "Reverse the most recent local release (before it is pushed)",
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
	lf.BoolVarP(&yes, "yes", "y", false, "skip the confirmation prompt")
	lf.BoolVar(&force, "force", false, "undo even with a dirty working tree (git reset --hard discards uncommitted changes)")
	return cmd
}

// runUndo deletes the latest tag and, when HEAD is its `chore(release)` commit,
// drops that commit with `git reset --hard HEAD~1`. It refuses once the release
// has been pushed or is no longer the current commit.
func runUndo(cmd *cobra.Command, repo *gitrepo.Repo, cfg config.Config, yes, force bool) error {
	out := cmd.OutOrStdout()

	tag, ok, err := repo.LatestTag()
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintln(out, ui.Info("No tags yet — nothing to undo."))
		return nil
	}

	atHead, err := repo.TagPointsAtHead(tag)
	if err != nil {
		return err
	}
	if !atHead {
		return fmt.Errorf("%s does not point at HEAD — the last release is not the current commit, nothing to undo safely", tag)
	}

	pushed, err := repo.RemoteContainsHead()
	if err != nil {
		return err
	}
	if pushed {
		return fmt.Errorf("%s is already on a remote — undo would rewrite shared history.\n"+
			"  remove it on the remote yourself:  git push origin :%s\n"+
			"  and delete the GitHub Release if you created one", tag, tag)
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
			return errors.New("the working tree has uncommitted changes — commit or stash them first, or re-run with --force")
		}
	}

	fmt.Fprintln(out, ui.Key.Render("Undo "+tag))
	fmt.Fprintln(out, ui.Dim.Render("  · delete the local tag "+tag))
	if isReleaseCommit {
		fmt.Fprintln(out, ui.Dim.Render("  · remove the `chore(release): "+tag+"` commit (git reset --hard HEAD~1)"))
		fmt.Fprintln(out, ui.Dim.Render("    "+cfg.Release.ChangelogFile+" and any version files return to their previous state"))
	}
	fmt.Fprintln(out)

	if !yes {
		fmt.Fprint(out, "  Proceed? [y/N] ")
		if !readYes(cmd.InOrStdin()) {
			fmt.Fprintln(out, ui.Info("Cancelled. Nothing changed."))
			return nil
		}
	}

	if err := repo.DeleteTag(tag); err != nil {
		return err
	}

	done := []string{"deleted tag " + tag}
	if isReleaseCommit {
		if err := repo.ResetHardPrevious(); err != nil {
			return fmt.Errorf("tag %s deleted, but removing the release commit failed (finish with "+"`git reset --hard HEAD~1`"+"): %w", tag, err)
		}
		done = append(done, "removed the release commit")
	}

	fmt.Fprintln(out, ui.Success(done))
	return nil
}
