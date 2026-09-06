// Package upload sends a file to a public file host and returns a URL.
package upload

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// userAgent — some hosts reject the stock Go and curl agents.
const userAgent = "relio (+https://github.com/soyagvs/relio)"

var client = &http.Client{Timeout: 45 * time.Second}

const (
	litterbox = "https://litterbox.catbox.moe/resources/internals/api.php"
	catbox    = "https://catbox.moe/user/api.php"
)

// Upload sends path to a host and returns a URL. It tries litterbox (temporary,
// 72h) first, then catbox (permanent) as a fallback. The error names every host
// that failed.
func Upload(path string) (string, error) {
	attempts := []struct {
		name   string
		host   string
		fields map[string]string
	}{
		{"litterbox", litterbox, map[string]string{"reqtype": "fileupload", "time": "72h"}},
		{"catbox", catbox, map[string]string{"reqtype": "fileupload"}},
	}

	var failed []string
	for _, a := range attempts {
		if url, err := post(a.host, "fileToUpload", path, a.fields); err == nil {
			return url, nil
		} else {
			failed = append(failed, a.name+" ("+oneLine(err)+")")
		}
	}
	return "", fmt.Errorf("all upload hosts failed: %s", strings.Join(failed, "; "))
}

// ToPlain POSTs the file to a host that answers with the URL as its plain-text
// body, under the "file" field. Exposed for testing.
func ToPlain(host, path string) (string, error) {
	return post(host, "file", path, nil)
}

func post(host, fileField, path string, fields map[string]string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	part, err := mw.CreateFormFile(fileField, filepath.Base(path))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, host, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	text := strings.TrimSpace(string(raw))
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("%d: %s", resp.StatusCode, firstLine(text, resp.Status))
	}
	if !strings.HasPrefix(text, "http") {
		return "", fmt.Errorf("unexpected response: %q", firstLine(text, "empty"))
	}
	return text, nil
}

func firstLine(s, fallback string) string {
	s = strings.TrimSpace(strings.SplitN(s, "\n", 2)[0])
	if s == "" {
		return fallback
	}
	if len(s) > 120 {
		s = s[:120] + "…"
	}
	return s
}

func oneLine(err error) string { return firstLine(err.Error(), "error") }
