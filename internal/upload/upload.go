// Package upload sends a file to a public file host and returns a URL.
package upload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/soyagvs/relio/internal/i18n"
)

// userAgent — some hosts reject the stock Go and curl agents.
const userAgent = "relio (+https://github.com/soyagvs/relio)"

var client = &http.Client{Timeout: 45 * time.Second}

// litterbox and catbox are vars (not consts) so tests can point them at an
// httptest.Server, matching the pattern already used by
// ghrelease.DeviceFlowBase / ghrelease.APIBase.
var (
	litterbox = "https://litterbox.catbox.moe/resources/internals/api.php"
	catbox    = "https://catbox.moe/user/api.php"
)

// maxAttempts is how many times Upload tries a single host before falling
// through to the next one (or giving up, for the last host). A small fixed
// count is enough here — this isn't a configurable retry policy.
const maxAttempts = 3

// retryBackoff is the base delay between retry attempts against the same
// host; the actual delay grows linearly with the attempt number
// (attempt * retryBackoff). It's a package var so tests can shrink it and
// avoid burning real wall-clock time, matching ui.introFrameDelay.
var retryBackoff = 2 * time.Second

// perHostTimeout bounds the time Upload spends on a single host — every
// attempt and backoff wait against it combined — regardless of what deadline
// (if any) the caller's own context carries. Without this, a caller passing
// context.Background() (as cmd/image.go does) combined with maxAttempts
// retries at up to 45s each could hang for minutes with nothing to stop it.
// The bound is applied per host, not once across the whole call: a single
// global budget would let a slow-but-responsive first host alone consume it,
// starving the second host of the real chance the fallback exists to give
// it. It's a package var so tests can shrink it; a caller-supplied shorter
// deadline still wins, since context.WithTimeout always honours the earlier
// of the two.
//
// It must stay large enough to let all maxAttempts actually run even if
// every single attempt takes the full client.Timeout, or it silently cuts
// retries short exactly under the slow-but-alive conditions retrying exists
// to help with: worst case is maxAttempts*client.Timeout plus the backoff
// between attempts (1*retryBackoff + 2*retryBackoff for maxAttempts=3) =
// 3*45s + 3*2s = 141s. 150s leaves a small margin.
// TestPerHostTimeoutAccommodatesMaxAttempts enforces this relationship, so
// bumping maxAttempts or client.Timeout without also raising this value
// fails the build.
var perHostTimeout = 150 * time.Second

// Upload sends path to a host and returns a URL. It tries litterbox (temporary,
// 72h) first, then catbox (permanent) as a fallback, retrying each host up to
// maxAttempts times with a backoff delay between attempts, each host bounded
// by its own perHostTimeout. progress receives one line per retry (nothing
// for a host's first attempt); a nil progress is safe and simply discards
// the lines. Cancelling ctx itself (as opposed to one host's own budget
// expiring) aborts immediately and short-circuits the remaining hosts. The
// error names every host that failed, including one that failed because its
// own per-host budget ran out.
func Upload(ctx context.Context, path string, progress io.Writer) (string, error) {
	if progress == nil {
		progress = io.Discard
	}
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
		hostCtx, cancel := context.WithTimeout(ctx, perHostTimeout)
		url, err := postWithRetry(hostCtx, progress, a.name, a.host, "fileToUpload", path, a.fields)
		cancel()
		if err == nil {
			return url, nil
		}
		// ctx (not hostCtx, which cancel() just ended regardless) tells us
		// whether the caller itself is done -- as opposed to just this
		// host's own budget running out, which must not abort the loop.
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		failed = append(failed, a.name+" ("+oneLine(err)+")")
	}
	return "", fmt.Errorf("all upload hosts failed: %s", strings.Join(failed, "; "))
}

// postWithRetry calls post against a single host up to maxAttempts times,
// printing a progress line to progress and sleeping a cancellable backoff
// between attempts. It returns the last error once attempts are exhausted, or
// ctx's error if ctx is done during a backoff wait.
//
// Accepted tradeoff: the retried request (post) is not idempotent, and
// litterbox/catbox offer no idempotency key. If a request's response is lost
// after the server already stored the file (a timeout on the read, say), the
// next retry can create a duplicate, orphaned upload. Retrying is still
// deliberately on by default here because these hosts' own transient
// failures (observed as an ordinary non-2xx response) are the common case
// this exists to smooth over, and refusing to retry after any response would
// defeat that -- the residual duplicate risk is a known, accepted tradeoff of
// retrying against a non-idempotent third-party API with no better primitive
// to reach for.
func postWithRetry(ctx context.Context, progress io.Writer, name, host, fileField, path string, fields map[string]string) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			// A failure writing the progress line (a broken pipe, say) is
			// cosmetic -- it must not abort the substantive network
			// operation, so its error is deliberately discarded.
			_, _ = fmt.Fprintln(progress, i18n.T(i18n.UploadRetrying, name, attempt, maxAttempts))
		}
		url, err := post(ctx, host, fileField, path, fields)
		if err == nil {
			return url, nil
		}
		lastErr = err
		if attempt < maxAttempts {
			if serr := sleepBackoff(ctx, attempt); serr != nil {
				return "", serr
			}
		}
	}
	return "", lastErr
}

// sleepBackoff waits attempt*retryBackoff, or returns ctx's error early if
// ctx is done first. It uses time.NewTimer selected against ctx.Done(),
// copying PollForToken's shape (internal/ghrelease/deviceflow.go), not a bare
// time.Sleep, so the wait is both testable and cancellable.
func sleepBackoff(ctx context.Context, attempt int) error {
	timer := time.NewTimer(time.Duration(attempt) * retryBackoff)
	select {
	case <-ctx.Done():
		timer.Stop()
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// ToPlain POSTs the file to a host that answers with the URL as its plain-text
// body, under the "file" field. Exposed for testing.
func ToPlain(host, path string) (string, error) {
	return post(context.Background(), host, "file", path, nil)
}

func post(ctx context.Context, host, fileField, path string, fields map[string]string) (string, error) {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, host, &body)
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
