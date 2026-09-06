// Package ghstats reads public release/download statistics for a GitHub
// repository. It only ever performs GET requests to the GitHub REST API — no
// telemetry, no tracking, nothing is sent anywhere else.
//
// "Downloads" means the number of times a release asset (a binary archive) was
// downloaded — not unique users or active installs.
package ghstats

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// APIBase is the GitHub REST API root; overridable in tests.
var APIBase = "https://api.github.com"

// Stats is the full picture for one repository.
type Stats struct {
	Repo           string
	Stars          int
	Forks          int
	TotalDownloads int
	Releases       []Release // newest first
	Latest         *Release  // points into Releases, nil if none published
}

// Release is one GitHub Release plus its per-asset download counts.
type Release struct {
	Tag        string
	Prerelease bool
	Downloads  int // sum of binary-archive assets only
	Assets     []Asset
}

// Asset is one uploaded file.
type Asset struct {
	Name          string
	DownloadCount int
	Platform      string // "macOS arm64" etc.; "" when not a platform binary
}

// RateLimitError is returned when GitHub refuses the request for rate limiting.
type RateLimitError struct{ Reset time.Time }

func (e *RateLimitError) Error() string {
	when := "soon"
	if !e.Reset.IsZero() {
		when = e.Reset.Local().Format("15:04:05")
	}
	return fmt.Sprintf("GitHub API rate limit reached (resets at %s). "+
		"Set GITHUB_TOKEN to raise the limit.", when)
}

// Fetch gathers the stats for repo ("owner/name"). client may be nil.
func Fetch(ctx context.Context, client *http.Client, repo string) (*Stats, error) {
	if client == nil {
		client = http.DefaultClient
	}
	repo = strings.Trim(strings.TrimSpace(repo), "/")
	if strings.Count(repo, "/") != 1 || strings.HasPrefix(repo, "/") {
		return nil, fmt.Errorf("repo must be \"owner/name\", got %q", repo)
	}

	s := &Stats{Repo: repo}

	var meta struct {
		Stars int `json:"stargazers_count"`
		Forks int `json:"forks_count"`
	}
	if err := get(ctx, client, APIBase+"/repos/"+repo, &meta); err != nil {
		return nil, err
	}
	s.Stars, s.Forks = meta.Stars, meta.Forks

	url := APIBase + "/repos/" + repo + "/releases?per_page=100"
	for url != "" {
		var page []struct {
			Tag        string `json:"tag_name"`
			Draft      bool   `json:"draft"`
			Prerelease bool   `json:"prerelease"`
			Assets     []struct {
				Name          string `json:"name"`
				DownloadCount int    `json:"download_count"`
			} `json:"assets"`
		}
		next, err := getPaged(ctx, client, url, &page)
		if err != nil {
			return nil, err
		}
		for _, r := range page {
			if r.Draft {
				continue
			}
			rel := Release{Tag: r.Tag, Prerelease: r.Prerelease}
			for _, a := range r.Assets {
				as := Asset{Name: a.Name, DownloadCount: a.DownloadCount, Platform: platformOf(a.Name)}
				rel.Assets = append(rel.Assets, as)
				if isBinaryArchive(a.Name) {
					rel.Downloads += a.DownloadCount
				}
			}
			s.Releases = append(s.Releases, rel)
			s.TotalDownloads += rel.Downloads
		}
		url = next
	}

	for i := range s.Releases {
		if !s.Releases[i].Prerelease {
			s.Latest = &s.Releases[i]
			break
		}
	}
	if s.Latest == nil && len(s.Releases) > 0 {
		s.Latest = &s.Releases[0]
	}
	return s, nil
}

// isBinaryArchive reports whether name is a downloadable executable archive
// (and not checksums / signatures / SBOMs).
func isBinaryArchive(name string) bool {
	n := strings.ToLower(name)
	switch {
	case strings.HasSuffix(n, ".sig"), strings.HasSuffix(n, ".pem"),
		strings.HasSuffix(n, ".sbom"), strings.HasSuffix(n, ".sbom.json"),
		n == "checksums.txt", strings.HasSuffix(n, "checksums.txt"):
		return false
	}
	return strings.HasSuffix(n, ".tar.gz") || strings.HasSuffix(n, ".tgz") || strings.HasSuffix(n, ".zip")
}

var osLabel = map[string]string{"darwin": "macOS", "linux": "Linux", "windows": "Windows"}

// platformOf turns "relio_1.2.0_darwin_arm64.tar.gz" into "macOS arm64".
func platformOf(name string) string {
	n := name
	for _, ext := range []string{".tar.gz", ".tgz", ".zip"} {
		n = strings.TrimSuffix(n, ext)
	}
	parts := strings.Split(n, "_")
	if len(parts) < 2 {
		return ""
	}
	arch := strings.ToLower(parts[len(parts)-1])
	label, ok := osLabel[strings.ToLower(parts[len(parts)-2])]
	if !ok {
		return ""
	}
	switch arch {
	case "amd64", "arm64", "386", "arm", "x86_64", "aarch64":
	default:
		return ""
	}
	return label + " " + arch
}

func get(ctx context.Context, c *http.Client, url string, out any) error {
	_, err := do(ctx, c, url, out)
	return err
}

// getPaged decodes url into out and returns the URL of the next page ("" if none).
func getPaged(ctx context.Context, c *http.Client, url string, out any) (string, error) {
	return do(ctx, c, url, out)
}

func do(ctx context.Context, c *http.Client, url string, out any) (nextURL string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "relio-stats")
	if tok := token(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		// Error bodies are tiny; a page of releases can be many MB, so only the
		// error path buffers.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		switch {
		case resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests:
			if resp.Header.Get("X-RateLimit-Remaining") == "0" {
				var reset time.Time
				if v, e := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64); e == nil {
					reset = time.Unix(v, 0)
				}
				return "", &RateLimitError{Reset: reset}
			}
			return "", fmt.Errorf("GitHub API %d: %s", resp.StatusCode, firstLine(body))
		case resp.StatusCode == http.StatusNotFound:
			return "", fmt.Errorf("repository not found (or has no public releases)")
		default:
			return "", fmt.Errorf("GitHub API %d: %s", resp.StatusCode, firstLine(body))
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return "", fmt.Errorf("decoding GitHub response: %w", err)
	}
	return nextLink(resp.Header.Get("Link")), nil
}

func token() string {
	for _, k := range []string{"RELIO_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

// nextLink extracts the rel="next" URL from a GitHub Link header.
func nextLink(header string) string {
	for _, part := range strings.Split(header, ",") {
		seg := strings.Split(strings.TrimSpace(part), ";")
		if len(seg) < 2 {
			continue
		}
		url := strings.Trim(strings.TrimSpace(seg[0]), "<>")
		for _, p := range seg[1:] {
			if strings.TrimSpace(p) == `rel="next"` {
				return url
			}
		}
	}
	return ""
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
