// Package cmd wires the CLI: a styled command shell (Cobra + Lipgloss) with an
// interactive Bubble Tea main menu and a confirmation wizard for releases.
package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/editor"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/guide"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/menu"
	"github.com/soyagvs/relio/internal/pick"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/releases"
	"github.com/soyagvs/relio/internal/semver"
	"github.com/soyagvs/relio/internal/settings"
	"github.com/soyagvs/relio/internal/ui"
	"github.com/soyagvs/relio/internal/update"
	"github.com/soyagvs/relio/internal/wizard"
)

// Build metadata, overridable with -ldflags "-X .../cmd.version=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type releaseFlags struct {
	dir            string
	patch          bool
	minor          bool
	major          bool
	yes            bool
	noChangelog    bool
	noTag          bool
	noHash         bool
	noVersionFiles bool
	publish        bool
	rc             bool
	noHooks        bool
	edit           bool
}

// NewRootCmd builds the root command. Running it with no subcommand opens the
// interactive menu (TTY) or performs a release directly (CI / --yes / forced bump).
func NewRootCmd() *cobra.Command {
	f := &releaseFlags{}

	root := &cobra.Command{
		Use:           "relio",
		Short:         i18n.T(i18n.RootShort),
		Long:          ui.Title.Render("⬢ "+ui.AppName) + "\n\n" + i18n.T(i18n.RootLong),
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ui.HideHashes = f.noHash
			// Idempotent re-resolution: Execute() already resolved the
			// language before the tree was built (so Short/Long/flag usage
			// text render correctly), but tests that construct the tree
			// directly via NewRootCmd() skip that step. f.dir reflects the
			// parsed --dir/-C flag by the time this runs.
			resolveLanguage(f.dir, cmd.ErrOrStderr())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoot(cmd, f)
		},
	}

	root.SetVersionTemplate(fmt.Sprintf(i18n.T(i18n.VersionInfoLine), ui.AppName, version, commit, date))

	pf := root.PersistentFlags()
	pf.StringVarP(&f.dir, "dir", "C", ".", i18n.T(i18n.FlagDirUsage))
	pf.BoolVar(&f.noHash, "no-hash", false, i18n.T(i18n.FlagNoHashUsage))

	lf := root.Flags()
	lf.BoolVar(&f.patch, "patch", false, i18n.T(i18n.FlagPatchUsage))
	lf.BoolVar(&f.minor, "minor", false, i18n.T(i18n.FlagMinorUsage))
	lf.BoolVar(&f.major, "major", false, i18n.T(i18n.FlagMajorUsage))
	lf.BoolVarP(&f.yes, "yes", "y", false, i18n.T(i18n.FlagYesUsage))
	lf.BoolVar(&f.noChangelog, "no-changelog", false, i18n.T(i18n.FlagNoChangelogUsage))
	lf.BoolVar(&f.noTag, "no-tag", false, i18n.T(i18n.FlagNoTagUsage))
	lf.BoolVar(&f.noVersionFiles, "no-version-files", false, i18n.T(i18n.FlagNoVersionFilesUsage))
	lf.BoolVar(&f.publish, "publish", false, i18n.T(i18n.FlagPublishUsage))
	lf.BoolVar(&f.rc, "rc", false, i18n.T(i18n.FlagRCUsage))
	lf.BoolVar(&f.noHooks, "no-hooks", false, i18n.T(i18n.FlagNoHooksUsage))
	lf.BoolVar(&f.edit, "edit", false, i18n.T(i18n.FlagEditUsage))

	root.AddCommand(newStatusCmd(f), newCheckCmd(f), newUndoCmd(f), newGuideCmd(f), newStatsCmd(), newInitCmd(f), newPostCmd(f), newImageCmd(f), newAuthCmd(), newVersionCmd(), newReleasesCmd(f))
	return root
}

