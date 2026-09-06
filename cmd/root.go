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

	"github.com/soyagvs/go-release/internal/config"
	"github.com/soyagvs/go-release/internal/gitrepo"
	"github.com/soyagvs/go-release/internal/menu"
	"github.com/soyagvs/go-release/internal/release"
	"github.com/soyagvs/go-release/internal/semver"
	"github.com/soyagvs/go-release/internal/ui"
	"github.com/soyagvs/go-release/internal/wizard"
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
		Use:   "go-release",
		Short: "Turn finished code into a published release",
		Long: ui.Title.Render("⬢ "+ui.AppName) + "\n\n" +
			"  Read the repo's git activity and turn it into a version, changelog,\n" +
			"  and tag — in one command, with a preview before anything is written.\n\n" +
			"  Run `go-release` on its own for the interactive menu. Use the\n" +
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
	pf.StringVarP(&f.dir, "dir", "C", ".", "run as if go-release was started in `path`")

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
		return nil, config.Config{}, errors.New("no .release.yaml found — run `go-release init` first")
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

	return runMenu(out, f)
}

// runMenu loops the main menu until the user chooses Exit.
func runMenu(out io.Writer, f *releaseFlags) error {
	for {
		action, err := menu.Run(version)
		if err != nil {
			return err
		}

		switch action {
		case menu.Exit, menu.None:
			fmt.Fprintln(out, ui.Info("See you next release."))
			return nil

		case menu.GitHubAuth:
			fmt.Fprintln(out, ui.Banner("", version))
			fmt.Fprintln(out)
			fmt.Fprintln(out, ui.Info("GitHub auth lands in v0.2.0: per-user OAuth Device Flow,"))
			fmt.Fprintln(out, ui.Info("tokens stored in the OS keychain — never in .release.yaml."))
			fmt.Fprintln(out)

		case menu.CreateRelease:
			repo, cfg, oerr := openRepoAndConfig(f.dir)
			if oerr != nil {
				fmt.Fprintln(out, ui.Warn.Render("✗ ")+oerr.Error())
				fmt.Fprintln(out)
				continue
			}
			if derr := doRelease(out, repo, cfg, f, semver.None, true); derr != nil {
				fmt.Fprintln(out, ui.Warn.Render("✗ ")+derr.Error())
			}
			fmt.Fprintln(out)
		}
	}
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
