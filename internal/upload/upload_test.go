package upload

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "card.png")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestToPlain(t *testing.T) {
	path := writeTemp(t, "PNGDATA")

	var gotName, gotBody, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		f, h, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		defer f.Close()
		gotName = h.Filename
		b, _ := io.ReadAll(f)
		gotBody = string(b)
		w.Write([]byte("  https://x.example/abc.png\n"))
	}))
	defer srv.Close()

	url, err := ToPlain(srv.URL, path)
	if err != nil {
		t.Fatalf("ToPlain: %v", err)
	}
	if url != "https://x.example/abc.png" {
		t.Errorf("url = %q", url)
	}
	if gotName != "card.png" || gotBody != "PNGDATA" {
		t.Errorf("got name=%q body=%q", gotName, gotBody)
	}
	if !strings.Contains(gotUA, "relio") {
		t.Errorf("User-Agent = %q", gotUA)
	}
}

func TestToPlainServerError(t *testing.T) {
	path := writeTemp(t, "x")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "uploads disabled", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	_, err := ToPlain(srv.URL, path)
	if err == nil || !strings.Contains(err.Error(), "uploads disabled") {
		t.Fatalf("err = %v", err)
	}
}

func TestToPlainRejectsNonURLBody(t *testing.T) {
	path := writeTemp(t, "x")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html>nope</html>"))
	}))
	defer srv.Close()

	if _, err := ToPlain(srv.URL, path); err == nil {
		t.Fatal("expected error for non-URL body")
	}
}
