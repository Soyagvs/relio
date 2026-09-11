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
	"github.com/soyagvs/relio/internal/menu"
	"github.com/soyagvs/relio/internal/pick"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/releases"
	"github.com/soyagvs/relio/internal/semver"
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
		Use:   "relio",
		Short: "Turn finished code into a published release",
		Long: ui.Title.Render("⬢ "+ui.AppName) + "\n\n" +
			"  Read the repo's git activity and turn it into a version, changelog,\n" +
			"  and tag — in one command, with a preview before anything is written.\n\n" +
			"  Run `relio` on its own for the interactive menu. Use the\n" +
			"  subcommands below for setup, extras, and scripting.",
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

	root.SetVersionTemplate(fmt.Sprintf("%s %s (commit %s, built %s)\n", ui.AppName, version, commit, date))

	pf := root.PersistentFlags()
	pf.StringVarP(&f.dir, "dir", "C", ".", "run as if relio was started in `path`")
	pf.BoolVar(&f.noHash, "no-hash", false, "hide commit hashes in release notes")

	lf := root.Flags()
	lf.BoolVar(&f.patch, "patch", false, "force a PATCH bump")
	lf.BoolVar(&f.minor, "minor", false, "force a MINOR bump")
	lf.BoolVar(&f.major, "major", false, "force a MAJOR bump")
	lf.BoolVarP(&f.yes, "yes", "y", false, "skip the interactive menu and confirmation")
	lf.BoolVar(&f.noChangelog, "no-changelog", false, "do not touch the changelog file")
	lf.BoolVar(&f.noTag, "no-tag", false, "do not create the git tag")
	lf.BoolVar(&f.noVersionFiles, "no-version-files", false, "do not update the files listed in version_files")
	lf.BoolVar(&f.publish, "publish", false, "push and create the GitHub Release after tagging")
	lf.BoolVar(&f.rc, "rc", false, "cut a release candidate (vX.Y.Z-rc.N) instead of the final version")
	lf.BoolVar(&f.noHooks, "no-hooks", false, "skip the before/after hooks in .release.yaml for this run")
	lf.BoolVar(&f.edit, "edit", false, "open the generated release notes in your editor before writing")

	root.AddCommand(newStatusCmd(f), newCheckCmd(f), newUndoCmd(f), newGuideCmd(f), newStatsCmd(), newInitCmd(f), newPostCmd(f), newImageCmd(f), newAuthCmd(), newVersionCmd())
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
		errors.Is(err, guide.ErrQuit)
}

