package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/ghrelease"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/ui"
)

// publishGitHubRelease pushes the current branch and the new tag to origin, then
// creates the GitHub Release with the changelog notes as its body. The local
// release has already succeeded by the time this runs, so a missing token or a
// declined prompt prints a hint and returns without failing the command. A push
// or API failure is returned wrapped, with the message making clear that the
// local tag is intact.
//
// printedNext reports whether this step already told the user how to finish
// (so doRelease should not also print its "next: git push" line).
func publishGitHubRelease(out io.Writer, repo *gitrepo.Repo, cfg config.Config, plan release.Plan, applied release.ApplyResult, interactive, yes bool) (printedNext bool, err error) {
	writeLine := func(a ...any) error {
		_, err := fmt.Fprintln(out, a...)
		return err
	}

	token, _ := ghrelease.Token()
	if token == "" {
		if err := writeLine(); err != nil {
			return false, err
		}
		if err := writeLine(ui.Info(i18n.T(i18n.PublishNoTokenSkip))); err != nil {
			return false, err
		}
		if err := writeLine(ui.Dim.Render(i18n.T(i18n.PublishNoTokenHint))); err != nil {
			return false, err
		}
		// A literal command the user types verbatim, not prose — stays untranslated.
		if err := writeLine(ui.Dim.Render("  git push && git push origin " + applied.TagName)); err != nil {
			return false, err
		}
		return true, nil
	}

	ownerName := cfg.GitHub.Repo
	if ownerName == "" {
		remote, rerr := repo.RemoteURL("origin")
		if rerr != nil {
			return false, fmt.Errorf(i18n.T(i18n.PublishNoOriginRemote), applied.TagName, rerr)
		}
		ownerName, rerr = ghrelease.ParseRepo(remote)
		if rerr != nil {
			return false, fmt.Errorf(i18n.T(i18n.PublishUnknownRepo), applied.TagName, config.FileName, rerr)
		}
	}

	branch, berr := repo.CurrentBranch()
	if berr != nil {
		return false, berr
	}

	if interactive && !yes {
		if err := writeLine(); err != nil {
			return false, err
		}
		if _, err := fmt.Fprintf(out, i18n.T(i18n.PublishConfirmPrompt), branch, applied.TagName); err != nil {
			return false, err
		}
		if !readYes(os.Stdin) {
			return false, nil
		}
	}

	if perr := repo.Push("origin", branch); perr != nil {
		return false, fmt.Errorf(i18n.T(i18n.PublishPushBranchFailed), branch, applied.TagName, perr)
	}
	if perr := repo.Push("origin", applied.TagName); perr != nil {
		return false, fmt.Errorf(i18n.T(i18n.PublishPushTagFailed), applied.TagName, applied.TagName, perr)
	}
	if err := writeLine(); err != nil {
		return false, err
	}
	if err := writeLine(ui.Success([]string{i18n.T(i18n.PublishPushedToOrigin)})); err != nil {
		return false, err
	}

	prerelease := strings.Contains(applied.TagName, "-")
	url, cerr := ghrelease.Create(context.Background(), nil, token, ghrelease.Options{
		Repo:       ownerName,
		Tag:        applied.TagName,
		Name:       applied.TagName,
		Body:       plan.ReleaseBody(),
		Prerelease: prerelease,
	})
	if errors.Is(cerr, ghrelease.ErrReleaseExists) {
		if err := writeLine(ui.Info(i18n.T(i18n.PublishReleaseExists, applied.TagName))); err != nil {
			return true, err
		}
		return true, nil
	}
	if cerr != nil {
		return true, fmt.Errorf(i18n.T(i18n.PublishCreateFailed), applied.TagName, cerr)
	}

	if err := writeLine(ui.Success([]string{i18n.T(i18n.PublishReleasePublished, applied.TagName)})); err != nil {
		return true, err
	}
	if err := writeLine(ui.Dim.Render("  " + url)); err != nil {
		return true, err
	}
	return true, nil
}

// readYes reads one line from r and reports whether it is an affirmative
// ("y" / "yes", case-insensitive). Anything else — including EOF — is a no.
func readYes(r io.Reader) bool {
	line, _ := bufio.NewReader(r).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}
