package upload

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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

// withRetryBackoff sets retryBackoff for the duration of a test and restores
// it afterward, so retry tests don't burn real wall-clock time.
func withRetryBackoff(t *testing.T, d time.Duration) {
	t.Helper()
	saved := retryBackoff
	retryBackoff = d
	t.Cleanup(func() { retryBackoff = saved })
}

// withHosts points litterbox and catbox at the given URLs for the duration of
// a test and restores the real hosts afterward.
func withHosts(t *testing.T, litter, cat string) {
	t.Helper()
	savedLitter, savedCat := litterbox, catbox
	litterbox, catbox = litter, cat
	t.Cleanup(func() {
		litterbox, catbox = savedLitter, savedCat
	})
}

func TestUploadRetriesOnFailureThenSucceeds(t *testing.T) {
	withRetryBackoff(t, time.Millisecond)
	path := writeTemp(t, "PNGDATA")

	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&requests, 1)
		if n < 3 {
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("https://litter.example/ok.png"))
	}))
	defer srv.Close()
	withHosts(t, srv.URL, "http://catbox.invalid")

	var progress strings.Builder
	url, err := Upload(context.Background(), path, &progress)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if url != "https://litter.example/ok.png" {
		t.Errorf("url = %q", url)
	}
	if got := atomic.LoadInt32(&requests); got != 3 {
		t.Errorf("requests = %d, want 3", got)
	}
	out := progress.String()
	if !strings.Contains(out, "litterbox") || !strings.Contains(out, "2") || !strings.Contains(out, "3") {
		t.Errorf("progress output = %q, want retry lines mentioning litterbox and attempts 2 and 3", out)
	}
	if n := strings.Count(out, "\n"); n != 2 {
		t.Errorf("progress printed %d lines, want 2 (one per retry, none for the first attempt)", n)
	}
}

func TestUploadFallsBackToCatboxAfterLitterboxExhausted(t *testing.T) {
	withRetryBackoff(t, time.Millisecond)
	path := writeTemp(t, "PNGDATA")

	var litterHits, catHits int32
	litterSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&litterHits, 1)
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer litterSrv.Close()
	catSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&catHits, 1)
		if n < 2 {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("https://catbox.example/ok.png"))
	}))
	defer catSrv.Close()
	withHosts(t, litterSrv.URL, catSrv.URL)

	var progress strings.Builder
	url, err := Upload(context.Background(), path, &progress)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if url != "https://catbox.example/ok.png" {
		t.Errorf("url = %q", url)
	}
	if got := atomic.LoadInt32(&litterHits); got != 3 {
		t.Errorf("litterbox hits = %d, want 3 (all attempts exhausted before falling back)", got)
	}
	if got := atomic.LoadInt32(&catHits); got != 2 {
		t.Errorf("catbox hits = %d, want 2", got)
	}
	out := progress.String()
	if !strings.Contains(out, "catbox") {
		t.Errorf("progress output = %q, want a retry line mentioning catbox", out)
	}
}

func TestUploadGivesUpAfterAllHostsExhausted(t *testing.T) {
	withRetryBackoff(t, time.Millisecond)
	path := writeTemp(t, "PNGDATA")

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer failing.Close()
	withHosts(t, failing.URL, failing.URL)

	_, err := Upload(context.Background(), path, io.Discard)
	if err == nil {
		t.Fatal("expected error when every host is exhausted")
	}
	if !strings.Contains(err.Error(), "litterbox") || !strings.Contains(err.Error(), "catbox") {
		t.Errorf("err = %v, want it to name both litterbox and catbox", err)
	}
}

func TestUploadNilProgressWriterIsSafe(t *testing.T) {
	withRetryBackoff(t, time.Millisecond)
	path := writeTemp(t, "PNGDATA")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("https://litter.example/ok.png"))
	}))
	defer srv.Close()
	withHosts(t, srv.URL, "http://catbox.invalid")

	if _, err := Upload(context.Background(), path, nil); err != nil {
		t.Fatalf("Upload with nil progress writer: %v", err)
	}
}

func TestUploadReturnsPromptlyOnContextCancellationDuringBackoff(t *testing.T) {
	withRetryBackoff(t, 5*time.Second)
	path := writeTemp(t, "PNGDATA")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	withHosts(t, srv.URL, "http://catbox.invalid")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := Upload(ctx, path, io.Discard)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed > 1*time.Second {
		t.Errorf("Upload took %v to return after ctx cancellation, want well under 1s", elapsed)
	}
}

// withClient sets client for the duration of a test and restores it
// afterward, so a test can shrink the per-attempt HTTP timeout without
// touching the real 45s production default.
func withClient(t *testing.T, c *http.Client) {
	t.Helper()
	saved := client
	client = c
	t.Cleanup(func() { client = saved })
}

