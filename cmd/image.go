package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/card"
	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/conventional"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/pick"
	"github.com/soyagvs/relio/internal/ui"
)

func newImageCmd(f *releaseFlags) *cobra.Command {
	var shape, version string

	c := &cobra.Command{
		Use:   "image",
		Short: "Save a shareable PNG of a release",
		Long: "Pick a release and a shape (horizontal / vertical / square) and write a\n" +
			"PNG card of it into the current directory. Pass --version and --shape to\n" +
			"skip the prompts (also skipped automatically when there is no TTY).",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, cfg, err := openRepoAndConfig(f.dir)
			if err != nil {
				return err
			}
			return runReleaseImage(cmd, repo, cfg, version, shape)
		},
	}

	c.Flags().StringVar(&version, "version", "", "release tag to render (default: latest)")
	c.Flags().StringVar(&shape, "shape", "", "horizontal | vertical | square (default: horizontal)")
	return c
}

var shapeItems = []pick.Item{
	{Label: "Horizontal", Desc: "1200×630 — Twitter / OpenGraph", Value: "horizontal"},
	{Label: "Vertical", Desc: "1080×1350 — Instagram portrait", Value: "vertical"},
	{Label: "Square", Desc: "1080×1080", Value: "square"},
}

func runReleaseImage(cmd *cobra.Command, repo *gitrepo.Repo, cfg config.Config, versionFlag, shapeFlag string) error {
	out := cmd.OutOrStdout()
	interactive := stdinIsTTY() && stdoutIsTTY()

	tags, err := repo.Tags()
	if err != nil {
		return err
	}
	if len(tags) == 0 {
		fmt.Fprintln(out, ui.Info("No releases yet — create one first."))
		return nil
	}

	// Resolve the release: flag, else prompt (TTY), else latest.
	version := versionFlag
	if version == "" {
		if interactive {
			items := make([]pick.Item, len(tags))
			for i, t := range tags {
				items[i] = pick.Item{Label: t.Name, Desc: t.DateTime + " · " + t.Subject, Value: t.Name}
			}
			var ok bool
			if version, ok, err = pick.Run("Pick a release", items); err != nil || !ok {
				return err
			}
		} else {
			version = tags[0].Name // newest
		}
	}

	idx := -1
	for i, t := range tags {
		if t.Name == version {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("no such release %q", version)
	}

	// Resolve the shape: flag, else prompt (TTY), else horizontal.
	shape, valid := card.ParseShape(shapeFlag)
	if !valid {
		if interactive {
			name, ok, perr := pick.Run("Pick a shape", shapeItems)
			if perr != nil || !ok {
				return perr
			}
			shape, _ = card.ParseShape(name)
		} else {
			shape = card.Horizontal
		}
	}

	// Gather the commits for that version.
	from := ""
	if idx+1 < len(tags) {
		from = tags[idx+1].Name
	}
	raw, err := repo.CommitsBetween(from, version)
	if err != nil {
		return err
	}
	commits := conventional.ParseMany(raw)

	c := card.Card{
		Project: titleCase(cfg.Project),
		Version: version,
		Meta:    imageMeta(tags[idx].DateTime, tags[idx].Date, len(raw)),
		Rows:    socialRows(commits),
		Author:  ui.Author,
	}

	path := filepath.Join(".", fmt.Sprintf("relio-%s-%s.png", version, shape))
	if err := card.Save(card.Render(c, shape), path); err != nil {
		return err
	}
	abs, _ := filepath.Abs(path)

	fmt.Fprintln(out, ui.Success([]string{"saved " + path}))
	fmt.Fprintln(out, ui.Dim.Render("  "+abs))
	return nil
}

// imageMeta renders "DD.MM.YY · HH:MM · N commits" from a "2006-01-02 15:04"
// string, falling back to the plain date.
func imageMeta(datetime, date string, commits int) string {
	if t, err := time.Parse("2006-01-02 15:04", datetime); err == nil {
		return fmt.Sprintf("%s · %s · %d commits", t.Format("02.01.06"), t.Format("15:04"), commits)
	}
	if date != "" {
		return fmt.Sprintf("%s · %d commits", date, commits)
	}
	return fmt.Sprintf("%d commits", commits)
}
