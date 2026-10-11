package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/pick"
	relbrowser "github.com/soyagvs/relio/internal/releases"
	"github.com/soyagvs/relio/internal/ui"
	"github.com/soyagvs/relio/internal/upload"
)

type imageFlags struct {
	version  string
	shape    string
	design   string
	theme    string
	hash     bool
	upload   bool
	linkOnly bool
}

func newImageCmd(f *releaseFlags) *cobra.Command {
	var im imageFlags

	c := &cobra.Command{
		Use:   "image",
		Short: i18n.T(i18n.ImageShort),
		Long:  i18n.T(i18n.ImageLong),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, cfg, err := openRepoAndConfig(f.dir)
			if err != nil {
				return err
			}
			_, err = runReleaseImage(cmd, repo, cfg, im)
			return err
		},
	}

	c.Flags().StringVar(&im.version, "version", "", i18n.T(i18n.ImageFlagVersionUsage))
	// --shape/--theme list the literal values the user types verbatim, so
	// these usage strings stay untranslated on purpose — see shapeItems()/
	// themeItems() below for the translated display labels shown in the
	// interactive pickers, matching cmd/post.go's --format convention.
	c.Flags().StringVar(&im.shape, "shape", "", "horizontal | vertical | square (default: horizontal)")
	c.Flags().StringVar(&im.design, "design", "", "default | minimal | bold | neon (default: default)")
	c.Flags().StringVar(&im.theme, "theme", "", "orange | green | purple (default design); design-specific values with --design")
	c.Flags().BoolVar(&im.hash, "hash", false, i18n.T(i18n.ImageFlagHashUsage))
	c.Flags().BoolVar(&im.upload, "upload", false, i18n.T(i18n.ImageFlagUploadUsage))
	c.Flags().BoolVar(&im.linkOnly, "link-only", false, i18n.T(i18n.ImageFlagLinkOnlyUsage))
	return c
}

// shapeItems and themeItems are the menus offered by the --shape/--theme
// pickers. They are functions, not package-level vars, so their Label/Desc
// strings are localized at call time — a var would freeze at "en" before
// Execute() ever resolves the language (the same class of bug postFormatItems()
// fixed in cmd/post.go, PR6c).
func shapeItems() []pick.Item {
	return []pick.Item{
		{Label: i18n.T(i18n.ImageShapeHorizontalLabel), Desc: i18n.T(i18n.ImageShapeHorizontalDesc), Value: "horizontal"},
		{Label: i18n.T(i18n.ImageShapeVerticalLabel), Desc: i18n.T(i18n.ImageShapeVerticalDesc), Value: "vertical"},
		{Label: i18n.T(i18n.ImageShapeSquareLabel), Desc: i18n.T(i18n.ImageShapeSquareDesc), Value: "square"},
	}
}

func designItems() []pick.Item {
	return []pick.Item{
		{Label: i18n.T(i18n.ImageDesignDefaultLabel), Desc: i18n.T(i18n.ImageDesignDefaultDesc), Value: "default"},
		{Label: i18n.T(i18n.ImageDesignMinimalLabel), Desc: i18n.T(i18n.ImageDesignMinimalDesc), Value: "minimal"},
		{Label: i18n.T(i18n.ImageDesignBoldLabel), Desc: i18n.T(i18n.ImageDesignBoldDesc), Value: "bold"},
		{Label: i18n.T(i18n.ImageDesignNeonLabel), Desc: i18n.T(i18n.ImageDesignNeonDesc), Value: "neon"},
	}
}

func themeItems(design card.Design) []pick.Item {
	labels := map[string]pick.Item{
		"orange": {Label: i18n.T(i18n.ImageThemeOrangeLabel), Desc: i18n.T(i18n.ImageThemeOrangeDesc), Value: "orange"},
		"green":  {Label: i18n.T(i18n.ImageThemeGreenLabel), Desc: i18n.T(i18n.ImageThemeGreenDesc), Value: "green"},
		"purple": {Label: i18n.T(i18n.ImageThemePurpleLabel), Desc: i18n.T(i18n.ImageThemePurpleDesc), Value: "purple"},
	}
	var out []pick.Item
	for _, name := range card.ThemesForDesign(design) {
		if it, ok := labels[name]; ok {
			out = append(out, it)
		} else {
			out = append(out, pick.Item{Label: titleCase(name), Desc: i18n.T(i18n.ImageThemeDesignSpecificDesc, name), Value: name})
		}
	}
	return out
}

