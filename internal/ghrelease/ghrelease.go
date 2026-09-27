// Package ghrelease is the write side of relio's GitHub integration: it pushes a
// tag's changelog notes to GitHub by creating a Release. Like ghstats it only
// ever talks to the GitHub REST API — nothing is sent anywhere else, and the
// token never leaves the process except in the Authorization header.
package ghrelease

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/soyagvs/relio/internal/tokenstore"
)

// APIBase is the GitHub REST API root; overridable in tests.
var APIBase = "https://api.github.com"

// userAgent identifies relio to the GitHub API.
const userAgent = "relio"

// RelioOAuthClientID is relio's registered GitHub OAuth App client ID, used
// by `relio auth login`'s Device Flow (see deviceflow.go). It is public by
// design -- the Device Flow never needs a client secret -- the same way the
// `gh` and Docker CLIs hardcode theirs.
const RelioOAuthClientID = "Ov23liuEAG3Q8f19pkUJ"

// Options describes the GitHub Release to create.
type Options struct {
	Repo       string // "owner/name"
	Tag        string
	Name       string
	Body       string
	Prerelease bool
}

// ErrReleaseExists is returned by Create when GitHub already has a Release for
// the tag. It is wrapped with the tag name, so callers can both errors.Is it and
// print a useful message.
var ErrReleaseExists = errors.New("a GitHub Release already exists for this tag")

// RateLimitError is returned when GitHub refuses the request for rate limiting.
// It mirrors ghstats' shape so the CLI can format both the same way.
type RateLimitError struct{ Reset time.Time }

func (e *RateLimitError) Error() string {
	when := "soon"
	if !e.Reset.IsZero() {
		when = e.Reset.Local().Format("15:04:05")
	}
	return fmt.Sprintf("GitHub API rate limit reached (resets at %s). "+
		"Set GITHUB_TOKEN to raise the limit.", when)
}

// Create publishes a GitHub Release for opt.Tag and returns its html_url. The
// tag must already exist on the remote. client may be nil.
func Create(ctx context.Context, client *http.Client, token string, opt Options) (htmlURL string, err error) {
	if client == nil {
		client = http.DefaultClient
	}
	owner, name, err := splitRepo(opt.Repo)
	if err != nil {
		return "", err
	}

	payload := map[string]any{
		"tag_name":   opt.Tag,
		"name":       opt.Name,
		"body":       opt.Body,
		"prerelease": opt.Prerelease,
		"draft":      false,
	}
	if !opt.Prerelease {
		payload["make_latest"] = "true"
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := APIBase + "/repos/" + owner + "/" + name + "/releases"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))

	if resp.StatusCode == http.StatusCreated {
		var out struct {
			HTMLURL string `json:"html_url"`
		}
		if err := json.Unmarshal(body, &out); err != nil {
			return "", fmt.Errorf("decoding GitHub response: %w", err)
		}
		return out.HTMLURL, nil
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return "", errors.New("GitHub rejected the token (401): check its `repo` scope")
	case http.StatusForbidden, http.StatusTooManyRequests:
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			var reset time.Time
			if v, e := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64); e == nil {
				reset = time.Unix(v, 0)
			}
			return "", &RateLimitError{Reset: reset}
		}
		return "", fmt.Errorf("GitHub API %d: %s", resp.StatusCode, firstLine(body))
	case http.StatusUnprocessableEntity:
		if hasErrorCode(body, "already_exists") {
			return "", fmt.Errorf("%w: %s", ErrReleaseExists, opt.Tag)
		}
		return "", fmt.Errorf("GitHub API %d: %s", resp.StatusCode, firstLine(body))
	default:
		return "", fmt.Errorf("GitHub API %d: %s", resp.StatusCode, firstLine(body))
	}
}

// AuthenticatedUser returns the login of whoever the token belongs to. Used by
// `relio auth status`. client may be nil.
func AuthenticatedUser(ctx context.Context, client *http.Client, token string) (login string, err error) {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, APIBase+"/user", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))

	if resp.StatusCode == http.StatusUnauthorized {
		return "", errors.New("GitHub rejected the token (401): check its `repo` scope")
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("GitHub API %d: %s", resp.StatusCode, firstLine(body))
	}
	var out struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decoding GitHub response: %w", err)
	}
	return out.Login, nil
}

