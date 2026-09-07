package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/card"
	"github.com/soyagvs/relio/internal/changelog"
	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/conventional"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/pick"
	"github.com/soyagvs/relio/internal/ui"
	"github.com/soyagvs/relio/internal/upload"
)

type imageFlags struct {
	version  string
	shape    string
	theme    string
	hash     bool
	upload   bool
	linkOnly bool
}

func newImageCmd(f *releaseFlags) *cobra.Command {
	var im imageFlags

	c := &cobra.Command{
		Use:   "image",
		Short: "Make a shareable image of a release",
		Long: "Render a dark, developer-styled release card. Everything on it comes\n" +
			"from the real release.\n\n" +
			"In a terminal, Relio asks what to do with it: save it to the current\n" +
			"directory, upload it for a QR + link, or both. With no TTY and no flags\n" +
			"it just saves to the current directory.\n\n" +
			"Flags skip the prompts:\n" +
			"  --version    release tag            --shape   horizontal|vertical|square\n" +
			"  --theme      orange|green|purple    --hash    show commit hashes\n" +
			"  --upload     also upload to a temp host (litterbox, 72h): link + QR\n" +
			"  --link-only  upload only — do not write a file to disk",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, cfg, err := openRepoAndConfig(f.dir)
			if err != nil {
				return err
			}
			return runReleaseImage(cmd, repo, cfg, im)
		},
	}

	c.Flags().StringVar(&im.version, "version", "", "release tag to render (default: latest)")
	c.Flags().StringVar(&im.shape, "shape", "", "horizontal | vertical | square (default: horizontal)")
	c.Flags().StringVar(&im.theme, "theme", "", "orange | green | purple (default: orange)")
	c.Flags().BoolVar(&im.hash, "hash", false, "show the commit hash on each line")
	c.Flags().BoolVar(&im.upload, "upload", false, "also upload to a temp host (litterbox 72h) and show a link + QR")
	c.Flags().BoolVar(&im.linkOnly, "link-only", false, "upload for a link + QR without writing a local file")
	return c
}

var shapeItems = []pick.Item{
	{Label: "Horizontal", Desc: "1200×630 — Twitter / OpenGraph", Value: "horizontal"},
	{Label: "Vertical", Desc: "1080×1920 — Instagram story", Value: "vertical"},
	{Label: "Square", Desc: "1080×1080", Value: "square"},
}

var themeItems = []pick.Item{
	{Label: "Orange", Desc: "The Relio default", Value: "orange"},
	{Label: "Green", Desc: "Teal-green accent", Value: "green"},
	{Label: "Purple", Desc: "Purple accent", Value: "purple"},
}

