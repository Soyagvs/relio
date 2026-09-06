package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/ui"
)

func newStatusCmd(f *releaseFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show what's unreleased and the version it suggests",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, cfg, err := openRepoAndConfig(f.dir)
			if err != nil {
				return err
			}
			return runStatus(cmd, repo, cfg)
		},
	}
}

func runStatus(cmd *cobra.Command, repo *gitrepo.Repo, cfg config.Config) error {
	out := cmd.OutOrStdout()

	plan, err := release.BuildPlan(repo, cfg, release.Options{})
	if err != nil {
		return err
	}

	row := func(label, val string) {
		fmt.Fprintf(out, "%s %s\n", ui.Dim.Render(fmt.Sprintf("%-12s", label)), val)
	}

	fmt.Fprintln(out, ui.Key.Render(cfg.Project))
	fmt.Fprintln(out)

	cur := "none"
	if plan.Current.Major != 0 || plan.Current.Minor != 0 || plan.Current.Patch != 0 {
		cur = plan.Current.String()
	}
	row("Current", cur)
	row("Unreleased", fmt.Sprintf("%d commits", len(plan.Commits)))

	if counts := typeCounts(plan); len(counts) > 0 {
		w := 0
		for _, c := range counts {
			if len(c.typ) > w {
				w = len(c.typ)
			}
		}
		fmt.Fprintln(out)
		for _, c := range counts {
			fmt.Fprintf(out, "%s  %d\n", ui.Key.Render(fmt.Sprintf("%-*s", w, c.typ)), c.n)
		}
	}

	fmt.Fprintln(out)
	if plan.NothingToRelease() {
		row("Suggested", ui.Dim.Render("—"))
		fmt.Fprintln(out)
		fmt.Fprintln(out, ui.Info("Nothing to release."))
		return nil
	}
	row("Suggested", ui.Ok.Render(plan.Next.String())+ui.Dim.Render("  ("+plan.Bump.String()+")"))
	fmt.Fprintln(out)

	if clean, cerr := repo.IsClean(); cerr == nil && !clean {
		fmt.Fprintln(out, ui.Warn.Render("! ")+ui.Dim.Render("uncommitted changes in the working tree"))
	}
	fmt.Fprintln(out, ui.Ok.Render("Ready to release."))
	return nil
}

type typeCount struct {
	typ string
	n   int
}

// typeCounts tallies commit types in the unreleased range, most frequent first.
func typeCounts(p release.Plan) []typeCount {
	m := map[string]int{}
	for _, c := range p.Commits {
		t := c.Type
		if t == "" {
			t = "other"
		}
		m[t]++
	}
	out := make([]typeCount, 0, len(m))
	for t, n := range m {
		out = append(out, typeCount{t, n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].n != out[j].n {
			return out[i].n > out[j].n
		}
		return out[i].typ < out[j].typ
	})
	return out
}
