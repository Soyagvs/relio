package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/ghrelease"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/guide"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/release"
)

// newGuideCmd's Short reuses cmd/help.go's HelpCmdGuideDesc key — the two
// strings are byte-identical, so this avoids declaring a duplicate key
// (same convention as cmd/post.go's postFormatItems() reusing HelpPost*Desc).
func newGuideCmd(f *releaseFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "guide",
		Short: i18n.T(i18n.HelpCmdGuideDesc),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runGuide(cmd, f); err != nil && !errors.Is(err, guide.ErrQuit) {
				return err
			}
			return nil
		},
	}
}

// runGuide renders the walkthrough — interactive when stdin/stdout are a TTY,
// plain text otherwise. A non-repo is fine: the guide still explains the flow.
func runGuide(cmd *cobra.Command, f *releaseFlags) error {
	repo, rerr := gitrepo.Open(f.dir)
	hasRepo := rerr == nil

	var (
		cfg       config.Config
		hasConfig bool
	)
	if hasRepo {
		c, cerr := config.Load(repo.Root())
		if cerr != nil && !errors.Is(cerr, config.ErrNotFound) {
			return cerr
		}
		if cerr == nil {
			cfg, hasConfig = c, true
		}
	}

	tok, _ := ghrelease.Token()

	ctx := guide.Context{
		HasRepo:   hasRepo,
		HasConfig: hasConfig,
		HasToken:  tok != "",
	}
	if hasConfig {
		ctx.Project = cfg.Project
	}

	// "Run it now" actions: only wired when they can actually work.
	if hasRepo && !hasConfig {
		ctx.RunInit = func() error { return runInit(cmd.OutOrStdout(), repo, "") }
	}
	if hasRepo && hasConfig {
		ctx.RunCheck = func() error {
			plan, err := release.BuildPlan(repo, cfg, release.Options{})
			if err != nil {
				return err
			}
			return runCheck(cmd, repo, cfg, plan, false)
		}
		ctx.RunStatus = func() error { return runStatus(cmd, repo, cfg) }
	}

	if stdinIsTTY() && stdoutIsTTY() {
		return guide.Run(ctx)
	}
	return guide.PrintPlain(cmd.OutOrStdout(), ctx)
}