// runMenu shows the animated entry banner together with the main menu. Every
// action returns to the menu when it finishes; Relio is left only via the Exit
// item, q/ctrl+c at the menu itself, or ctrl+c inside a sub-screen.
func runMenu(cmd *cobra.Command, f *releaseFlags) error {
	available := update.Available(version)
	out := cmd.OutOrStdout()

	for {
		action, err := menu.Run(version, available, stdoutIsTTY())
		if err != nil {
			return err
		}

		switch action {
		case menu.Exit, menu.None:
			return nil

		case menu.Release:
			if err := runMenuRelease(cmd, f); err != nil {
				if isHardQuit(err) {
					return nil
				}
				return err
			}

		case menu.Auth:
			if err := runMenuAuth(cmd); err != nil {
				if isHardQuit(err) {
					return nil
				}
				return err
			}

		case menu.Status:
			repo, cfg, oerr := openRepoAndConfig(f.dir)
			if oerr != nil {
				return oerr
			}
			if err := runStatus(cmd, repo, cfg); err != nil {
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

		case menu.ReleaseText:
			repo, cfg, oerr := openRepoAndConfig(f.dir)
			if oerr != nil {
				return oerr
			}
			if err := runReleaseText(cmd, repo, cfg); err != nil {
				return err
			}

		case menu.ReleaseImage:
			repo, cfg, oerr := openRepoAndConfig(f.dir)
			if oerr != nil {
				return oerr
			}
			if err := runReleaseImage(cmd, repo, cfg, imageFlags{}); err != nil {
				return err
			}

		case menu.Setup:
			if err := runMenuSetup(cmd, f); err != nil {
				return err
			}

		case menu.Guide:
			if err := runGuide(cmd, f); err != nil {
				if isHardQuit(err) {
					return nil
				}
				return err
			}

		case menu.Help:
			fmt.Fprintln(out, helpReference())
		}
		// loop: redraw the menu
	}
}

// runMenuRelease is the menu's Release entry: two quick picks (final vs rc, and
// whether to publish) that stand in for the --rc / --publish flags, then the
// normal interactive release. Backing out of a pick with q/esc returns nil so
// the menu is redrawn; a ctrl+c hard-quit propagates pick.ErrQuit.
func runMenuRelease(cmd *cobra.Command, f *releaseFlags) error {
	repo, cfg, err := openRepoAndConfig(f.dir)
	if err != nil {
		return err
	}

	relType, chosen, err := pick.Run("Release type", []pick.Item{
		{Label: "Final release", Desc: "the next stable version", Value: "final"},
		{Label: "Release candidate (rc.N)", Desc: "a pre-release you can iterate on, then finalize", Value: "rc"},
	})
	if err != nil {
		return err
	}
	if !chosen {
		return nil
	}

	pub, chosen, err := pick.Run("Publish to GitHub?", []pick.Item{
		{Label: "Just tag locally", Desc: "write the changelog, commit, and tag — nothing leaves your machine", Value: "local"},
		{Label: "Push and create the GitHub Release", Desc: "needs a GitHub token (GITHUB_TOKEN / GH_TOKEN or `gh auth login`)", Value: "publish"},
	})
	if err != nil {
		return err
	}
	if !chosen {
		return nil
	}

	ed, chosen, err := pick.Run("Edit the notes first?", []pick.Item{
		{Label: "Use the generated notes", Desc: "write the changelog straight from the commits", Value: "no"},
		{Label: "Edit them in my editor", Desc: "open $EDITOR on the generated notes before writing", Value: "yes"},
	})
	if err != nil {
		return err
	}
	if !chosen {
		return nil
	}

	f.rc = relType == "rc"
	f.publish = pub == "publish"
	f.edit = ed == "yes"
	return doRelease(cmd.OutOrStdout(), repo, cfg, f, semver.None, true)
}

// runMenuAuth is the menu's Auth entry: show the resolved sign-in status, or
// explain how to connect a token. Backing out of the pick with q/esc returns nil
// so the menu is redrawn; a ctrl+c hard-quit propagates pick.ErrQuit.
func runMenuAuth(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()

	choice, chosen, err := pick.Run("Auth", []pick.Item{
		{Label: "Show sign-in status", Desc: "which token relio found and who it belongs to", Value: "status"},
		{Label: "How to connect", Desc: "env vars or `gh auth login`", Value: "how"},
	})
	if err != nil {
		return err
	}
	if !chosen {
		return nil
	}

	if choice == "status" {
		return authStatus(out)
	}

	fmt.Fprintln(out, ui.Info("Relio reads a GitHub personal access token from RELIO_GITHUB_TOKEN, GITHUB_TOKEN"))
	fmt.Fprintln(out, ui.Info("or GH_TOKEN, and falls back to `gh auth token` when the GitHub CLI is signed in."))
	fmt.Fprintln(out, ui.Info("Run `gh auth login` (or set one of those vars) to connect one."))
	fmt.Fprintln(out, ui.Info("The token is only ever sent to GitHub in the Authorization header — Relio stores nothing."))
	return nil
}

// runMenuSetup is the menu's Setup entry: point at an existing .release.yaml, or
// run the `relio init` flow with a prompted project name.
func runMenuSetup(cmd *cobra.Command, f *releaseFlags) error {
	out := cmd.OutOrStdout()

	repo, err := gitrepo.Open(f.dir)
	if err != nil {
		fmt.Fprintln(out, ui.Info("not a git repository — run this inside a repo, or pass -C <path>"))
		return nil
	}
	root := repo.Root()

	if config.Exists(root) {
		fmt.Fprintln(out, ui.Info(config.Path(root)+" already exists"))
		fmt.Fprintln(out, ui.Dim.Render("  Edit it by hand; see the README for every field."))
		return nil
	}

	name := promptLine(cmd.InOrStdin(), out, "Project name", guessProjectName(repo, root))
	return runInit(out, repo, name)
}

// promptLine writes "<prompt> [<def>]: " and reads one line from r, returning def
// when the line is blank or unreadable.
func promptLine(r io.Reader, w io.Writer, prompt, def string) string {
	fmt.Fprintf(w, "%s [%s]: ", prompt, def)
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
		fmt.Fprintln(cmd.ErrOrStderr(), ui.Info("No commits since the last tag — nothing to announce."))
		return nil
	}

	format, chosen, err := pick.Run("Release text — pick a format", postFormats)
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
	fmt.Fprintln(cmd.ErrOrStderr(), ui.Dim.Render("# release text — copy from here:"))
	fmt.Fprintln(cmd.OutOrStdout(), text)
	return nil
}