func runReleaseImage(cmd *cobra.Command, repo *gitrepo.Repo, cfg config.Config, im imageFlags) error {
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
	version := im.version
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
	shape, valid := card.ParseShape(im.shape)
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

	// Resolve the theme: flag, else prompt (TTY), else orange.
	theme := strings.ToLower(im.theme)
	if !validTheme(theme) {
		if interactive {
			name, ok, perr := pick.Run("Pick a colour", themeItems)
			if perr != nil || !ok {
				return perr
			}
			theme = name
		} else {
			theme = "orange"
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
	notes := changelog.Build(conventional.ParseMany(raw))

	c := card.Card{
		Project: titleCase(cfg.Project),
		Version: version,
		Meta:    imageMeta(tags[idx].DateTime, tags[idx].Date, len(raw)),
		Groups:  cardGroups(notes),
	}
	opt := card.Options{Theme: theme, ShowHash: im.hash}

	img := card.Render(c, shape, opt)
	localPath := filepath.Join(".", fmt.Sprintf("relio-%s-%s.png", version, shape))
	destDir := "."
	if a, aerr := filepath.Abs(localPath); aerr == nil {
		destDir = filepath.Dir(a)
	}

	// Decide what to do with the image: save it, upload it for a link, or both.
	save, link, ask := imageDest(im, interactive)
	if ask {
		ans, ok, perr := pick.Run("What should Relio do with the image?", []pick.Item{
			{Label: "Save + download link", Desc: "Write it to " + destDir + " and upload for a QR + link", Value: "both"},
			{Label: "Save only", Desc: "Write it to " + destDir, Value: "save"},
			{Label: "Download link only", Desc: "Upload for a QR + link — nothing written to disk", Value: "link"},
		})
		if perr != nil || !ok {
			return perr
		}
		save, link = destFromAnswer(ans)
	}

	uploadSrc := ""
	if save {
		if err := card.Save(img, localPath); err != nil {
			return err
		}
		abs, _ := filepath.Abs(localPath)
		fmt.Fprintln(out, ui.Success([]string{"saved " + localPath}))
		fmt.Fprintln(out, ui.Dim.Render("  "+abs))
		uploadSrc = localPath
	}

	if link {
		// No local copy to send — render to a temp file just for the upload.
		if uploadSrc == "" {
			tmp, terr := os.CreateTemp("", "relio-*.png")
			if terr != nil {
				return terr
			}
			tmp.Close()
			defer os.Remove(tmp.Name())
			if err := card.Save(img, tmp.Name()); err != nil {
				return err
			}
			uploadSrc = tmp.Name()
		}

		if save {
			fmt.Fprintln(out)
		}
		url, uerr := upload.Upload(uploadSrc)
		if uerr != nil {
			fmt.Fprintln(out, ui.Warn.Render("✗ ")+uerr.Error())
			if save {
				fmt.Fprintln(out, ui.Dim.Render("  the image is still saved locally"))
			}
			return nil
		}
		fmt.Fprint(out, qrBlock(url))
		fmt.Fprintln(out, "  "+ui.Key.Render(url))
	}
	return nil
}

// imageDest reads the flags and TTY state to decide what to do with the render.
// When ask is true the caller runs the interactive save/link/both prompt.
//
//	--link-only        -> upload only
//	--upload           -> save and upload
//	interactive, else  -> ask
//	no TTY, no flags    -> save only (keeps scripts working)
func imageDest(im imageFlags, interactive bool) (save, link, ask bool) {
	switch {
	case im.linkOnly:
		return false, true, false
	case im.upload:
		return true, true, false
	case interactive:
		return false, false, true
	default:
		return true, false, false
	}
}

// destFromAnswer maps the prompt's Value ("both" / "save" / "link") to the
// save and link switches.
func destFromAnswer(ans string) (save, link bool) {
	return ans == "both" || ans == "save", ans == "both" || ans == "link"
}

// qrBlock renders an ANSI QR of s (black/white cells so it scans on any terminal
// theme, and mosh passes the colour codes fine), each line indented two spaces.
func qrBlock(s string) string {
	var buf bytes.Buffer
	qrterminal.GenerateWithConfig(s, qrterminal.Config{
		Level:     qrterminal.L,
		Writer:    &buf,
		QuietZone: 2,
		BlackChar: qrterminal.BLACK,
		WhiteChar: qrterminal.WHITE,
	})
	var b strings.Builder
	for _, ln := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		b.WriteString("  " + ln + "\n")
	}
	return b.String()
}

func validTheme(s string) bool {
	for _, t := range card.ThemeNames {
		if s == t {
			return true
		}
	}
	return false
}

// cardGroups turns changelog notes into the card's Added / Changed / Fixed
// sections (Removed/Deprecated fold into Changed, Security into Fixed).
func cardGroups(n changelog.Notes) []card.Group {
	build := func(kind card.GroupKind, title string, from ...changelog.Group) *card.Group {
		var items []card.Item
		for _, g := range from {
			for _, it := range n.Groups[g] {
				items = append(items, card.Item{
					Text: strings.TrimPrefix(it.Text, "**Breaking:** "),
					Hash: it.Hash,
				})
			}
		}
		if len(items) == 0 {
			return nil
		}
		return &card.Group{Kind: kind, Title: title, Items: items}
	}

	var out []card.Group
	for _, g := range []*card.Group{
		build(card.KindAdded, "Added", changelog.Added),
		build(card.KindChanged, "Changed", changelog.Changed, changelog.Removed, changelog.Deprecated),
		build(card.KindFixed, "Fixed", changelog.Fixed, changelog.Security),
	} {
		if g != nil {
			out = append(out, *g)
		}
	}
	return out
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
