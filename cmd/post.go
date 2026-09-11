package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/changelog"
	"github.com/soyagvs/relio/internal/conventional"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/pick"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/ui"
)

func newPostCmd(f *releaseFlags) *cobra.Command {
	var format string

	c := &cobra.Command{
		Use:   "post",
		Short: i18n.T(i18n.PostShort),
		Long:  i18n.T(i18n.PostLong),
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
			if plan.NothingToRelease() {
				fmt.Fprintln(cmd.ErrOrStderr(), ui.Info(i18n.T(i18n.PostNoCommits)))
				return nil
			}

			text, err := renderPost(cfg.Project, plan, format)
			if err != nil {
				return err
			}
			// Text only on stdout so it can be piped straight to the clipboard.
			fmt.Fprintln(cmd.ErrOrStderr(), ui.Dim.Render(i18n.T(i18n.PostCopyHint)))
			fmt.Fprintln(cmd.OutOrStdout(), text)
			return nil
		},
	}

	// The enumerated values here are the literal argument the user passes to
	// --format, so this usage string stays untranslated on purpose — see
	// postFormatItems below for the translated display labels shown in the
	// interactive picker.
	c.Flags().StringVar(&format, "format", "minimal", "minimal | social | technical | casual | changelog")
	return c
}

// postFormatItems is the menu of styles offered by `Release text` in the UI.
// It is a function, not a package-level var, so its Label/Desc strings are
// localized at call time — a var would freeze at "en" before Execute() ever
// resolves the language (the same class of bug help.go's refCommands/etc.
// hit in PR6a). Four of the five Desc strings are byte-identical to
// cmd/help.go's HelpPost*Desc keys and reuse them; only "Minimal" differs.
func postFormatItems() []pick.Item {
	return []pick.Item{
		{Label: i18n.T(i18n.PostFormatMinimalLabel), Desc: i18n.T(i18n.PostFormatMinimalDesc), Value: "minimal"},
		{Label: i18n.T(i18n.PostFormatSocialLabel), Desc: i18n.T(i18n.HelpPostSocialDesc), Value: "social"},
		{Label: i18n.T(i18n.PostFormatTechnicalLabel), Desc: i18n.T(i18n.HelpPostTechnicalDesc), Value: "technical"},
		{Label: i18n.T(i18n.PostFormatCasualLabel), Desc: i18n.T(i18n.HelpPostCasualDesc), Value: "casual"},
		{Label: i18n.T(i18n.PostFormatChangelogLabel), Desc: i18n.T(i18n.HelpPostChangelogDesc), Value: "changelog"},
	}
}

// renderPost builds the announcement text for the given format.
func renderPost(project string, plan release.Plan, format string) (string, error) {
	switch format {
	case "minimal", "":
		return minimalPost(project, plan), nil
	case "social":
		return socialPost(project, plan), nil
	case "technical", "tech":
		return technicalPost(project, plan), nil
	case "casual":
		return casualPost(project, plan), nil
	case "changelog":
		return ui.Markdownish(plan.Section()), nil
	default:
		return "", fmt.Errorf(i18n.T(i18n.PostUnknownFormat), format)
	}
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
// prints for a version: the "<cli> -- release" header, "<project> · <version>",
// "<date> · <time> · N commits", then the grouped notes.
func minimalPost(project string, p release.Plan) string {
	return ui.ReleaseText(project, p.Next.String(), commitMeta(p), p.Notes)
}

func commitMeta(p release.Plan) string {
	return ui.ReleaseMeta(p.Now.Format("2006-01-02 15:04"), len(p.Commits))
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// socialNotable are the commit types worth putting in a social post.
var socialNotable = map[string]bool{
	"feat": true, "fix": true, "perf": true, "refactor": true, "revert": true, "style": true,
}

type socialRow struct{ Type, Text string }

// socialRows keeps only the notable commits, as "type / Description" rows.
func socialRows(commits []conventional.Commit) []socialRow {
	var rows []socialRow
	for _, c := range commits {
		if !socialNotable[c.Type] {
			continue
		}
		desc := c.Description
		if desc == "" {
			desc = c.Raw
		}
		rows = append(rows, socialRow{Type: c.Type, Text: titleCase(strings.TrimSpace(desc))})
	}
	return rows
}

// socialPost is the shortest format:
//
//	Project -- Release
//
//	v1.4.0 · 06.09.26 · 13:47
//
//	feat  Facial attendance
//	fix   Supervisor login
func socialPost(project string, p release.Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", ui.Key.Render(i18n.T(i18n.PostSocialHeader, titleCase(project))))
	fmt.Fprintf(&b, "%s %s\n\n",
		ui.Ok.Render(p.Next.String()),
		ui.Dim.Render(fmt.Sprintf("· %s · %s", p.Now.Format("02.01.06"), p.Now.Format("15:04"))))

	rows := socialRows(p.Commits)
	if len(rows) == 0 {
		b.WriteString(ui.Dim.Render(i18n.T(i18n.PostSocialNoNotableChanges)))
		return b.String()
	}
	width := 0
	for _, r := range rows {
		if len(r.Type) > width {
			width = len(r.Type)
		}
	}
	for _, r := range rows {
		fmt.Fprintf(&b, "%s  %s\n", ui.Key.Render(fmt.Sprintf("%-*s", width, r.Type)), r.Text)
	}
	return strings.TrimRight(b.String(), "\n")
}

func technicalPost(project string, p release.Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n\n", ui.Key.Render(project), ui.Ok.Render(p.Next.String()))
	for _, it := range bulletList(p.Notes, 5) {
		fmt.Fprintf(&b, "%s %s\n", ui.Dim.Render("•"), it)
	}
	fmt.Fprintf(&b, "\n%s", ui.Dim.Render(i18n.T(i18n.PostTechnicalCommitsSummary, len(p.Commits), p.Next.String())))
	return b.String()
}

func casualPost(project string, p release.Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s %s\n\n", ui.Key.Render(project), ui.Ok.Render(p.Next.String()), ui.Dim.Render(i18n.T(i18n.PostCasualIsOut)))
	for _, it := range bulletList(p.Notes, 4) {
		fmt.Fprintf(&b, "%s %s\n", ui.Key.Render("→"), it)
	}
	return strings.TrimRight(b.String(), "\n")
}
