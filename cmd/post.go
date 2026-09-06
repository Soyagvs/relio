package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/soyagvs/go-release/internal/changelog"
	"github.com/soyagvs/go-release/internal/release"
	"github.com/soyagvs/go-release/internal/ui"
)

func newPostCmd(f *releaseFlags) *cobra.Command {
	var format string

	c := &cobra.Command{
		Use:   "post",
		Short: "Generate a short announcement from the pending release",
		Long: "Build social/changelog copy from commits since the last tag.\n" +
			"Experimental preview of the v0.3.0 content generator — it prints text, nothing is published.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			repo, cfg, err := openRepoAndConfig(f.dir)
			if err != nil {
				return err
			}
			plan, err := release.BuildPlan(repo, cfg, release.Options{})
			if err != nil {
				return err
			}
			if plan.NothingToRelease() {
				fmt.Fprintln(out, ui.Info("No commits since the last tag — nothing to announce."))
				return nil
			}

			var text string
			switch format {
			case "technical", "tech":
				text = technicalPost(cfg.Project, plan)
			case "casual":
				text = casualPost(cfg.Project, plan)
			case "changelog":
				text = plan.Section()
			default:
				return fmt.Errorf("unknown format %q (technical|casual|changelog)", format)
			}

			fmt.Fprintln(out, ui.Banner(cfg.Project, version))
			fmt.Fprintln(out)
			fmt.Fprintln(out, text)
			return nil
		},
	}

	c.Flags().StringVar(&format, "format", "technical", "technical | casual | changelog")
	return c
}

func bulletList(n changelog.Notes, max int) []string {
	order := []changelog.Group{changelog.Added, changelog.Changed, changelog.Fixed}
	var items []string
	for _, g := range order {
		for _, it := range n.Groups[g] {
			items = append(items, strings.TrimPrefix(it, "**Breaking:** "))
		}
	}
	if max > 0 && len(items) > max {
		items = items[:max]
	}
	return items
}

func technicalPost(project string, p release.Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — %s\n\n", project, p.Next.String())
	for _, it := range bulletList(p.Notes, 5) {
		fmt.Fprintf(&b, "• %s\n", it)
	}
	fmt.Fprintf(&b, "\n%d commits · %s", len(p.Commits), p.Next.String())
	return b.String()
}

func casualPost(project string, p release.Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s is out.\n\n", project, p.Next.String())
	for _, it := range bulletList(p.Notes, 4) {
		fmt.Fprintf(&b, "→ %s\n", it)
	}
	return strings.TrimRight(b.String(), "\n")
}
