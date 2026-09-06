package ghstats

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPlatformOf(t *testing.T) {
	cases := map[string]string{
		"relio_1.2.0_darwin_arm64.tar.gz":    "macOS arm64",
		"relio_1.2.0_darwin_amd64.tar.gz":    "macOS amd64",
		"relio_1.2.0_linux_amd64.tar.gz":     "Linux amd64",
		"relio_1.2.0_linux_arm64.tar.gz":     "Linux arm64",
		"relio_1.2.0_windows_amd64.zip":      "Windows amd64",
		"checksums.txt":                      "",
		"relio_1.2.0_linux_amd64.tar.gz.sig": "", // not os_arch tail after trim
	}
	for in, want := range cases {
		if got := platformOf(in); got != want {
			t.Errorf("platformOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsBinaryArchive(t *testing.T) {
	yes := []string{"relio_1.2.0_linux_amd64.tar.gz", "relio_1.2.0_windows_amd64.zip"}
	no := []string{"checksums.txt", "relio_1.2.0_linux_amd64.tar.gz.sig", "relio.sbom.json", "relio_1.2.0.pem"}
	for _, n := range yes {
		if !isBinaryArchive(n) {
			t.Errorf("isBinaryArchive(%q) = false, want true", n)
		}
	}
	for _, n := range no {
		if isBinaryArchive(n) {
			t.Errorf("isBinaryArchive(%q) = true, want false", n)
		}
	}
}

func TestNextLink(t *testing.T) {
	h := `<https://api.github.com/x?page=2>; rel="next", <https://api.github.com/x?page=5>; rel="last"`
	if got := nextLink(h); got != "https://api.github.com/x?page=2" {
		t.Errorf("nextLink = %q", got)
	}
	if got := nextLink(`<https://x>; rel="last"`); got != "" {
		t.Errorf("nextLink (no next) = %q", got)
	}
}

func TestFetch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/acme/relio", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"stargazers_count":126,"forks_count":14}`)
	})
	mux.HandleFunc("/repos/acme/relio/releases", func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "" || page == "1" {
			w.Header().Set("Link", `<`+baseURL+`/repos/acme/relio/releases?per_page=100&page=2>; rel="next"`)
			fmt.Fprint(w, `[
              {"tag_name":"v1.2.0","draft":false,"prerelease":false,"assets":[
                {"name":"relio_1.2.0_darwin_arm64.tar.gz","download_count":291},
                {"name":"relio_1.2.0_darwin_amd64.tar.gz","download_count":38},
                {"name":"relio_1.2.0_linux_amd64.tar.gz","download_count":82},
                {"name":"relio_1.2.0_linux_arm64.tar.gz","download_count":26},
                {"name":"checksums.txt","download_count":999}
              ]},
              {"tag_name":"v1.2.0-rc.1","draft":false,"prerelease":true,"assets":[
                {"name":"relio_1.2.0-rc.1_linux_amd64.tar.gz","download_count":5}
              ]}
            ]`)
			return
		}
		// page 2
		fmt.Fprint(w, `[
          {"tag_name":"v1.1.0","draft":false,"prerelease":false,"assets":[
            {"name":"relio_1.1.0_linux_amd64.tar.gz","download_count":521}
          ]},
          {"tag_name":"v1.0.0-draft","draft":true,"prerelease":false,"assets":[]}
        ]`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	baseURL = srv.URL
	APIBase = srv.URL
	t.Cleanup(func() { APIBase = "https://api.github.com" })

	s, err := Fetch(context.Background(), srv.Client(), "acme/relio")
	if err != nil {
		t.Fatal(err)
	}

	if s.Stars != 126 || s.Forks != 14 {
		t.Errorf("stars/forks = %d/%d", s.Stars, s.Forks)
	}
	// checksums.txt excluded; rc.1 counted in total but it's a prerelease
	wantTotal := (291 + 38 + 82 + 26) + 5 + 521
	if s.TotalDownloads != wantTotal {
		t.Errorf("total downloads = %d, want %d", s.TotalDownloads, wantTotal)
	}
	if len(s.Releases) != 3 { // draft skipped
		t.Fatalf("releases = %d, want 3", len(s.Releases))
	}
	if s.Latest == nil || s.Latest.Tag != "v1.2.0" {
		t.Fatalf("latest = %+v (want v1.2.0, skipping the prerelease)", s.Latest)
	}
	if s.Latest.Downloads != 291+38+82+26 {
		t.Errorf("latest downloads = %d", s.Latest.Downloads)
	}
	// platform label present on a real asset, absent on checksums
	var seenChecksum bool
	for _, a := range s.Latest.Assets {
		if a.Name == "checksums.txt" {
			seenChecksum = true
			if a.Platform != "" {
				t.Errorf("checksums.txt got platform %q", a.Platform)
			}
		}
	}
	if !seenChecksum {
		t.Error("checksums.txt asset missing from list")
	}
}

func TestFetchRateLimited(t *testing.T) {
	reset := time.Now().Add(20 * time.Minute).Unix()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", fmt.Sprint(reset))
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"message":"API rate limit exceeded"}`)
	}))
	defer srv.Close()
	APIBase = srv.URL
	t.Cleanup(func() { APIBase = "https://api.github.com" })

	_, err := Fetch(context.Background(), srv.Client(), "acme/relio")
	if _, ok := err.(*RateLimitError); !ok {
		t.Fatalf("err = %v (%T), want *RateLimitError", err, err)
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("error text = %q", err.Error())
	}
}

func TestFetchBadRepo(t *testing.T) {
	if _, err := Fetch(context.Background(), nil, "not-a-repo"); err == nil {
		t.Fatal("expected error for malformed repo")
	}
}

// baseURL mirrors APIBase for building the Link header in the mux handler.
var baseURL string