// Execute runs the CLI and returns the process exit code. The language is
// resolved from os.Args BEFORE NewRootCmd() builds the command tree: cobra
// evaluates Short/Long and flag usage strings at construction time, and
// --help short-circuits before PersistentPreRun ever fires, so resolving
// there would leave --help permanently English.
func Execute() int {
	resolveLanguage(prescanDir(os.Args[1:]), os.Stderr)
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, ui.Warn.Render("✗ ")+err.Error())
		return 1
	}
	return 0
}

func forcedBump(f *releaseFlags) (semver.Bump, error) {
	n := 0
	b := semver.None
	if f.patch {
		n, b = n+1, semver.Patch
	}
	if f.minor {
		n, b = n+1, semver.Minor
	}
	if f.major {
		n, b = n+1, semver.Major
	}
	if n > 1 {
		return semver.None, errors.New("choose only one of --patch, --minor, --major")
	}
	return b, nil
}

func openRepoAndConfig(dir string) (*gitrepo.Repo, config.Config, error) {
	repo, err := gitrepo.Open(dir)
	if errors.Is(err, gitrepo.ErrNotARepo) {
		return nil, config.Config{}, errors.New("not a git repository — run this inside a repo, or pass -C <path>")
	}
	if err != nil {
		return nil, config.Config{}, err
	}
	cfg, err := config.Load(repo.Root())
	if errors.Is(err, config.ErrNotFound) {
		return nil, config.Config{}, errors.New("no .release.yaml found — run `relio init` first")
	}
	if err != nil {
		return nil, config.Config{}, err
	}
	return repo, cfg, nil
}

func stdinIsTTY() bool  { return isatty.IsTerminal(os.Stdin.Fd()) }
func stdoutIsTTY() bool { return isatty.IsTerminal(os.Stdout.Fd()) }

var runPicker = pick.Run

// runRoot decides between the interactive menu and a direct release.
func runRoot(cmd *cobra.Command, f *releaseFlags) error {
	out := cmd.OutOrStdout()

	force, err := forcedBump(f)
	if err != nil {
		return err
	}

	interactive := stdinIsTTY() && stdoutIsTTY()
	useMenu := interactive && !f.yes && force == semver.None

	if !useMenu {
		repo, cfg, oerr := openRepoAndConfig(f.dir)
		if oerr != nil {
			return oerr
		}
		return doRelease(out, repo, cfg, f, force, interactive)
	}

	return runMenu(cmd, f)
}

// isHardQuit reports whether err is a ctrl+c from an interactive sub-screen,
// which should leave Relio rather than fall back to the menu.
func isHardQuit(err error) bool {
	return errors.Is(err, pick.ErrQuit) ||
		errors.Is(err, releases.ErrQuit) ||
		errors.Is(err, guide.ErrQuit) ||
		errors.Is(err, settings.ErrQuit)
}

