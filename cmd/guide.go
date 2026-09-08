package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/ghrelease"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/guide"
	"github.com/soyagvs/relio/internal/release"
)

func newGuideCmd(f *releaseFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "guide",
		Short: "Walk through the whole release flow step by step",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGuide(cmd, f)
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