func runReleaseImage(cmd *cobra.Command, repo *gitrepo.Repo, cfg config.Config, im imageFlags) (bool, error) {
	out := cmd.OutOrStdout()
	interactive := stdinIsTTY() && stdoutIsTTY()

	tags, err := repo.Tags()
	if err != nil {
		return false, err
	}
	if len(tags) == 0 {
		_, err := fmt.Fprintln(out, ui.Info(i18n.T(i18n.ImageNoReleasesYet)))
		return true, err
	}

	if interactive && im.version == "" {
		for {
			version, ok, err := relbrowser.RunSelect(repo, cfg)
			if err != nil || !ok {
				return false, err
			}

			shape, ok, err := pickImageShape(im, interactive)
			if err != nil {
				return false, err
			}
			if !ok {
				continue
			}

			for {
				design, ok, err := pickImageDesign(im, interactive)
				if err != nil {
					return false, err
				}
				if !ok {
					continue
				}

				theme, ok, err := pickImageTheme(im, design, interactive)
				if err != nil {
					return false, err
				}
				if !ok {
					break
				}

				ok, err = renderAndDeliverReleaseImage(cmd, repo, cfg, im, tags, version, shape, design, theme, interactive)
				if err != nil {
					return false, err
				}
				if ok {
					return true, nil
				}
			}
		}
	}

	version := im.version
	if version == "" {
		version = tags[0].Name // newest
	}

	shape, ok, err := pickImageShape(im, interactive)
	if err != nil || !ok {
		return false, err
	}
	design, ok, err := pickImageDesign(im, interactive)
	if err != nil || !ok {
		return false, err
	}
	theme, ok, err := pickImageTheme(im, design, interactive)
	if err != nil || !ok {
		return false, err
	}
	return renderAndDeliverReleaseImage(cmd, repo, cfg, im, tags, version, shape, design, theme, interactive)
}

func pickImageShape(im imageFlags, interactive bool) (card.Shape, bool, error) {
	shape, valid := card.ParseShape(im.shape)
	if valid {
		return shape, true, nil
	}
	if !interactive {
		return card.Horizontal, true, nil
	}
	name, ok, err := pick.Run(i18n.T(i18n.ImagePickShapeTitle), shapeItems())
	if err != nil || !ok {
		return card.Horizontal, ok, err
	}
	shape, _ = card.ParseShape(name)
	return shape, true, nil
}

func pickImageDesign(im imageFlags, interactive bool) (card.Design, bool, error) {
	design, valid := card.ParseDesign(im.design)
	if valid {
		return design, true, nil
	}
	if im.design != "" {
		return card.Default, false, fmt.Errorf("unknown design %q (default|minimal|bold|neon)", im.design)
	}
	if !interactive {
		return card.Default, true, nil
	}
	name, ok, err := pick.Run(i18n.T(i18n.ImagePickDesignTitle), designItems())
	if err != nil || !ok {
		return card.Default, ok, err
	}
	design, _ = card.ParseDesign(name)
	return design, true, nil
}

func pickImageTheme(im imageFlags, design card.Design, interactive bool) (string, bool, error) {
	theme := strings.ToLower(strings.TrimSpace(im.theme))
	if theme != "" {
		if card.ThemeForDesign(design, theme) {
			return theme, true, nil
		}
		return "", false, fmt.Errorf("unknown theme %q for design %q (%s)", im.theme, design, strings.Join(card.ThemesForDesign(design), "|"))
	}
	if !interactive {
		return card.DefaultTheme(design), true, nil
	}
	name, ok, err := pick.Run(i18n.T(i18n.ImagePickThemeTitle), themeItems(design))
	if err != nil || !ok {
		return "", ok, err
	}
	return name, true, nil
}

