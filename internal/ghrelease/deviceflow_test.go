package ghrelease

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// withDeviceFlowServer swaps DeviceFlowBase to point at an httptest.Server for
// the duration of the test, restoring it via t.Cleanup. Mirrors withServer's
// idiom in ghrelease_test.go, but for the github.com (not api.github.com)
// Device Flow endpoints.
func withDeviceFlowServer(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	DeviceFlowBase = srv.URL
	t.Cleanup(func() {
		srv.Close()
		DeviceFlowBase = "https://github.com"
	})
	return srv
}

func TestRequestDeviceCodeSuccess(t *testing.T) {
	var gotMethod, gotPath, gotAccept, gotContentType, gotUserAgent string
	var gotBody string

	withDeviceFlowServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotAccept = r.Header.Get("Accept")
		gotContentType = r.Header.Get("Content-Type")
		gotUserAgent = r.Header.Get("User-Agent")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"device_code":"dc123","user_code":"ABCD-1234","verification_uri":"https://github.com/login/device","expires_in":900,"interval":5}`)
	})

	dc, err := RequestDeviceCode(context.Background(), nil, "client-abc")
	if err != nil {
		t.Fatalf("RequestDeviceCode: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/login/device/code" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q", gotContentType)
	}
	if gotUserAgent != "relio" {
		t.Errorf("User-Agent = %q, want relio", gotUserAgent)
	}
	form, perr := url.ParseQuery(gotBody)
	if perr != nil {
		t.Fatalf("parsing request body %q: %v", gotBody, perr)
	}
	if form.Get("client_id") != "client-abc" {
		t.Errorf("client_id = %q", form.Get("client_id"))
	}
	if form.Get("scope") != "repo" {
		t.Errorf("scope = %q, want repo", form.Get("scope"))
	}

	if dc.DeviceCode != "dc123" {
		t.Errorf("DeviceCode = %q", dc.DeviceCode)
	}
	if dc.UserCode != "ABCD-1234" {
		t.Errorf("UserCode = %q", dc.UserCode)
	}
	if dc.VerificationURI != "https://github.com/login/device" {
		t.Errorf("VerificationURI = %q", dc.VerificationURI)
	}
	if dc.ExpiresIn != 900 {
		t.Errorf("ExpiresIn = %d, want 900", dc.ExpiresIn)
	}
	if dc.Interval != 5 {
		t.Errorf("Interval = %d, want 5", dc.Interval)
	}
}

func TestRequestDeviceCodeGitHubError(t *testing.T) {
	withDeviceFlowServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"error":"invalid_client","error_description":"The client_id is not valid."}`)
	})

	_, err := RequestDeviceCode(context.Background(), nil, "bad-client")
	if err == nil {
		t.Fatal("RequestDeviceCode: want error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid_client") && !strings.Contains(err.Error(), "not valid") {
		t.Errorf("error text %q should mention the GitHub error description", err.Error())
	}
}

func TestRequestDeviceCodeHTTPError(t *testing.T) {
	withDeviceFlowServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "boom")
	})

	_, err := RequestDeviceCode(context.Background(), nil, "client-abc")
	if err == nil {
		t.Fatal("RequestDeviceCode: want error, got nil")
	}
}

func TestPollForTokenSuccessFirstTry(t *testing.T) {
	var calls int32
	withDeviceFlowServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.URL.Path != "/login/oauth/access_token" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q", r.Header.Get("Accept"))
		}
		if r.Header.Get("User-Agent") != "relio" {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		raw, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(raw))
		if form.Get("client_id") != "client-abc" {
			t.Errorf("client_id = %q", form.Get("client_id"))
		}
		if form.Get("device_code") != "dc123" {
			t.Errorf("device_code = %q", form.Get("device_code"))
		}
		if form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" {
			t.Errorf("grant_type = %q", form.Get("grant_type"))
		}
		_, _ = fmt.Fprint(w, `{"access_token":"tok-xyz","token_type":"bearer","scope":"repo"}`)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	tok, err := PollForToken(ctx, nil, "client-abc", "dc123", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("PollForToken: %v", err)
	}
	if tok != "tok-xyz" {
		t.Errorf("token = %q", tok)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("calls = %d, want 1", got)
	}
}

