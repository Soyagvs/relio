package ghrelease

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DeviceFlowBase is the root of GitHub's OAuth Device Flow endpoints. These
// live on github.com, not api.github.com (see APIBase in ghrelease.go), so
// they get their own overridable var for tests.
var DeviceFlowBase = "https://github.com"

// DeviceCode is the response from the initial device authorization request.
type DeviceCode struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	ExpiresIn       int
	Interval        int
}

// Sentinel errors returned by PollForToken while the user has not yet
// finished (or has rejected) the device authorization on GitHub's side.
// ErrAuthorizationPending and ErrSlowDown are internal poll outcomes that
// PollForToken handles by looping; they are exported for completeness/testing
// but never returned to PollForToken's caller. ErrExpiredToken and
// ErrAccessDenied are terminal and are returned to the caller.
var (
	ErrAuthorizationPending = errors.New("authorization pending")
	ErrSlowDown             = errors.New("slow down")
	ErrExpiredToken         = errors.New("device code expired")
	ErrAccessDenied         = errors.New("access denied")
)

// deviceCodeResponse mirrors the JSON body GitHub sends for a successful
// device code request.
type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// githubErrorResponse mirrors the JSON body GitHub sends for an error, both
// for the device code request and for the token poll.
type githubErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Interval         int    `json:"interval"`
}

// RequestDeviceCode starts the OAuth Device Flow by asking GitHub for a
// device code, user code and verification URL to show the user. client may
// be nil.
func RequestDeviceCode(ctx context.Context, client *http.Client, clientID string) (DeviceCode, error) {
	if client == nil {
		client = http.DefaultClient
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", "repo")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, DeviceFlowBase+"/login/device/code", strings.NewReader(form.Encode()))
	if err != nil {
		return DeviceCode{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return DeviceCode{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))

	if resp.StatusCode/100 != 2 {
		return DeviceCode{}, fmt.Errorf("GitHub API %d: %s", resp.StatusCode, firstLine(body))
	}

	if ghErr := decodeGitHubError(body); ghErr != nil {
		return DeviceCode{}, ghErr
	}

	var out deviceCodeResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return DeviceCode{}, fmt.Errorf("decoding GitHub response: %w", err)
	}

	return DeviceCode{
		DeviceCode:      out.DeviceCode,
		UserCode:        out.UserCode,
		VerificationURI: out.VerificationURI,
		ExpiresIn:       out.ExpiresIn,
		Interval:        out.Interval,
	}, nil
}

// PollForToken polls GitHub's OAuth token endpoint on the given interval
// until the user completes (or rejects) the device authorization, the device
// code expires, or ctx is done. Callers should give ctx a deadline based on
// the DeviceCode's ExpiresIn. client may be nil.
func PollForToken(ctx context.Context, client *http.Client, clientID, deviceCode string, interval time.Duration) (accessToken string, err error) {
	if client == nil {
		client = http.DefaultClient
	}

	for {
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}

		token, nextInterval, pollErr := pollOnce(ctx, client, clientID, deviceCode)
		if pollErr != nil {
			switch {
			case errors.Is(pollErr, ErrAuthorizationPending):
				continue
			case errors.Is(pollErr, ErrSlowDown):
				if nextInterval > 0 {
					interval = time.Duration(nextInterval) * time.Second
				} else {
					interval += 5 * time.Second
				}
				continue
			default:
				return "", pollErr
			}
		}
		return token, nil
	}
}

// pollOnce makes a single request to GitHub's access token endpoint. It
// returns the access token on success, or an error (possibly wrapping
// ErrAuthorizationPending, ErrSlowDown, ErrExpiredToken or ErrAccessDenied)
// otherwise. nextInterval carries GitHub's requested new interval on a
// slow_down response, when present.
func pollOnce(ctx context.Context, client *http.Client, clientID, deviceCode string) (token string, nextInterval int, err error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("device_code", deviceCode)
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, DeviceFlowBase+"/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))

	if resp.StatusCode/100 != 2 {
		return "", 0, fmt.Errorf("GitHub API %d: %s", resp.StatusCode, firstLine(body))
	}

	var out struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		Interval    int    `json:"interval"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", 0, fmt.Errorf("decoding GitHub response: %w", err)
	}

	switch out.Error {
	case "":
		if out.AccessToken == "" {
			return "", 0, errors.New("GitHub response had neither access_token nor error")
		}
		return out.AccessToken, 0, nil
	case "authorization_pending":
		return "", 0, ErrAuthorizationPending
	case "slow_down":
		return "", out.Interval, ErrSlowDown
	case "expired_token":
		return "", 0, fmt.Errorf("%w", ErrExpiredToken)
	case "access_denied":
		return "", 0, fmt.Errorf("%w", ErrAccessDenied)
	default:
		return "", 0, fmt.Errorf("GitHub OAuth error %q", out.Error)
	}
}

// decodeGitHubError reports a non-nil error when body carries a GitHub
// error/error_description payload (used by the device code endpoint).
func decodeGitHubError(body []byte) error {
	var e githubErrorResponse
	if json.Unmarshal(body, &e) != nil {
		return nil
	}
	if e.Error == "" {
		return nil
	}
	if e.ErrorDescription != "" {
		return fmt.Errorf("GitHub OAuth error %q: %s", e.Error, e.ErrorDescription)
	}
	return fmt.Errorf("GitHub OAuth error %q", e.Error)
}