// TestPerHostTimeoutAccommodatesMaxAttempts is an invariant test, not a
// timing test: it guards against perHostTimeout silently drifting below
// what maxAttempts actually needs at the worst case (every attempt taking
// the full client.Timeout) -- exactly the CRITICAL bug this fixes, where a
// too-small perHostTimeout cut retries short after roughly one attempt
// precisely under the slow-but-alive conditions retrying exists to help
// with. Runs instantly; it never waits on a real clock.
func TestPerHostTimeoutAccommodatesMaxAttempts(t *testing.T) {
	var backoffSum time.Duration
	for attempt := 1; attempt < maxAttempts; attempt++ {
		backoffSum += time.Duration(attempt) * retryBackoff
	}
	worstCase := time.Duration(maxAttempts)*client.Timeout + backoffSum
	if perHostTimeout < worstCase {
		t.Errorf("perHostTimeout = %v, want at least %v (maxAttempts=%d * client.Timeout=%v + backoff=%v) so a slow-but-alive host actually gets all its retries",
			perHostTimeout, worstCase, maxAttempts, client.Timeout, backoffSum)
	}
}

// TestUploadSucceedsOnFinalAttemptEvenWhenEarlyAttemptsApproachClientTimeout
// is the behavioral counterpart to TestPerHostTimeoutAccommodatesMaxAttempts:
// a host that is merely slow (each failed attempt takes close to the full
// per-attempt client timeout) must still get to try maxAttempts times within
// its per-host budget, succeeding on the last one -- the scenario the
// CRITICAL finding showed the previous 60s perHostTimeout cut short after
// roughly one attempt. Scaled to milliseconds so it runs fast; the shrunk
// perHostTimeout mirrors the same formula the production default now uses.
func TestUploadSucceedsOnFinalAttemptEvenWhenEarlyAttemptsApproachClientTimeout(t *testing.T) {
	shrunkClientTimeout := 40 * time.Millisecond
	withClient(t, &http.Client{Timeout: shrunkClientTimeout})
	withRetryBackoff(t, 5*time.Millisecond)
	withPerHostTimeout(t, 3*shrunkClientTimeout+(1+2)*5*time.Millisecond+20*time.Millisecond) // + margin
	path := writeTemp(t, "PNGDATA")

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n < 3 {
			time.Sleep(shrunkClientTimeout + 10*time.Millisecond) // exceeds the client timeout -> this attempt fails
			return
		}
		_, _ = w.Write([]byte("https://litter.example/ok.png"))
	}))
	defer srv.Close()
	withHosts(t, srv.URL, "http://catbox.invalid")

	url, err := Upload(context.Background(), path, io.Discard)
	if err != nil {
		t.Fatalf("Upload: %v, want the 3rd attempt to succeed within the per-host budget", err)
	}
	if url != "https://litter.example/ok.png" {
		t.Errorf("url = %q", url)
	}
	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Errorf("hits = %d, want 3 (all attempts must get a chance within budget)", got)
	}
}

// withPerHostTimeout sets perHostTimeout for the duration of a test and
// restores it afterward, so the bound can be shrunk without burning real
// wall-clock time.
func withPerHostTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	saved := perHostTimeout
	perHostTimeout = d
	t.Cleanup(func() { perHostTimeout = saved })
}

// TestUploadBoundsTotalTimeEvenWithBackgroundContext guards against Upload
// hanging for minutes when the caller passes a context with no deadline of
// its own (cmd/image.go currently calls it with context.Background()) --
// retrying every host up to maxAttempts times at up to 45s per attempt could
// otherwise take several minutes with nothing to stop it.
func TestUploadBoundsTotalTimeEvenWithBackgroundContext(t *testing.T) {
	withRetryBackoff(t, 5*time.Second)
	withPerHostTimeout(t, 20*time.Millisecond)
	path := writeTemp(t, "PNGDATA")

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer failing.Close()
	withHosts(t, failing.URL, failing.URL)

	start := time.Now()
	_, err := Upload(context.Background(), path, io.Discard)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error when both hosts exhaust their per-host budget")
	}
	if elapsed > 1*time.Second {
		t.Errorf("Upload took %v with context.Background() and no caller deadline, want well under 1s given perHostTimeout", elapsed)
	}
}