// loadStoredToken resolves the token saved by `relio auth login`'s OAuth
// Device Flow (see internal/tokenstore). It is a package var aliasing
// tokenstore.Load purely so Token()'s tests can stub it without touching a
// real OS keychain or file.
var loadStoredToken = tokenstore.Load

// Token resolves a GitHub token, checking sources in this order:
//
//  1. RELIO_GITHUB_TOKEN, GITHUB_TOKEN, GH_TOKEN environment variables, in
//     that order (source is the env var name). These stay top priority so
//     CI/scripts can always force an explicit token, overriding anything
//     `relio auth login` stored.
//  2. The token stored by `relio auth login`'s OAuth Device Flow, via
//     internal/tokenstore (source is "keychain" or "file", whichever backend
//     held it). A read error here is treated the same as "nothing found"
//     and falls through to the next source, rather than failing Token().
//  3. `gh auth token`, as a last-resort fallback for anyone already signed
//     into the GitHub CLI (source is "gh").
//
// Both values are "" when nothing is found in any source.
func Token() (tok string, source string) {
	for _, k := range []string{"RELIO_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v, k
		}
	}
	if v, src, err := loadStoredToken(); err == nil && v != "" {
		return v, src
	}
	if v := strings.TrimSpace(ghAuthToken()); v != "" {
		return v, "gh"
	}
	return "", ""
}

// ghAuthToken shells out to `gh auth token`. Any failure yields "" — a missing
// token is "not authenticated", never a hard error.
func ghAuthToken() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// ParseRepo turns a git remote URL into "Owner/Name". It accepts the SSH and
// HTTPS forms GitHub hands out and rejects anything that is not github.com.
func ParseRepo(remoteURL string) (ownerName string, err error) {
	raw := strings.TrimSpace(remoteURL)
	if raw == "" {
		return "", errors.New("no remote URL to parse")
	}

	var path string
	switch {
	case strings.HasPrefix(raw, "git@"):
		host, rest, ok := strings.Cut(strings.TrimPrefix(raw, "git@"), ":")
		if !ok || !strings.EqualFold(host, "github.com") {
			return "", fmt.Errorf("not a github.com remote: %q", remoteURL)
		}
		path = rest
	case strings.HasPrefix(raw, "ssh://"), strings.HasPrefix(raw, "https://"), strings.HasPrefix(raw, "http://"):
		u, perr := url.Parse(raw)
		if perr != nil {
			return "", fmt.Errorf("parsing remote URL %q: %w", remoteURL, perr)
		}
		if !strings.EqualFold(u.Hostname(), "github.com") {
			return "", fmt.Errorf("not a github.com remote: %q", remoteURL)
		}
		path = strings.TrimPrefix(u.Path, "/")
	default:
		return "", fmt.Errorf("unrecognised remote URL: %q", remoteURL)
	}

	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	if strings.Count(path, "/") != 1 || strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") {
		return "", fmt.Errorf("cannot read owner/name from %q", remoteURL)
	}
	return path, nil
}

func splitRepo(repo string) (owner, name string, err error) {
	repo = strings.Trim(strings.TrimSpace(repo), "/")
	if strings.Count(repo, "/") != 1 || strings.HasPrefix(repo, "/") {
		return "", "", fmt.Errorf("repo must be \"owner/name\", got %q", repo)
	}
	owner, name, _ = strings.Cut(repo, "/")
	return owner, name, nil
}

// hasErrorCode reports whether a GitHub error body carries an errors[] entry
// with the given code.
func hasErrorCode(body []byte, code string) bool {
	var e struct {
		Errors []struct {
			Code string `json:"code"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &e) != nil {
		return false
	}
	for _, it := range e.Errors {
		if it.Code == code {
			return true
		}
	}
	return false
}

func firstLine(b []byte) string {
	s := strings.TrimSpace(string(b))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}
