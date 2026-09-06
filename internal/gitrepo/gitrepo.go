// Package gitrepo is a thin wrapper over the system `git` binary, exposing only
// the read and write operations the release tool needs.
package gitrepo

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/soyagvs/go-release/internal/conventional"
)

// unitSep and recordSep delimit `git log` output so subjects/bodies can contain
// anything without breaking parsing.
const (
	unitSep   = "\x1f"
	recordSep = "\x1e"
)

// Repo is an opened git repository.
type Repo struct {
	root string
}

// ErrNotARepo is returned by Open when dir is not inside a git work tree.
var ErrNotARepo = errors.New("gitrepo: not a git repository")

// Open verifies that dir is inside a git work tree and returns a Repo rooted at
// its top level.
func Open(dir string) (*Repo, error) {
	out, err := run(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, ErrNotARepo
	}
	return &Repo{root: strings.TrimSpace(out)}, nil
}

// Root is the absolute path to the repository top level.
func (r *Repo) Root() string { return r.root }

// LatestTag returns the most recent tag reachable from HEAD. ok is false when the
// repository has no tags yet.
func (r *Repo) LatestTag() (tag string, ok bool, err error) {
	out, err := run(r.root, "describe", "--tags", "--abbrev=0")
	if err != nil {
		if strings.Contains(err.Error(), "No names found") ||
			strings.Contains(err.Error(), "No tags can describe") ||
			strings.Contains(err.Error(), "cannot describe") {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(out), true, nil
}

// CommitsSince returns commits in sinceTag..HEAD, oldest first. When sinceTag is
// empty every commit reachable from HEAD is returned.
func (r *Repo) CommitsSince(sinceTag string) ([]conventional.Raw, error) {
	format := "%H" + unitSep + "%s" + unitSep + "%b" + recordSep
	args := []string{"log", "--reverse", "--no-merges", "--pretty=format:" + format}
	if sinceTag != "" {
		args = append(args, sinceTag+"..HEAD")
	}
	out, err := run(r.root, args...)
	if err != nil {
		return nil, err
	}
	return parseLog(out), nil
}

func parseLog(out string) []conventional.Raw {
	var commits []conventional.Raw
	for _, rec := range strings.Split(out, recordSep) {
		rec = strings.Trim(rec, "\n")
		if rec == "" {
			continue
		}
		fields := strings.SplitN(rec, unitSep, 3)
		if len(fields) < 2 {
			continue
		}
		c := conventional.Raw{Hash: fields[0], Subject: fields[1]}
		if len(fields) == 3 {
			c.Body = strings.TrimSpace(fields[2])
		}
		commits = append(commits, c)
	}
	return commits
}

// IsClean reports whether the work tree has no staged or unstaged changes.
func (r *Repo) IsClean() (bool, error) {
	out, err := run(r.root, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

// HasTag reports whether a tag with the given name already exists.
func (r *Repo) HasTag(name string) (bool, error) {
	out, err := run(r.root, "tag", "--list", name)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// CreateTag creates an annotated tag at HEAD.
func (r *Repo) CreateTag(name, message string) error {
	_, err := run(r.root, "tag", "-a", name, "-m", message)
	return err
}

// RemoteURL returns the URL configured for the given remote (usually "origin").
func (r *Repo) RemoteURL(remote string) (string, error) {
	out, err := run(r.root, "remote", "get-url", remote)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// HeadShortHash returns the abbreviated hash of HEAD.
func (r *Repo) HeadShortHash() (string, error) {
	out, err := run(r.root, "rev-parse", "--short", "HEAD")
	return strings.TrimSpace(out), err
}

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}
