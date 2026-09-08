package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/conventional"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/ui"
)

func newCheckCmd(f *releaseFlags) *cobra.Command {
	var strict bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check the commits since the last tag before releasing",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, cfg, err := openRepoAndConfig(f.dir)
			if err != nil {
				return err
			}
			plan, err := release.BuildPlan(repo, cfg, release.Options{})
			if err != nil {
				return err
			}
			return runCheck(cmd, repo, cfg, plan, strict)
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", false, "exit non-zero when any commit is not a Conventional Commit")
	return cmd
}

// runCheck gathers the tag and the conventional/non-conventional split, renders
// the report, and — with strict — returns an error when the history is not clean.
func runCheck(cmd *cobra.Command, repo *gitrepo.Repo, cfg config.Config, plan release.Plan, strict bool) error {
	tag, hasTag, err := repo.LatestTag()
	if err != nil {
		return err
	}
	if !hasTag {
		tag = ""
	}

	conv, nonConv := plan.Lint()

	if err := checkReport(cmd.OutOrStdout(), cfg.Project, tag, len(plan.Commits),
		conv, nonConv, plan.Bump.String(), plan.Next.String(), ui.HideHashes); err != nil {
		return err
	}

	if strict && len(plan.Commits) > 0 && len(nonConv) > 0 {
		return fmt.Errorf("%d commit(s) are not Conventional Commits (--strict)", len(nonConv))
	}
	return nil
}

// checkReport is the pure view for `relio check`: every input is a plain value so
// it renders without a repo. It follows the visual grammar of `relio status` —
// the project key, dim labels, blank-line spacing.
func checkReport(w io.Writer, project, tag string, nCommits int, conv, nonConv []conventional.Commit, bump, next string, hideHashes bool) error {
	base := tag
	if base == "" {
		base = "the last tag"
	}

	if nCommits == 0 {
		fmt.Fprintln(w, ui.Info(fmt.Sprintf("Nothing to check — no commits since %s.", base)))
		return nil
	}

	fmt.Fprintln(w, ui.Key.Render(project))
	fmt.Fprintln(w)

	if tag == "" {
		fmt.Fprintf(w, "%d commits (no tag yet)\n", nCommits)
	} else {
		fmt.Fprintf(w, "%d commits since %s\n", nCommits, tag)
	}

	fmt.Fprintf(w, "  %s %s\n",
		ui.Ok.Render("✓"), ui.Dim.Render(fmt.Sprintf("%d conventional", len(conv))))

	if len(nonConv) > 0 {
		fmt.Fprintf(w, "  %s %s\n",
			ui.Warn.Render("✗"), ui.Dim.Render(fmt.Sprintf("%d not conventional:", len(nonConv))))
		for _, c := range nonConv {
			if hideHashes || c.Hash == "" {
				fmt.Fprintf(w, "      %s\n", c.Raw)
				continue
			}
			fmt.Fprintf(w, "      %s  %s\n", ui.Dim.Render(shortHash(c.Hash)), c.Raw)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Detected bump: %s  →  %s\n", bump, next)
	return nil
}

// shortHash trims a commit hash to the conventional 7-character prefix.
func shortHash(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}