// runMenu shows the animated entry banner together with the main menu. Every
// action returns to the menu when it finishes; Relio is left only via the Exit
// item, q/ctrl+c at the menu itself, or ctrl+c inside a sub-screen.
func runMenu(cmd *cobra.Command, f *releaseFlags) error {
	available := update.Available(version)
	out := cmd.OutOrStdout()

	for {
		waitForBack := false
		action, err := menu.Run(version, available, stdoutIsTTY())
		if err != nil {
			return err
		}

		switch action {
		case menu.Exit, menu.None:
			return nil

		case menu.Release:
			waitForBack, err = runMenuRelease(cmd, f)
			if err != nil {
				if isHardQuit(err) {
					return nil
				}
				return err
			}

		case menu.Auth:
			waitForBack, err = runMenuAuth(cmd)
			if err != nil {
				if isHardQuit(err) {
					return nil
				}
				return err
			}

		case menu.Status:
			waitForBack, err = runMenuStatus(cmd, f)
			if err != nil {
				if isHardQuit(err) {
					return nil
				}
				return err
			}

		case menu.Check:
			repo, cfg, oerr := openRepoAndConfig(f.dir)
			if oerr != nil {
				return oerr
			}
			plan, perr := release.BuildPlan(repo, cfg, release.Options{})
			if perr != nil {
				return perr
			}
			if err := runCheck(cmd, repo, cfg, plan, false); err != nil {
				return err
			}
			waitForBack = true

		case menu.ViewReleases:
			repo, cfg, oerr := openRepoAndConfig(f.dir)
			if oerr != nil {
				return oerr
			}
			if err := releases.Run(repo, cfg); err != nil {
				if isHardQuit(err) {
					return nil
				}
				return err
			}
			waitForBack = true

		case menu.ReleaseText:
			repo, cfg, oerr := openRepoAndConfig(f.dir)
			if oerr != nil {
				return oerr
			}
			if err := runReleaseText(cmd, repo, cfg); err != nil {
				return err
			}
			waitForBack = true

		case menu.ReleaseImage:
			repo, cfg, oerr := openRepoAndConfig(f.dir)
			if oerr != nil {
				return oerr
			}
			if err := runReleaseImage(cmd, repo, cfg, imageFlags{}); err != nil {
				return err
			}
			waitForBack = true

		case menu.Setup:
			if err := runMenuSetup(cmd, f); err != nil {
				return err
			}
			waitForBack = true

		case menu.Settings:
			root := ""
			if repo, err := gitrepo.Open(f.dir); err == nil {
				root = repo.Root()
			}
			if err := settings.Run(root); err != nil {
				if isHardQuit(err) {
					return nil
				}
				return err
			}
			waitForBack = true

		case menu.Guide:
			if err := runGuide(cmd, f); err != nil {
				if isHardQuit(err) {
					return nil
				}
				return err
			}
			waitForBack = true

		case menu.Help:
			if _, err := fmt.Fprintln(out, helpReference()); err != nil {
				return err
			}
			waitForBack = true
		}

		if waitForBack {
			if err := waitMenuBack(cmd); err != nil {
				if isHardQuit(err) {
					return nil
				}
				return err
			}
		}
		// loop: redraw the menu
	}
}

// waitMenuBack waits for an explicit back selection so the user can read an
// action's result before the main menu is redrawn.
func waitMenuBack(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	_, _, err := runPicker("", nil)
	if err != nil {
		return err
	}
	return clearScreen(out)
}

func clearScreen(out io.Writer) error {
	_, err := fmt.Fprint(out, "\x1b[2J\x1b[H")
	return err
}

// runMenuStatus prints the status result. The menu loop handles the shared
// bottom back selector after successful menu actions.
func runMenuStatus(cmd *cobra.Command, f *releaseFlags) (bool, error) {
	repo, cfg, err := openRepoAndConfig(f.dir)
	if err != nil {
		return false, err
	}
	if err := runStatus(cmd, repo, cfg); err != nil {
		return false, err
	}
	return true, nil
}

// runMenuRelease is the menu's Release entry: three quick picks (final vs rc,
// whether to publish, and whether to edit notes) that stand in for the matching
// flags, then the normal interactive release. Backing out moves one step back;
// backing out of the first pick redraws the menu. A ctrl+c hard-quit propagates
// pick.ErrQuit.
func runMenuRelease(cmd *cobra.Command, f *releaseFlags) (bool, error) {
	repo, cfg, err := openRepoAndConfig(f.dir)
	if err != nil {
		return false, err
	}

	step := 0
	var relType, pub, ed string
	for step < 3 {
		var chosen bool
		switch step {
		case 0:
			relType, chosen, err = runPicker("Release type", []pick.Item{
				{Label: "Final release", Desc: "the next stable version", Value: "final"},
				{Label: "Release candidate (rc.N)", Desc: "a pre-release you can iterate on, then finalize", Value: "rc"},
			})
		case 1:
			pub, chosen, err = runPicker("Publish to GitHub?", []pick.Item{
				{Label: "Just tag locally", Desc: "write the changelog, commit, and tag — nothing leaves your machine", Value: "local"},
				{Label: "Push and create the GitHub Release", Desc: "needs a GitHub token (GITHUB_TOKEN / GH_TOKEN or `gh auth login`)", Value: "publish"},
			})
		case 2:
			ed, chosen, err = runPicker("Edit the notes first?", []pick.Item{
				{Label: "Use the generated notes", Desc: "write the changelog straight from the commits", Value: "no"},
				{Label: "Edit them in my editor", Desc: "open $EDITOR on the generated notes before writing", Value: "yes"},
			})
		}
		if err != nil {
			return false, err
		}
		if !chosen {
			if step == 0 {
				return false, nil
			}
			step--
			continue
		}
		step++
	}

	f.rc = relType == "rc"
	f.publish = pub == "publish"
	f.edit = ed == "yes"
	return true, doRelease(cmd.OutOrStdout(), repo, cfg, f, semver.None, true)
}

