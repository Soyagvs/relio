// Package release orchestrates the read side (build a Plan from repo state) and
// the write side (apply a Plan: write the changelog, commit it, then tag).
package release

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/soyagvs/relio/internal/changelog"
	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/conventional"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/semver"
)

// Plan is a proposed release, fully computed but not yet applied.
type Plan struct {
	Config          config.Config
	Current         semver.Version
	Next            semver.Version
	Bump            semver.Bump
	BumpForced      bool
	Commits         []conventional.Commit
	Notes           changelog.Notes
	ChangelogUpdate bool
	TagUpdate       bool
	PublishGitHub   bool
	Now             time.Time
}

// TagName is the git tag the plan will create, honouring the configured prefix.
func (p Plan) TagName() string {
	name := p.Next.String()
	prefix := p.Config.Release.TagPrefix
	if prefix != "" && !strings.HasPrefix(name, prefix) {
		name = prefix + strings.TrimPrefix(name, "v")
	}
	if prefix == "" {
		name = strings.TrimPrefix(name, "v")
	}
	return name
}

// Section renders the changelog block for this release.
func (p Plan) Section() string {
	return changelog.RenderSection(p.Next.String(), p.Now, p.Notes)
}

// NothingToRelease reports whether there are no commits since the last tag.
func (p Plan) NothingToRelease() bool { return len(p.Commits) == 0 }

// ReleaseBody is the changelog notes for this release without the
// "## [x.y.z] - date" heading line — the text to send as a GitHub Release body.
func (p Plan) ReleaseBody() string {
	section := p.Section()
	if i := strings.IndexByte(section, '\n'); i >= 0 {
		return strings.TrimSpace(section[i+1:])
	}
	return ""
}

// Options tune plan construction.
type Options struct {
	// ForceBump overrides the computed bump when not semver.None.
	ForceBump semver.Bump
	// Now is injectable for deterministic tests; defaults to time.Now().
	Now time.Time
}

// BuildPlan inspects the repo and produces a Plan.
func BuildPlan(repo *gitrepo.Repo, cfg config.Config, opts Options) (Plan, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	tag, hasTag, err := repo.LatestTag()
	if err != nil {
		return Plan{}, err
	}

	current := semver.Zero()
	if hasTag {
		current, err = semver.Parse(tag)
		if err != nil {
			return Plan{}, fmt.Errorf("latest tag %q is not semver: %w", tag, err)
		}
	}

	raw, err := repo.CommitsSince(tag)
	if err != nil {
		return Plan{}, err
	}
	commits := conventional.ParseMany(raw)

	bump := semver.BumpFor(commits)
	forced := false
	if opts.ForceBump != semver.None {
		bump = opts.ForceBump
		forced = true
	}
	// With commits present but no conventional signal, default to a patch so a
	// release is still possible.
	if bump == semver.None && len(commits) > 0 {
		bump = semver.Patch
	}

	return Plan{
		Config:          cfg,
		Current:         current,
		Next:            current.Next(bump),
		Bump:            bump,
		BumpForced:      forced,
		Commits:         commits,
		Notes:           changelog.Build(commits),
		ChangelogUpdate: cfg.Release.Changelog,
		TagUpdate:       cfg.Release.Tag,
		PublishGitHub:   cfg.GitHub.Release,
		Now:             now,
	}, nil
}

// ApplyResult reports what Apply actually changed.
type ApplyResult struct {
	ChangelogPath string
	Committed     bool
	TagName       string
}

// Apply writes the changelog file, commits it, and creates the git tag, per the
// plan's flags. When both the changelog and the tag are enabled the changelog is
// committed first, so the tag points at a commit that already carries its own
// changelog section.
func (p Plan) Apply(repo *gitrepo.Repo) (ApplyResult, error) {
	var res ApplyResult

	// Fail before touching anything if the target tag is already taken.
	name := p.TagName()
	if p.TagUpdate {
		exists, err := repo.HasTag(name)
		if err != nil {
			return res, err
		}
		if exists {
			return res, fmt.Errorf("tag %s already exists", name)
		}
	}

	if p.ChangelogUpdate {
		path := filepath.Join(repo.Root(), p.Config.Release.ChangelogFile)
		existing, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return res, fmt.Errorf("reading %s: %w", p.Config.Release.ChangelogFile, err)
		}
		updated := changelog.Update(string(existing), p.Section())
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			return res, fmt.Errorf("writing %s: %w", p.Config.Release.ChangelogFile, err)
		}
		res.ChangelogPath = path

		if p.TagUpdate {
			msg := fmt.Sprintf("chore(release): %s", name)
			if err := repo.CommitPaths(msg, p.Config.Release.ChangelogFile); err != nil {
				return res, fmt.Errorf("committing %s: %w", p.Config.Release.ChangelogFile, err)
			}
			res.Committed = true
		}
	}

	if p.TagUpdate {
		msg := fmt.Sprintf("release %s", name)
		if err := repo.CreateTag(name, msg); err != nil {
			return res, err
		}
		res.TagName = name
	}

	return res, nil
}
