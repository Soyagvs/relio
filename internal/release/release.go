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
	"github.com/soyagvs/relio/internal/versionfile"
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
	// VersionChanges are the edits to keep release.version_files in sync; empty
	// when the feature is unused. VersionFilesUpdate gates whether Apply makes
	// them (cleared by --no-version-files).
	VersionChanges     []versionfile.Change
	VersionFilesUpdate bool
	// Prerelease reports that Next is a release candidate (vX.Y.Z-rc.N).
	Prerelease bool
	// Finalizing reports that this run drops a pre-release to ship its core
	// (e.g. v1.6.0-rc.2 -> v1.6.0), summarising the whole span since the last
	// stable tag.
	Finalizing bool
	Now        time.Time
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
// Finalising a pre-release is always a release, even with no new commits.
func (p Plan) NothingToRelease() bool {
	if p.Finalizing {
		return false
	}
	return len(p.Commits) == 0
}

// Lint splits the plan's commits into conventional and not, preserving order.
func (p Plan) Lint() (ok, notConventional []conventional.Commit) {
	for _, c := range p.Commits {
		if c.IsConventional() {
			ok = append(ok, c)
		} else {
			notConventional = append(notConventional, c)
		}
	}
	return ok, notConventional
}

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
	// Prerelease cuts a release candidate (vX.Y.Z-rc.N) instead of a final
	// version.
	Prerelease bool
	// Now is injectable for deterministic tests; defaults to time.Now().
	Now time.Time
}

// BuildPlan inspects the repo and produces a Plan.
func BuildPlan(repo *gitrepo.Repo, cfg config.Config, opts Options) (Plan, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	latest, hasTag, err := repo.LatestTag()
	if err != nil {
		return Plan{}, err
	}

	var latestV semver.Version
	if hasTag {
		latestV, err = semver.Parse(latest)
		if err != nil {
			return Plan{}, fmt.Errorf("latest tag %q is not semver: %w", latest, err)
		}
	}

	// baseV is the stable version this release builds on. When the latest tag is
	// itself a pre-release we look past it to the newest stable tag.
	var (
		baseV      semver.Version
		stableBase string
	)
	switch {
	case hasTag && latestV.IsPrerelease():
		sb, hasSB, _ := repo.LatestStableTag()
		if hasSB {
			stableBase = sb
			baseV, err = semver.Parse(sb)
			if err != nil {
				return Plan{}, fmt.Errorf("stable tag %q is not semver: %w", sb, err)
			}
		} else {
			baseV = semver.Zero()
		}
	case hasTag:
		baseV = latestV
	default:
		baseV = semver.Zero()
	}

	finalizing := !opts.Prerelease && hasTag && latestV.IsPrerelease()

	// rangeFrom is where the commit scan starts. When finalising a pre-release
	// we summarise the whole span since the last stable tag; otherwise it is the
	// latest tag (or the start of history).
	var rangeFrom string
	switch {
	case finalizing:
		rangeFrom = stableBase
	case hasTag:
		rangeFrom = latest
	default:
		rangeFrom = ""
	}

	raw, err := repo.CommitsSince(rangeFrom)
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

	// coreTarget is the stable version this run aims at.
	coreTarget := baseV.Next(bump)
	if finalizing {
		coreTarget = latestV.Core()
	}

	current := semver.Zero()
	if hasTag {
		current = latestV
	}

	var next semver.Version
	prerelease := false
	switch {
	case opts.Prerelease:
		prerelease = true
		if hasTag && latestV.IsPrerelease() &&
			semver.Compare(coreTarget, latestV.Core()) <= 0 {
			// Same core as the current rc — just bump the counter.
			next = latestV.NextPre()
		} else {
			// First rc for this core (stable -> rc, or an escalated core).
			next = coreTarget.WithPre("rc.1")
		}
	default:
		// Finalising -> the rc's core; normal path -> today's behaviour.
		next = coreTarget
	}

	var versionChanges []versionfile.Change
	versionFilesUpdate := false
	if len(cfg.Release.VersionFiles) > 0 {
		versionChanges, err = versionfile.Plan(
			repo.Root(), cfg.Release.VersionTargets(), strings.TrimPrefix(next.String(), "v"))
		if err != nil {
			return Plan{}, err
		}
		versionFilesUpdate = true
	}

	return Plan{
		Config:             cfg,
		Current:            current,
		Next:               next,
		Bump:               bump,
		BumpForced:         forced,
		Commits:            commits,
		Notes:              changelog.Build(commits),
		ChangelogUpdate:    cfg.Release.ChangelogEnabled(),
		TagUpdate:          cfg.Release.TagEnabled(),
		PublishGitHub:      cfg.GitHub.Release,
		VersionChanges:     versionChanges,
		VersionFilesUpdate: versionFilesUpdate,
		Prerelease:         prerelease,
		Finalizing:         finalizing,
		Now:                now,
	}, nil
}

// ApplyResult reports what Apply actually changed.
type ApplyResult struct {
	ChangelogPath string
	Committed     bool
	TagName       string
	VersionFiles  []string // Rel paths of the version files written
}

// Apply writes the changelog file, syncs any declared version files, commits
// them together, and creates the git tag, per the plan's flags. The changelog
// and version files are written before the single "chore(release): vX.Y.Z"
// commit, so the tag points at a commit where every file agrees on the version.
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

	// commitPaths accumulates every file that must ride in the release commit —
	// the changelog first, then any version files — so the tag points at a
	// commit where they all agree on the version.
	var commitPaths []string

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
		commitPaths = append(commitPaths, p.Config.Release.ChangelogFile)
	}

	if p.VersionFilesUpdate && len(p.VersionChanges) > 0 {
		if err := versionfile.Apply(p.VersionChanges); err != nil {
			return res, err
		}
		for _, c := range p.VersionChanges {
			res.VersionFiles = append(res.VersionFiles, c.Rel)
			commitPaths = append(commitPaths, c.Rel)
		}
	}

	if p.TagUpdate && len(commitPaths) > 0 {
		msg := fmt.Sprintf("chore(release): %s", name)
		if err := repo.CommitPaths(msg, commitPaths...); err != nil {
			return res, fmt.Errorf("committing release files: %w", err)
		}
		res.Committed = true
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