// TestUploadCallerDeadlineShorterThanPerHostTimeoutWins guards against the
// internal perHostTimeout silently overriding a shorter deadline the caller
// already supplied.
func TestUploadCallerDeadlineShorterThanPerHostTimeoutWins(t *testing.T) {
	withRetryBackoff(t, 5*time.Second)
	withPerHostTimeout(t, time.Minute) // deliberately much longer than the caller's own deadline below
	path := writeTemp(t, "PNGDATA")

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer failing.Close()
	withHosts(t, failing.URL, failing.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := Upload(ctx, path, io.Discard)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed > 1*time.Second {
		t.Errorf("Upload took %v, want the caller's shorter 20ms deadline to win over the 1-minute perHostTimeout", elapsed)
	}
}

// TestUploadPerHostBudgetDoesNotStarveFallback is the regression test for
// the CRITICAL finding: a single global time budget shared across both
// hosts let a slow-but-responsive first host alone consume the whole
// budget, so catbox never got a real chance -- defeating the fallback the
// retry feature exists to provide. With a per-host budget, litterbox
// exhausting its own share must still leave catbox free to try (and here,
// succeed).
func TestUploadPerHostBudgetDoesNotStarveFallback(t *testing.T) {
	withRetryBackoff(t, 5*time.Second) // long enough that only the per-host deadline (not a retry) explains litterbox stopping
	withPerHostTimeout(t, 30*time.Millisecond)
	path := writeTemp(t, "PNGDATA")

	// litterbox hangs past its own per-host budget on every request. block
	// must close before litterSrv.Close() runs (Close waits for in-flight
	// handlers to return), so this defer is declared after litterSrv's —
	// defers run LIFO, so close(block) fires first.
	block := make(chan struct{})
	litterSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
	}))
	defer litterSrv.Close()
	defer close(block)

	catSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("https://catbox.example/ok.png"))
	}))
	defer catSrv.Close()
	withHosts(t, litterSrv.URL, catSrv.URL)

	url, err := Upload(context.Background(), path, io.Discard)
	if err != nil {
		t.Fatalf("Upload: %v, want catbox to still succeed despite litterbox exhausting its own budget", err)
	}
	if url != "https://catbox.example/ok.png" {
		t.Errorf("url = %q", url)
	}
}

// TestUploadBothHostsExhaustedNamesBothInError guards against a per-host
// timeout swallowing the real failure information: when every host exhausts
// its own budget, the final error must still name every host (as it already
// does for an ordinary non-timeout failure), not just report a bare
// "context deadline exceeded" with no indication of what was tried.
func TestUploadBothHostsExhaustedNamesBothInError(t *testing.T) {
	withRetryBackoff(t, 5*time.Second)
	withPerHostTimeout(t, 20*time.Millisecond)
	path := writeTemp(t, "PNGDATA")

	// block must close before hanging.Close() runs; defers run LIFO, so
	// declare hanging.Close() first.
	block := make(chan struct{})
	hanging := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
	}))
	defer hanging.Close()
	defer close(block)
	withHosts(t, hanging.URL, hanging.URL)

	_, err := Upload(context.Background(), path, io.Discard)
	if err == nil {
		t.Fatal("expected an error when every host exhausts its own budget")
	}
	if !strings.Contains(err.Error(), "litterbox") || !strings.Contains(err.Error(), "catbox") {
		t.Errorf("err = %v, want it to name both litterbox and catbox even though both timed out", err)
	}
}

// TestUploadProgressWriteFailureDoesNotAbortRetry guards against a cosmetic
// I/O error on the progress writer (e.g. a broken stdout pipe) turning into
// a hard abort of the substantive network operation.
func TestUploadProgressWriteFailureDoesNotAbortRetry(t *testing.T) {
	withRetryBackoff(t, time.Millisecond)
	path := writeTemp(t, "PNGDATA")

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n < 2 {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("https://litter.example/ok.png"))
	}))
	defer srv.Close()
	withHosts(t, srv.URL, "http://catbox.invalid")

	url, err := Upload(context.Background(), path, alwaysErrorWriter{})
	if err != nil {
		t.Fatalf("Upload: %v, want it to succeed despite the progress writer failing", err)
	}
	if url != "https://litter.example/ok.png" {
		t.Errorf("url = %q", url)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("requests = %d, want 2 (the retry must still have happened)", got)
	}
}

type alwaysErrorWriter struct{}

func (alwaysErrorWriter) Write([]byte) (int, error) {
	return 0, errors.New("broken pipe")
}

func TestUploadRetryProgressMessageFormat(t *testing.T) {
	withRetryBackoff(t, time.Millisecond)
	path := writeTemp(t, "PNGDATA")

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n < 2 {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("https://litter.example/ok.png"))
	}))
	defer srv.Close()
	withHosts(t, srv.URL, "http://catbox.invalid")

	var progress strings.Builder
	if _, err := Upload(context.Background(), path, &progress); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	line := strings.TrimSpace(progress.String())
	if !strings.Contains(line, "litterbox") {
		t.Errorf("progress line = %q, want it to name the host", line)
	}
	if !strings.Contains(line, strconv.Itoa(2)) || !strings.Contains(line, strconv.Itoa(maxAttempts)) {
		t.Errorf("progress line = %q, want it to show attempt 2 of %d", line, maxAttempts)
	}
}