// runMenuAuth is the menu's Auth entry: show the resolved sign-in status, or
// explain how to connect a token. Backing out of the pick with q/esc returns nil
// so the menu is redrawn; a ctrl+c hard-quit propagates pick.ErrQuit.
func runMenuAuth(cmd *cobra.Command) (bool, error) {
	out := cmd.OutOrStdout()

	choice, chosen, err := runPicker("Auth", []pick.Item{
		{Label: "Show sign-in status", Desc: "which token relio found and who it belongs to", Value: "status"},
		{Label: "How to connect", Desc: "env vars or `gh auth login`", Value: "how"},
	})
	if err != nil {
		return false, err
	}
	if !chosen {
		return false, nil
	}

	if choice == "status" {
		return true, authStatus(out)
	}

	if _, err := fmt.Fprintln(out, ui.Info("Relio reads a GitHub personal access token from RELIO_GITHUB_TOKEN, GITHUB_TOKEN")); err != nil {
		return false, err
	}
	if _, err := fmt.Fprintln(out, ui.Info("or GH_TOKEN, and falls back to `gh auth token` when the GitHub CLI is signed in.")); err != nil {
		return false, err
	}
	if _, err := fmt.Fprintln(out, ui.Info("Run `gh auth login` (or set one of those vars) to connect one.")); err != nil {
		return false, err
	}
	if _, err := fmt.Fprintln(out, ui.Info("The token is only ever sent to GitHub in the Authorization header — Relio stores nothing.")); err != nil {
		return false, err
	}
	return true, nil
}

// runMenuSetup is the menu's Setup entry: point at an existing .release.yaml, or
// run the `relio init` flow with a prompted project name.
func runMenuSetup(cmd *cobra.Command, f *releaseFlags) error {
	out := cmd.OutOrStdout()

	repo, err := gitrepo.Open(f.dir)
	if err != nil {
		_, err := fmt.Fprintln(out, ui.Info("not a git repository — run this inside a repo, or pass -C <path>"))
		return err
	}
	root := repo.Root()

	if config.Exists(root) {
		if _, err := fmt.Fprintln(out, ui.Info(config.Path(root)+" already exists")); err != nil {
			return err
		}
		_, err := fmt.Fprintln(out, ui.Dim.Render("  Edit it by hand; see the README for every field."))
		return err
	}

	name := promptLine(cmd.InOrStdin(), out, "Project name", guessProjectName(repo, root))
	return runInit(out, repo, name)
}

// promptLine writes "<prompt> [<def>]: " and reads one line from r, returning def
// when the line is blank or unreadable.
func promptLine(r io.Reader, w io.Writer, prompt, def string) string {
	_, _ = fmt.Fprintf(w, "%s [%s]: ", prompt, def)
	sc := bufio.NewScanner(r)
	if sc.Scan() {
		if v := strings.TrimSpace(sc.Text()); v != "" {
			return v
		}
	}
	return def
}