func renderAndDeliverReleaseImage(cmd *cobra.Command, repo *gitrepo.Repo, cfg config.Config, im imageFlags, tags []gitrepo.TagInfo, version string, shape card.Shape, design card.Design, theme string, interactive bool) (bool, error) {
	out := cmd.OutOrStdout()

	idx := -1
	for i, t := range tags {
		if t.Name == version {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false, fmt.Errorf(i18n.T(i18n.ImageNoSuchRelease), version)
	}

	from := ""
	if idx+1 < len(tags) {
		from = tags[idx+1].Name
	}
	raw, err := repo.CommitsBetween(from, version)
	if err != nil {
		return false, err
	}
	notes := changelog.Build(conventional.ParseMany(raw))

	c := card.Card{
		Project: titleCase(cfg.Project),
		Version: version,
		Meta:    imageMeta(tags[idx].DateTime, tags[idx].Date, len(raw)),
		Groups:  cardGroups(notes),
	}
	opt := card.Options{Design: design, Theme: theme, ShowHash: im.hash}

	img := card.Render(c, shape, opt)
	localPath := filepath.Join(".", imageFileName(version, shape, design, theme))
	destDir := "."
	if a, aerr := filepath.Abs(localPath); aerr == nil {
		destDir = filepath.Dir(a)
	}

	save, link, ask := imageDest(im, interactive)
	if ask {
		ans, ok, perr := pick.Run(i18n.T(i18n.ImagePickDestTitle), []pick.Item{
			{Label: i18n.T(i18n.ImageDestBothLabel), Desc: i18n.T(i18n.ImageDestBothDesc, destDir), Value: "both"},
			{Label: i18n.T(i18n.ImageDestSaveLabel), Desc: i18n.T(i18n.ImageDestSaveDesc, destDir), Value: "save"},
			{Label: i18n.T(i18n.ImageDestLinkLabel), Desc: i18n.T(i18n.ImageDestLinkDesc), Value: "link"},
		})
		if perr != nil || !ok {
			return false, perr
		}
		save, link = destFromAnswer(ans)
	}

	uploadSrc := ""
	if save {
		if err := card.Save(img, localPath); err != nil {
			return false, err
		}
		abs, _ := filepath.Abs(localPath)
		if _, err := fmt.Fprintln(out, ui.Success([]string{i18n.T(i18n.ImageSaved, localPath)})); err != nil {
			return false, err
		}
		if _, err := fmt.Fprintln(out, ui.Dim.Render("  "+abs)); err != nil {
			return false, err
		}
		uploadSrc = localPath
	}

	if link {
		if uploadSrc == "" {
			tmp, terr := os.CreateTemp("", "relio-*.png")
			if terr != nil {
				return false, terr
			}
			if err := tmp.Close(); err != nil {
				return false, err
			}
			defer func() { _ = os.Remove(tmp.Name()) }()
			if err := card.Save(img, tmp.Name()); err != nil {
				return false, err
			}
			uploadSrc = tmp.Name()
		}

		if save {
			if _, err := fmt.Fprintln(out); err != nil {
				return false, err
			}
		}
		url, uerr := upload.Upload(context.Background(), uploadSrc, out)
		if uerr != nil {
			if _, err := fmt.Fprintln(out, ui.Warn.Render("✗ ")+uerr.Error()); err != nil {
				return false, err
			}
			if save {
				if _, err := fmt.Fprintln(out, ui.Dim.Render(i18n.T(i18n.ImageStillSavedLocally))); err != nil {
					return false, err
				}
			}
			return true, nil
		}
		if err := printImageLink(out, url); err != nil {
			return false, err
		}
	}
	return true, nil
}

func printImageLink(out io.Writer, url string) error {
	// Print the URL as plain text before the QR. Mobile terminals and some
	// multiplexers do not always recognize ANSI-styled URLs as tappable links,
	// and the QR can push the actual URL out of view.
	if _, err := fmt.Fprintln(out, "  "+url); err != nil {
		return err
	}
	if _, err := fmt.Fprint(out, qrBlock(url)); err != nil {
		return err
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

func imageFileName(version string, shape card.Shape, design card.Design, theme string) string {
	if design == card.Default {
		return fmt.Sprintf("relio-%s-%s.png", version, shape)
	}
	return fmt.Sprintf("relio-%s-%s-%s-%s.png", version, shape, design, theme)
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
