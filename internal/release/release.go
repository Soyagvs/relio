// Package release orchestrates the read side (build a Plan from repo state) and
// the write side (apply a Plan: changelog + tag).
package release

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/soyagvs/go-release/internal/changelog"
	"github.com/soyagvs/go-release/internal/config"
	"github.com/soyagvs/go-release/internal/conventional"
	"github.com/soyagvs/go-release/internal/gitrepo"
	"github.com/soyagvs/go-release/internal/semver"
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
		Now:             now,
	}, nil
}

// ApplyResult reports what Apply actually changed.
type ApplyResult struct {
	ChangelogPath string
	TagName       string
}

// Apply writes the changelog file and creates the git tag, per the plan's flags.
func (p Plan) Apply(repo *gitrepo.Repo) (ApplyResult, error) {
	var res ApplyResult

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
	}

	if p.TagUpdate {
		name := p.TagName()
		exists, err := repo.HasTag(name)
		if err != nil {
			return res, err
		}
		if exists {
			return res, fmt.Errorf("tag %s already exists", name)
		}
		msg := fmt.Sprintf("release %s", name)
		if err := repo.CreateTag(name, msg); err != nil {
			return res, err
		}
		res.TagName = name
	}

	return res, nil
}
