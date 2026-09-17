package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/ui"
)

func newStatusCmd(f *releaseFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: i18n.T(i18n.StatusShort),
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

	row := func(label, val string) error {
		_, err := fmt.Fprintf(out, "%s %s\n", ui.Dim.Render(fmt.Sprintf("%-12s", label)), val)
		return err
	}

	if _, err := fmt.Fprintln(out, ui.Key.Render(cfg.Project)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	cur := "none"
	if plan.Current.Major != 0 || plan.Current.Minor != 0 || plan.Current.Patch != 0 {
		cur = plan.Current.String()
	}
	if err := row(i18n.T(i18n.StatusLabelCurrent), cur); err != nil {
		return err
	}
	if err := row(i18n.T(i18n.StatusLabelUnreleased), i18n.T(i18n.StatusCommitsCount, len(plan.Commits))); err != nil {
		return err
	}

	if counts := typeCounts(plan); len(counts) > 0 {
		w := 0
		for _, c := range counts {
			if len(c.typ) > w {
				w = len(c.typ)
			}
		}
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
		for _, c := range counts {
			if _, err := fmt.Fprintf(out, "%s  %d\n", ui.Key.Render(fmt.Sprintf("%-*s", w, c.typ)), c.n); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	if plan.NothingToRelease() {
		if err := row(i18n.T(i18n.StatusLabelSuggested), ui.Dim.Render(i18n.T(i18n.StatusNoSuggestion))); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
		_, err := fmt.Fprintln(out, ui.Info(i18n.T(i18n.StatusNothingToRelease)))
		return err
	}
	if err := row(i18n.T(i18n.StatusLabelSuggested), ui.Ok.Render(plan.Next.String())+ui.Dim.Render("  ("+plan.Bump.String()+")")); err != nil {
		return err
	}
	if plan.Current.IsPrerelease() {
		if _, err := fmt.Fprintln(out, ui.Dim.Render(i18n.T(i18n.StatusPrereleaseHint, plan.Current.Core().String()))); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	if clean, cerr := repo.IsClean(); cerr == nil && !clean {
		if _, err := fmt.Fprintln(out, ui.Warn.Render("! ")+ui.Dim.Render(i18n.T(i18n.StatusUncommittedChange))); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(out, ui.Ok.Render(i18n.T(i18n.StatusReadyToRelease)))
	return err
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
