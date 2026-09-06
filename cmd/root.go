// Package cmd wires the CLI: a styled command shell (Cobra + Lipgloss) with an
// interactive Bubble Tea main menu and a confirmation wizard for releases.
package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/menu"
	"github.com/soyagvs/relio/internal/pick"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/releases"
	"github.com/soyagvs/relio/internal/semver"
	"github.com/soyagvs/relio/internal/ui"
	"github.com/soyagvs/relio/internal/wizard"
)

// Build metadata, overridable with -ldflags "-X .../cmd.version=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type releaseFlags struct {
	dir         string
	patch       bool
	minor       bool
	major       bool
	yes         bool
	noChangelog bool
	noTag       bool
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoot(cmd, f)
		},
	}

	root.SetVersionTemplate(fmt.Sprintf("%s %s (commit %s, built %s)\n", ui.AppName, version, commit, date))

	pf := root.PersistentFlags()
	pf.StringVarP(&f.dir, "dir", "C", ".", "run as if relio was started in `path`")

	lf := root.Flags()
	lf.BoolVar(&f.patch, "patch", false, "force a PATCH bump")
	lf.BoolVar(&f.minor, "minor", false, "force a MINOR bump")
	lf.BoolVar(&f.major, "major", false, "force a MAJOR bump")
	lf.BoolVarP(&f.yes, "yes", "y", false, "skip the interactive menu and confirmation")
	lf.BoolVar(&f.noChangelog, "no-changelog", false, "do not touch the changelog file")
	lf.BoolVar(&f.noTag, "no-tag", false, "do not create the git tag")

	root.AddCommand(newInitCmd(f), newPostCmd(f), newAuthCmd(), newVersionCmd())
	return root
}

// Execute runs the CLI and returns the process exit code.
func Execute() int {
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

// runMenu shows the main menu once, runs the chosen action, and returns. It does
// not loop — the action's output stays on screen as the last thing printed.
func runMenu(cmd *cobra.Command, f *releaseFlags) error {
	out := cmd.OutOrStdout()

	action, err := menu.Run(version)
	if err != nil {
		return err
	}

	switch action {
	case menu.Exit, menu.None:
		return nil

	case menu.Help:
		fmt.Fprintln(out, helpReference())
		return nil

	case menu.GitHubAuth:
		fmt.Fprintln(out, ui.Banner("", version))
		fmt.Fprintln(out)
		fmt.Fprintln(out, ui.Info("GitHub auth lands in v0.2.0: per-user OAuth Device Flow,"))
		fmt.Fprintln(out, ui.Info("tokens stored in the OS keychain — never in .release.yaml."))
		return nil

	case menu.CreateRelease:
		repo, cfg, oerr := openRepoAndConfig(f.dir)
		if oerr != nil {
			return oerr
		}
		return doRelease(out, repo, cfg, f, semver.None, true)

	case menu.ViewReleases:
		repo, cfg, oerr := openRepoAndConfig(f.dir)
		if oerr != nil {
			return oerr
		}
		return releases.Run(repo, cfg)

	case menu.ReleaseText:
		repo, cfg, oerr := openRepoAndConfig(f.dir)
		if oerr != nil {
			return oerr
		}
		return runReleaseText(cmd, repo, cfg)
	}
	return nil
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

	plan, err := release.BuildPlan(repo, cfg, release.Options{ForceBump: force})
	if err != nil {
		return err
	}
	applyFlagOverrides(&plan, f)

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
			plan, err = release.BuildPlan(repo, cfg, release.Options{ForceBump: res.Bump})
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

	applied, err := plan.Apply(repo)
	if err != nil {
		return err
	}

	var done []string
	if applied.ChangelogPath != "" {
		done = append(done, fmt.Sprintf("%s updated", cfg.Release.ChangelogFile))
	}
	if applied.TagName != "" {
		done = append(done, fmt.Sprintf("git tag %s created", applied.TagName))
	}
	done = append(done, "release ready")

	fmt.Fprintln(out)
	fmt.Fprintln(out, ui.Success(done))
	if applied.TagName != "" {
		fmt.Fprintln(out)
		fmt.Fprintln(out, ui.Dim.Render("  next:  git push && git push origin "+applied.TagName))
	}
	return nil
}

func applyFlagOverrides(p *release.Plan, f *releaseFlags) {
	if f.noChangelog {
		p.ChangelogUpdate = false
	}
	if f.noTag {
		p.TagUpdate = false
	}
}
