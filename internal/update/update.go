// Package update performs a best-effort check for a newer published Relio
// release and renders a one-line notice for the interactive menu.
//
// It is deliberately unobtrusive:
//
//   - the menu hot path only ever reads a small on-disk cache — it never blocks
//     on the network;
//   - a cache older than the TTL triggers a detached background refresh whose
//     result is shown on the *next* run, the same approach npm's
//     update-notifier takes;
//   - the only network traffic is an anonymous GET for the latest release tag.
//     Nothing about the user, the repository, or the invocation is ever sent —
//     this stays true to Relio's no-telemetry stance.
//
// Set RELIO_NO_UPDATE_CHECK to any value (or run in CI) to turn it off.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// APIBase is the GitHub REST API root; overridable in tests.
var APIBase = "https://api.github.com"

// Repo is the "owner/name" whose releases are checked; overridable in tests.
var Repo = "soyagvs/relio"

const (
	ttl        = 24 * time.Hour
	fetchLimit = 3 * time.Second
	disableEnv = "RELIO_NO_UPDATE_CHECK"
)

// cacheDir is the parent of the cache file. Empty means os.UserCacheDir(); tests
// point it at a temp directory.
var cacheDir string

// asyncRefresh runs the network refresh in a detached goroutine in production.
// Tests set it to false so the refresh completes inline.
var asyncRefresh = true

type cache struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"`
}

// Available returns the latest published version as a bare "1.2.3" (no "v") when
// it is newer than current, or "" when the build is current, the check is
// disabled, or nothing is cached yet. It never makes a blocking network call; a
// stale or missing cache is refreshed in the background so the result appears on
// the following run.
func Available(current string) string {
	if !enabled(current) {
		return ""
	}

	c, ok := readCache()
	if !ok || time.Since(c.CheckedAt) > ttl {
		if asyncRefresh {
			go backgroundRefresh()
		} else {
			backgroundRefresh()
			c, ok = readCache()
		}
	}
	if !ok || c.Latest == "" {
		return ""
	}

	cur, curOK := parse(current)
	lat, latOK := parse(c.Latest)
	if !curOK || !latOK || !less(cur, lat) {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(c.Latest), "v")
}

// enabled reports whether the check should run for this build at all.
func enabled(current string) bool {
	if os.Getenv(disableEnv) != "" || os.Getenv("CI") != "" {
		return false
	}
	_, ok := parse(current) // "dev" / "" / anything unparseable -> skip
	return ok
}

func backgroundRefresh() {
	ctx, cancel := context.WithTimeout(context.Background(), fetchLimit)
	defer cancel()

	tag, err := fetchLatest(ctx, http.DefaultClient)
	if err != nil || tag == "" {
		return
	}
	writeCache(cache{CheckedAt: time.Now(), Latest: tag})
}

// fetchLatest asks GitHub for the newest published (non-prerelease) release tag.
func fetchLatest(ctx context.Context, c *http.Client) (string, error) {
	url := APIBase + "/repos/" + Repo + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "relio")

	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("github: %s", resp.Status)
	}

	var body struct {
		Tag string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&body); err != nil {
		return "", err
	}
	return strings.TrimSpace(body.Tag), nil
}

func cachePath() (string, error) {
	base := cacheDir
	if base == "" {
		d, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		base = d
	}
	return filepath.Join(base, "relio", "update-check.json"), nil
}

func readCache() (cache, bool) {
	p, err := cachePath()
	if err != nil {
		return cache{}, false
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return cache{}, false
	}
	var c cache
	if err := json.Unmarshal(b, &c); err != nil {
		return cache{}, false
	}
	return c, true
}

func writeCache(c cache) {
	p, err := cachePath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	b, err := json.Marshal(c)
	if err != nil {
		return
	}
	// Atomic replace so a process exiting mid-write never leaves a torn file.
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
	}
}

// ver is a lenient MAJOR.MINOR.PATCH triple.
type ver struct{ major, minor, patch int }

// parse reads "v1.2.3" / "1.2.3", ignoring any "-rc.1" / "+meta" suffix. It
// returns ok=false for "dev", "", and anything that is not three integers.
func parse(s string) (ver, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return ver{}, false
	}
	out := [3]int{}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return ver{}, false
		}
		out[i] = n
	}
	return ver{out[0], out[1], out[2]}, true
}

// less reports whether a is an older version than b.
func less(a, b ver) bool {
	switch {
	case a.major != b.major:
		return a.major < b.major
	case a.minor != b.minor:
		return a.minor < b.minor
	default:
		return a.patch < b.patch
	}
}
