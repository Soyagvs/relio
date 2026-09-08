package ghrelease

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func withServer(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	APIBase = srv.URL
	t.Cleanup(func() {
		srv.Close()
		APIBase = "https://api.github.com"
	})
	return srv
}

func TestCreateSuccess(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	var gotBody map[string]any

	srv := withServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotAuth = r.Method, r.URL.Path, r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"html_url":"https://github.com/acme/relio/releases/tag/v1.2.0"}`)
	})
	_ = srv

	url, err := Create(context.Background(), nil, "tok123", Options{
		Repo: "acme/relio",
		Tag:  "v1.2.0",
		Name: "v1.2.0",
		Body: "### Added\n\n- A thing",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if url != "https://github.com/acme/relio/releases/tag/v1.2.0" {
		t.Errorf("html_url = %q", url)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/repos/acme/relio/releases" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAuth != "Bearer tok123" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotBody["tag_name"] != "v1.2.0" || gotBody["name"] != "v1.2.0" {
		t.Errorf("tag/name in body = %+v", gotBody)
	}
	if gotBody["body"] != "### Added\n\n- A thing" {
		t.Errorf("body = %q", gotBody["body"])
	}
	if gotBody["draft"] != false {
		t.Errorf("draft = %v, want false", gotBody["draft"])
	}
	if gotBody["prerelease"] != false {
		t.Errorf("prerelease = %v, want false", gotBody["prerelease"])
	}
	if gotBody["make_latest"] != "true" {
		t.Errorf("make_latest = %v, want \"true\"", gotBody["make_latest"])
	}
}

func TestCreatePrereleaseOmitsMakeLatest(t *testing.T) {
	var gotBody map[string]any
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"html_url":"https://example.test/r"}`)
	})

	if _, err := Create(context.Background(), nil, "t", Options{
		Repo: "acme/relio", Tag: "v1.2.0-rc.1", Name: "v1.2.0-rc.1", Prerelease: true,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, ok := gotBody["make_latest"]; ok {
		t.Errorf("make_latest present for a prerelease: %+v", gotBody)
	}
	if gotBody["prerelease"] != true {
		t.Errorf("prerelease = %v, want true", gotBody["prerelease"])
	}
}

func TestCreateAlreadyExists(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"message":"Validation Failed","errors":[{"resource":"Release","code":"already_exists","field":"tag_name"}]}`)
	})

	_, err := Create(context.Background(), nil, "t", Options{Repo: "acme/relio", Tag: "v1.2.0", Name: "v1.2.0"})
	if !errors.Is(err, ErrReleaseExists) {
		t.Fatalf("err = %v, want ErrReleaseExists", err)
	}
	if !strings.Contains(err.Error(), "v1.2.0") {
		t.Errorf("error text %q should name the tag", err.Error())
	}
}

func TestCreateUnauthorized(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"message":"Bad credentials"}`)
	})

	_, err := Create(context.Background(), nil, "t", Options{Repo: "acme/relio", Tag: "v1", Name: "v1"})
	if err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "repo") {
		t.Errorf("err = %v, want a clear 401/`repo` scope message", err)
	}
}

func TestCreateRateLimited(t *testing.T) {
	reset := time.Now().Add(15 * time.Minute).Unix()
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", fmt.Sprint(reset))
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"message":"API rate limit exceeded"}`)
	})

	_, err := Create(context.Background(), nil, "t", Options{Repo: "acme/relio", Tag: "v1", Name: "v1"})
	if _, ok := err.(*RateLimitError); !ok {
		t.Fatalf("err = %v (%T), want *RateLimitError", err, err)
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("error text = %q", err.Error())
	}
}

func TestCreateOtherError(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "boom\nsecond line")
	})
	_, err := Create(context.Background(), nil, "t", Options{Repo: "acme/relio", Tag: "v1", Name: "v1"})
	if err == nil || !strings.Contains(err.Error(), "GitHub API 500") || !strings.Contains(err.Error(), "boom") {
		t.Errorf("err = %v", err)
	}
}

func TestAuthenticatedUser(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("path = %q, want /user", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer abc" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		fmt.Fprint(w, `{"login":"octocat"}`)
	})

	login, err := AuthenticatedUser(context.Background(), nil, "abc")
	if err != nil {
		t.Fatalf("AuthenticatedUser: %v", err)
	}
	if login != "octocat" {
		t.Errorf("login = %q", login)
	}
}

func TestAuthenticatedUserUnauthorized(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"message":"Bad credentials"}`)
	})
	if _, err := AuthenticatedUser(context.Background(), nil, "x"); err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("err = %v, want a 401 message", err)
	}
}

func TestParseRepo(t *testing.T) {
	ok := map[string]string{
		"git@github.com:Owner/Name.git":           "Owner/Name",
		"git@github.com:Owner/Name":               "Owner/Name",
		"https://github.com/Owner/Name.git":       "Owner/Name",
		"https://github.com/Owner/Name":           "Owner/Name",
		"ssh://git@github.com/Owner/Name.git":     "Owner/Name",
		"  https://github.com/Owner/Name.git  \n": "Owner/Name",
	}
	for in, want := range ok {
		got, err := ParseRepo(in)
		if err != nil {
			t.Errorf("ParseRepo(%q) error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseRepo(%q) = %q, want %q", in, got, want)
		}
	}

	bad := []string{
		"",
		"git@gitlab.com:Owner/Name.git",
		"https://example.com/Owner/Name",
		"https://github.com/OnlyOwner",
		"not a url",
	}
	for _, in := range bad {
		if got, err := ParseRepo(in); err == nil {
			t.Errorf("ParseRepo(%q) = %q, want error", in, got)
		}
	}
}

func TestTokenResolutionOrder(t *testing.T) {
	// Neutralise the `gh` fallback: an empty PATH makes the lookup fail.
	t.Setenv("PATH", "")

	t.Run("RELIO_GITHUB_TOKEN wins", func(t *testing.T) {
		t.Setenv("RELIO_GITHUB_TOKEN", "  relio-tok  ")
		t.Setenv("GITHUB_TOKEN", "gh-tok")
		t.Setenv("GH_TOKEN", "ght-tok")
		tok, src := Token()
		if tok != "relio-tok" || src != "RELIO_GITHUB_TOKEN" {
			t.Errorf("Token() = %q, %q", tok, src)
		}
	})

	t.Run("GITHUB_TOKEN before GH_TOKEN", func(t *testing.T) {
		t.Setenv("RELIO_GITHUB_TOKEN", "")
		t.Setenv("GITHUB_TOKEN", "gh-tok")
		t.Setenv("GH_TOKEN", "ght-tok")
		tok, src := Token()
		if tok != "gh-tok" || src != "GITHUB_TOKEN" {
			t.Errorf("Token() = %q, %q", tok, src)
		}
	})

	t.Run("GH_TOKEN last", func(t *testing.T) {
		t.Setenv("RELIO_GITHUB_TOKEN", "")
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GH_TOKEN", "ght-tok")
		tok, src := Token()
		if tok != "ght-tok" || src != "GH_TOKEN" {
			t.Errorf("Token() = %q, %q", tok, src)
		}
	})

	t.Run("nothing set", func(t *testing.T) {
		t.Setenv("RELIO_GITHUB_TOKEN", "")
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GH_TOKEN", "")
		tok, src := Token()
		if tok != "" || src != "" {
			t.Errorf("Token() = %q, %q, want empty", tok, src)
		}
	})
}
