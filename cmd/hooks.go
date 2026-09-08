package cmd

import (
	"io"
	"strings"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/hook"
	"github.com/soyagvs/relio/internal/release"
)

// runHooks executes the given hook commands from the repo root, exporting the
// RELIO_* environment derived from the plan (version, tag) and the tag the
// release was computed from (previousTag).
func runHooks(w io.Writer, repo *gitrepo.Repo, cfg config.Config, cmds config.StringList, plan release.Plan, previousTag string) error {
	return hook.Run(w, repo.Root(), cmds, hook.Env{
		Version:     strings.TrimPrefix(plan.Next.String(), "v"),
		Tag:         plan.TagName(),
		PreviousTag: previousTag,
	})
}