func TestPollForTokenAuthorizationPendingThenSuccess(t *testing.T) {
	var calls int32
	withDeviceFlowServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n <= 2 {
			_, _ = fmt.Fprint(w, `{"error":"authorization_pending"}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"access_token":"tok-final","token_type":"bearer","scope":"repo"}`)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	tok, err := PollForToken(ctx, nil, "client-abc", "dc123", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("PollForToken: %v", err)
	}
	if tok != "tok-final" {
		t.Errorf("token = %q", tok)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("calls = %d, want 3", got)
	}
}

func TestPollForTokenSlowDownIncreasesInterval(t *testing.T) {
	var calls int32
	var callTimes []time.Time

	// The server reports a new (explicit) interval of 1s on its slow_down
	// response, per GitHub's documented (optional) behavior. We assert the
	// gap between the slow_down call and the following call reflects that
	// new interval, rather than the tiny base interval used beforehand -
	// this keeps the test fast (~1s) and non-flaky (no reliance on the
	// interval-absent branch, which per spec adds a full 5s).
	withDeviceFlowServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		callTimes = append(callTimes, time.Now())
		if n == 1 {
			_, _ = fmt.Fprint(w, `{"error":"slow_down","interval":1}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"access_token":"tok-slow","token_type":"bearer","scope":"repo"}`)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	tok, err := PollForToken(ctx, nil, "client-abc", "dc123", 20*time.Millisecond)
	if err != nil {
		t.Fatalf("PollForToken: %v", err)
	}
	if tok != "tok-slow" {
		t.Errorf("token = %q", tok)
	}
	if len(callTimes) != 2 {
		t.Fatalf("calls = %d, want 2", len(callTimes))
	}
	// The gap after the slow_down response (call 1 -> call 2) must be
	// noticeably larger than the base 20ms interval used before it, since
	// the server told us to wait 1s instead.
	gapBeforeSlowDown := callTimes[0].Sub(start)
	gapAfterSlowDown := callTimes[1].Sub(callTimes[0])
	if gapAfterSlowDown <= gapBeforeSlowDown {
		t.Errorf("gap after slow_down (%v) should exceed the base interval gap (%v)", gapAfterSlowDown, gapBeforeSlowDown)
	}
	if gapAfterSlowDown < 900*time.Millisecond {
		t.Errorf("gap after slow_down = %v, want at least ~1s (the server's new interval)", gapAfterSlowDown)
	}
}

func TestPollForTokenExpiredToken(t *testing.T) {
	withDeviceFlowServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"error":"expired_token"}`)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := PollForToken(ctx, nil, "client-abc", "dc123", 10*time.Millisecond)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("err = %v, want ErrExpiredToken", err)
	}
}

func TestPollForTokenAccessDenied(t *testing.T) {
	withDeviceFlowServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"error":"access_denied"}`)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := PollForToken(ctx, nil, "client-abc", "dc123", 10*time.Millisecond)
	if !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("err = %v, want ErrAccessDenied", err)
	}
}

func TestPollForTokenContextCancellationDuringSleep(t *testing.T) {
	withDeviceFlowServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Should never be reached: the ctx deadline is shorter than the poll
		// interval, so PollForToken must return before ever making a request.
		t.Error("server hit: PollForToken should have returned on ctx cancellation before polling")
		_, _ = fmt.Fprint(w, `{"error":"authorization_pending"}`)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := PollForToken(ctx, nil, "client-abc", "dc123", 5*time.Second)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed > 1*time.Second {
		t.Errorf("PollForToken took %v to return after ctx cancellation, want well under 1s", elapsed)
	}
}
