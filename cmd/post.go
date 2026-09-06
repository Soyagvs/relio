package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/soyagvs/go-release/internal/changelog"
	"github.com/soyagvs/go-release/internal/conventional"
	"github.com/soyagvs/go-release/internal/pick"
	"github.com/soyagvs/go-release/internal/release"
	"github.com/soyagvs/go-release/internal/ui"
)

func newPostCmd(f *releaseFlags) *cobra.Command {
	var format string

	c := &cobra.Command{
		Use:   "post",
		Short: "Generate copy-paste release text for social posts",
		Long: "Build a short, plain-text announcement from commits since the last tag.\n" +
			"Only the text goes to stdout, so `go-release post | pbcopy` works cleanly.\n" +
			"Experimental preview of the v0.3.0 content generator — nothing is published.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, cfg, err := openRepoAndConfig(f.dir)
			if err != nil {
				return err
			}
			plan, err := release.BuildPlan(repo, cfg, release.Options{})
			if err != nil {
				return err
			}
			if plan.NothingToRelease() {
				fmt.Fprintln(cmd.ErrOrStderr(), ui.Info("No commits since the last tag — nothing to announce."))
				return nil
			}

			text, err := renderPost(cfg.Project, plan, format)
			if err != nil {
				return err
			}
			// Text only on stdout so it can be piped straight to the clipboard.
			fmt.Fprintln(cmd.ErrOrStderr(), ui.Dim.Render("# release text — copy from here:"))
			fmt.Fprintln(cmd.OutOrStdout(), text)
			return nil
		},
	}

	c.Flags().StringVar(&format, "format", "minimal", "minimal | technical | casual | changelog")
	return c
}

// postFormats is the menu of styles offered by `Release text` in the UI.
var postFormats = []pick.Item{
	{Label: "Minimal", Desc: "Same as the releases browser: project, version, date, commits, grouped notes with hashes", Value: "minimal"},
	{Label: "Technical", Desc: "Terse bullet list, for a changelog or a dev channel", Value: "technical"},
	{Label: "Casual", Desc: "Loose tone: \"proj v1.4.0 is out. → …\"", Value: "casual"},
	{Label: "Changelog", Desc: "The exact section that goes into CHANGELOG.md", Value: "changelog"},
}

// renderPost builds the announcement text for the given format.
func renderPost(project string, plan release.Plan, format string) (string, error) {
	switch format {
	case "minimal", "":
		return minimalPost(project, plan), nil
	case "technical", "tech":
		return technicalPost(project, plan), nil
	case "casual":
		return casualPost(project, plan), nil
	case "changelog":
		return ui.Markdownish(plan.Section()), nil
	default:
		return "", fmt.Errorf("unknown format %q (minimal|technical|casual|changelog)", format)
	}
}

// headHash is the abbreviated hash of the newest commit in the release
// (BuildPlan returns commits oldest-first, so HEAD is last). "" when unknown.
func headHash(commits []conventional.Commit) string {
	if len(commits) == 0 {
		return ""
	}
	h := commits[len(commits)-1].Hash
	if len(h) > 7 {
		h = h[:7]
	}
	return h
}

// groupItems returns up to limit items of one changelog group, breaking prefix removed.
func groupItems(n changelog.Notes, g changelog.Group, limit int) []string {
	var items []string
	for _, it := range n.Groups[g] {
		items = append(items, strings.TrimPrefix(it.Text, "**Breaking:** "))
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func bulletList(n changelog.Notes, max int) []string {
	order := []changelog.Group{changelog.Added, changelog.Changed, changelog.Fixed}
	var items []string
	for _, g := range order {
		items = append(items, groupItems(n, g, 0)...)
	}
	if max > 0 && len(items) > max {
		items = items[:max]
	}
	return items
}

// minimalPost is the default and is identical to what the releases browser
// prints for a version: "<project> -- release", version, date/time and commit
// range, then the grouped notes with commit hashes.
func minimalPost(project string, p release.Plan) string {
	return ui.ReleaseText(project, p.Next.String(), commitMeta(p), p.Notes)
}

// commitMeta is the "<date time>  ·  N commits (from..head)" line.
func commitMeta(p release.Plan) string {
	when := p.Now.Format("2006-01-02 15:04")
	head := headHash(p.Commits)
	firstRelease := p.Current.Major == 0 && p.Current.Minor == 0 && p.Current.Patch == 0
	switch {
	case head == "":
		return fmt.Sprintf("%s  ·  %d commits", when, len(p.Commits))
	case firstRelease:
		return fmt.Sprintf("%s  ·  %d commits (%s)", when, len(p.Commits), head)
	default:
		return fmt.Sprintf("%s  ·  %d commits (%s..%s)", when, len(p.Commits), p.Current.String(), head)
	}
}

func technicalPost(project string, p release.Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n\n", ui.Key.Render(project), ui.Ok.Render(p.Next.String()))
	for _, it := range bulletList(p.Notes, 5) {
		fmt.Fprintf(&b, "%s %s\n", ui.Dim.Render("•"), it)
	}
	fmt.Fprintf(&b, "\n%s", ui.Dim.Render(fmt.Sprintf("%d commits · %s", len(p.Commits), p.Next.String())))
	return b.String()
}

func casualPost(project string, p release.Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s %s\n\n", ui.Key.Render(project), ui.Ok.Render(p.Next.String()), ui.Dim.Render("is out."))
	for _, it := range bulletList(p.Notes, 4) {
		fmt.Fprintf(&b, "%s %s\n", ui.Key.Render("→"), it)
	}
	return strings.TrimRight(b.String(), "\n")
}