// runReleaseText asks for a post format, then prints the text (stdout only).
func runReleaseText(cmd *cobra.Command, repo *gitrepo.Repo, cfg config.Config) error {
	plan, err := release.BuildPlan(repo, cfg, release.Options{})
	if err != nil {
		return err
	}
	if plan.NothingToRelease() {
		_, err := fmt.Fprintln(cmd.ErrOrStderr(), ui.Info("No commits since the last tag — nothing to announce."))
		return err
	}

	format, chosen, err := pick.Run("Release text — pick a format", postFormatItems())
	if err != nil {
		return err
	}
	if !chosen {
		return nil
	}

	text, err := renderPost(cfg.Project, plan, format)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(cmd.ErrOrStderr(), ui.Dim.Render("# release text — copy from here:")); err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), text)
	return err
}

// doRelease builds a plan, confirms it (wizard when interactive), and applies it.
func doRelease(out io.Writer, repo *gitrepo.Repo, cfg config.Config, f *releaseFlags, force semver.Bump, interactive bool) error {
	writeLine := func(a ...any) error {
		_, err := fmt.Fprintln(out, a...)
		return err
	}

	if err := writeLine(ui.Banner(cfg.Project, version)); err != nil {
		return err
	}
	if err := writeLine(); err != nil {
		return err
	}

	plan, err := release.BuildPlan(repo, cfg, release.Options{ForceBump: force, Prerelease: f.rc})
	if err != nil {
		return err
	}
	applyFlagOverrides(&plan, f)

	// The tag this release is computed from, captured before Apply moves state —
	// exported to hooks as RELIO_PREVIOUS_TAG.
	prev, _, _ := repo.LatestTag()

	if plan.NothingToRelease() {
		base := "the last tag"
		if plan.Current.String() != "v0.0.0" {
			base = plan.Current.String()
		}
		return writeLine(ui.Info(fmt.Sprintf("No commits since %s. Nothing to release.", base)))
	}

	if !f.noHooks && len(cfg.Release.Hooks.Validate) > 0 {
		if err := writeLine(); err != nil {
			return err
		}
		if err := runHooks(out, repo, cfg, cfg.Release.Hooks.Validate, plan, prev); err != nil {
			return fmt.Errorf("validate hook failed — nothing was written: %w", err)
		}
	}

	if interactive {
		// Print the preview to the scrollback first so it survives the wizard
		// clearing its own frame — the user can copy it afterwards.
		if err := writeLine(ui.PlanView(plan)); err != nil {
			return err
		}
		if err := writeLine(); err != nil {
			return err
		}

		res, werr := wizard.Run(plan)
		if werr != nil {
			return werr
		}
		if !res.Confirmed {
			return writeLine(ui.Info("Cancelled. Nothing was written."))
		}
		if res.Bump != plan.Bump {
			plan, err = release.BuildPlan(repo, cfg, release.Options{ForceBump: res.Bump, Prerelease: f.rc})
			if err != nil {
				return err
			}
			applyFlagOverrides(&plan, f)
		}
	} else {
		if err := writeLine(ui.PlanView(plan)); err != nil {
			return err
		}
		if err := writeLine(); err != nil {
			return err
		}
		if !f.yes {
			return errors.New("refusing to modify the repo without confirmation — re-run with --yes")
		}
		if err := writeLine(ui.Info("Proceeding (--yes).")); err != nil {
			return err
		}
	}

	if f.edit {
		if !interactive {
			return errors.New("--edit needs an interactive terminal")
		}
		edited, eerr := editor.Edit(plan.EditableNotes())
		if eerr != nil {
			return fmt.Errorf("editing release notes: %w", eerr)
		}
		switch {
		case strings.TrimSpace(edited) == "":
			if err := writeLine(ui.Info("Edited notes were empty — keeping the generated notes.")); err != nil {
				return err
			}
		case strings.TrimSpace(edited) == strings.TrimSpace(plan.EditableNotes()):
			if err := writeLine(ui.Info("Notes unchanged.")); err != nil {
				return err
			}
		default:
			plan.NotesOverride = strings.TrimSpace(edited)
			if err := writeLine(ui.Info("Using your edited release notes.")); err != nil {
				return err
			}
		}
	}

	if !f.noHooks && len(cfg.Release.Hooks.Before) > 0 {
		if err := writeLine(); err != nil {
			return err
		}
		if err := runHooks(out, repo, cfg, cfg.Release.Hooks.Before, plan, prev); err != nil {
			return fmt.Errorf("before hook failed — nothing was written: %w", err)
		}
	}

	applied, err := plan.Apply(repo)
	if err != nil {
		return err
	}

	var done []string
	if applied.ChangelogPath != "" {
		done = append(done, fmt.Sprintf("%s updated", cfg.Release.ChangelogFile))
	}
	if len(applied.VersionFiles) > 0 {
		done = append(done, fmt.Sprintf("%d version file(s) updated", len(applied.VersionFiles)))
	}
	if applied.Committed {
		done = append(done, fmt.Sprintf("%s committed", cfg.Release.ChangelogFile))
	}
	if applied.TagName != "" {
		done = append(done, fmt.Sprintf("git tag %s created", applied.TagName))
	}
	done = append(done, "release ready")

	if err := writeLine(); err != nil {
		return err
	}
	if err := writeLine(ui.Success(done)); err != nil {
		return err
	}

	if plan.Prerelease {
		if err := writeLine(ui.Dim.Render(fmt.Sprintf(
			"  this is a pre-release — run `relio` (no --rc) when you're ready to finalize %s",
			plan.Next.Core().String()))); err != nil {
			return err
		}
	}
	if plan.Finalizing {
		if err := writeLine(ui.Dim.Render("  finalized from " + plan.Current.String())); err != nil {
			return err
		}
	}

	if interactive && applied.ChangelogPath != "" {
		if tip := footerTip(cfg); tip != "" {
			if err := writeLine(tip); err != nil {
				return err
			}
		}
	}

	printedNext := false
	if plan.PublishGitHub && applied.TagName != "" {
		printedNext, err = publishGitHubRelease(out, repo, cfg, plan, applied, interactive, f.yes)
		if err != nil {
			return err
		}
	}

	if applied.TagName != "" && !printedNext {
		if err := writeLine(); err != nil {
			return err
		}
		if err := writeLine(ui.Dim.Render("  next:  git push && git push origin " + applied.TagName)); err != nil {
			return err
		}
	}

	if !f.noHooks && applied.TagName != "" && len(cfg.Release.Hooks.After) > 0 {
		if err := writeLine(); err != nil {
			return err
		}
		if err := runHooks(out, repo, cfg, cfg.Release.Hooks.After, plan, prev); err != nil {
			if err := writeLine(ui.Warn.Render("! ") + ui.Dim.Render(fmt.Sprintf("after hook failed: %v (the release itself is done)", err))); err != nil {
				return err
			}
		}
	}
	return nil
}

// footerTip is the one-line, dimmed reminder printed after an interactive
// release while both changelog-footer toggles are off, so the feature stays
// discoverable. It returns "" once either toggle is enabled.
func footerTip(cfg config.Config) string {
	if cfg.Release.Contributors || cfg.Release.CompareLink {
		return ""
	}
	return ui.Dim.Render("  tip: set release.contributors / release.compare_link in " +
		config.FileName + " to add a credits line and a compare link to the notes")
}

func applyFlagOverrides(p *release.Plan, f *releaseFlags) {
	if f.noChangelog {
		p.ChangelogUpdate = false
	}
	if f.noTag {
		p.TagUpdate = false
	}
	if f.noVersionFiles {
		p.VersionFilesUpdate = false
	}
	if f.publish {
		p.PublishGitHub = true
	}
}
