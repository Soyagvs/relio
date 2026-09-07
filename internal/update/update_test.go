package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in     string
		want   ver
		wantOK bool
	}{
		{"v1.2.3", ver{1, 2, 3}, true},
		{"1.2.3", ver{1, 2, 3}, true},
		{" v1.2.3 ", ver{1, 2, 3}, true},
		{"v1.2.3-rc.1", ver{1, 2, 3}, true},
		{"1.2.3+build.9", ver{1, 2, 3}, true},
		{"dev", ver{}, false},
		{"", ver{}, false},
		{"1.2", ver{}, false},
		{"1.2.3.4", ver{}, false},
		{"v1.x.0", ver{}, false},
	}
	for _, c := range cases {
		got, ok := parse(c.in)
		if ok != c.wantOK || (ok && got != c.want) {
			t.Errorf("parse(%q) = %+v,%v; want %+v,%v", c.in, got, ok, c.want, c.wantOK)
		}
	}
}

func TestLess(t *testing.T) {
	cases := []struct {
		a, b ver
		want bool
	}{
		{ver{1, 2, 3}, ver{1, 2, 4}, true},
		{ver{1, 2, 3}, ver{1, 3, 0}, true},
		{ver{1, 9, 9}, ver{2, 0, 0}, true},
		{ver{1, 2, 3}, ver{1, 2, 3}, false},
		{ver{2, 0, 0}, ver{1, 9, 9}, false},
	}
	for _, c := range cases {
		if got := less(c.a, c.b); got != c.want {
			t.Errorf("less(%+v,%+v) = %v; want %v", c.a, c.b, got, c.want)
		}
	}
}

// isolate points the cache at a temp dir and disables the env kill-switches so a
// stray CI= in the runner's environment can't turn the check off.
func isolate(t *testing.T) {
	t.Helper()
	cacheDir = t.TempDir()
	asyncRefresh = false
	t.Setenv(disableEnv, "")
	t.Setenv("CI", "")
	t.Cleanup(func() {
		cacheDir = ""
		asyncRefresh = true
		APIBase = "https://api.github.com"
		Repo = "soyagvs/relio"
	})
}

func seedCache(t *testing.T, c cache) {
	t.Helper()
	writeCache(c)
	p, _ := cachePath()
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("seed cache: %v", err)
	}
}

func TestNoticeFreshCache(t *testing.T) {
	isolate(t)
	seedCache(t, cache{CheckedAt: time.Now(), Latest: "v1.5.0"})

	if got := Notice("v1.5.0"); got != "" {
		t.Errorf("same version: got %q, want \"\"", got)
	}
	if got := Notice("v2.0.0"); got != "" {
		t.Errorf("ahead of latest: got %q, want \"\"", got)
	}

	got := Notice("v1.4.9")
	if got == "" {
		t.Fatal("behind latest: got \"\", want a notice")
	}
	for _, sub := range []string{"1.5.0", "1.4.9", "brew upgrade relio"} {
		if !contains(got, sub) {
			t.Errorf("notice %q missing %q", got, sub)
		}
	}
}

func TestNoticeDisabled(t *testing.T) {
	isolate(t)
	seedCache(t, cache{CheckedAt: time.Now(), Latest: "v9.9.9"})

	t.Setenv(disableEnv, "1")
	if got := Notice("v0.1.0"); got != "" {
		t.Errorf("RELIO_NO_UPDATE_CHECK set: got %q, want \"\"", got)
	}
}

func TestNoticeDevBuildNeverChecks(t *testing.T) {
	isolate(t)
	seedCache(t, cache{CheckedAt: time.Now(), Latest: "v9.9.9"})

	if got := Notice("dev"); got != "" {
		t.Errorf("dev build: got %q, want \"\"", got)
	}
}

func TestNoticeNoCacheRefreshesInline(t *testing.T) {
	isolate(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/soyagvs/relio/releases/latest" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": "v3.1.0"})
	}))
	defer srv.Close()
	APIBase = srv.URL

	// First call: cache is empty, so it refreshes inline and then renders.
	got := Notice("v3.0.0")
	if !contains(got, "3.1.0") {
		t.Fatalf("after inline refresh: got %q, want a 3.1.0 notice", got)
	}

	// The refresh must have persisted, so a second call needs no server.
	APIBase = "http://127.0.0.1:0"
	if got := Notice("v3.0.0"); !contains(got, "3.1.0") {
		t.Errorf("second call (cached): got %q", got)
	}
}

func TestNoticeStaleCacheStillShowsWhileRefreshing(t *testing.T) {
	isolate(t)
	APIBase = "http://127.0.0.1:0" // refresh will fail; stale value must survive
	seedCache(t, cache{CheckedAt: time.Now().Add(-48 * time.Hour), Latest: "v2.0.0"})

	if got := Notice("v1.0.0"); !contains(got, "2.0.0") {
		t.Errorf("stale cache: got %q, want the stale 2.0.0 notice", got)
	}
}

func TestFetchLatest(t *testing.T) {
	isolate(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": "v4.2.0"})
	}))
	defer srv.Close()
	APIBase = srv.URL

	tag, err := fetchLatest(context.Background(), http.DefaultClient)
	if err != nil || tag != "v4.2.0" {
		t.Fatalf("fetchLatest = %q, %v; want v4.2.0, nil", tag, err)
	}
}

func TestFetchLatestNon200(t *testing.T) {
	isolate(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	APIBase = srv.URL

	if _, err := fetchLatest(context.Background(), http.DefaultClient); err == nil {
		t.Fatal("expected an error on 403")
	}
}

func TestCacheRoundTrip(t *testing.T) {
	isolate(t)
	want := cache{CheckedAt: time.Now().Truncate(time.Second), Latest: "v1.2.3"}
	writeCache(want)

	got, ok := readCache()
	if !ok || !got.CheckedAt.Equal(want.CheckedAt) || got.Latest != want.Latest {
		t.Fatalf("readCache = %+v,%v; want %+v", got, ok, want)
	}

	// No torn temp file left behind.
	p, _ := cachePath()
	if _, err := os.Stat(p + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("stray %s", filepath.Base(p)+".tmp")
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