// doRelease builds a plan, confirms it (wizard when interactive), and applies it.
func doRelease(out io.Writer, repo *gitrepo.Repo, cfg config.Config, f *releaseFlags, force semver.Bump, interactive bool) error {
	fmt.Fprintln(out, ui.Banner(cfg.Project, version))
	fmt.Fprintln(out)

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
		fmt.Fprintln(out, ui.Info(fmt.Sprintf("No commits since %s. Nothing to release.", base)))
		return nil
	}

	if interactive {
		// Print the preview to the scrollback first so it survives the wizard
		// clearing its own frame — the user can copy it afterwards.
		fmt.Fprintln(out, ui.PlanView(plan))
		fmt.Fprintln(out)

		res, werr := wizard.Run(plan)
		if werr != nil {
			return werr
		}
		if !res.Confirmed {
			fmt.Fprintln(out, ui.Info("Cancelled. Nothing was written."))
			return nil
		}
		if res.Bump != plan.Bump {
			plan, err = release.BuildPlan(repo, cfg, release.Options{ForceBump: res.Bump, Prerelease: f.rc})
			if err != nil {
				return err
			}
			applyFlagOverrides(&plan, f)
		}
	} else {
		fmt.Fprintln(out, ui.PlanView(plan))
		fmt.Fprintln(out)
		if !f.yes {
			return errors.New("refusing to modify the repo without confirmation — re-run with --yes")
		}
		fmt.Fprintln(out, ui.Info("Proceeding (--yes)."))
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
			fmt.Fprintln(out, ui.Info("Edited notes were empty — keeping the generated notes."))
		case strings.TrimSpace(edited) == strings.TrimSpace(plan.EditableNotes()):
			fmt.Fprintln(out, ui.Info("Notes unchanged."))
		default:
			plan.NotesOverride = strings.TrimSpace(edited)
			fmt.Fprintln(out, ui.Info("Using your edited release notes."))
		}
	}

	if !f.noHooks && len(cfg.Release.Hooks.Before) > 0 {
		fmt.Fprintln(out)
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

	fmt.Fprintln(out)
	fmt.Fprintln(out, ui.Success(done))

	if plan.Prerelease {
		fmt.Fprintln(out, ui.Dim.Render(fmt.Sprintf(
			"  this is a pre-release — run `relio` (no --rc) when you're ready to finalize %s",
			plan.Next.Core().String())))
	}
	if plan.Finalizing {
		fmt.Fprintln(out, ui.Dim.Render("  finalized from "+plan.Current.String()))
	}

	if interactive && applied.ChangelogPath != "" {
		if tip := footerTip(cfg); tip != "" {
			fmt.Fprintln(out, tip)
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
		fmt.Fprintln(out)
		fmt.Fprintln(out, ui.Dim.Render("  next:  git push && git push origin "+applied.TagName))
	}

	if !f.noHooks && applied.TagName != "" && len(cfg.Release.Hooks.After) > 0 {
		fmt.Fprintln(out)
		if err := runHooks(out, repo, cfg, cfg.Release.Hooks.After, plan, prev); err != nil {
			fmt.Fprintln(out, ui.Warn.Render("! ")+ui.Dim.Render(fmt.Sprintf("after hook failed: %v (the release itself is done)", err)))
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
