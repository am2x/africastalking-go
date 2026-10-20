package transport

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/am2x/africastalking-go/internal/testutil"
)

// reqBody is the payload every test sends, so a test can assert it survived
// being resent on a retry.
const reqBody = `{"username":"testuser","message":"Test SMS","phoneNumbers":["+254700000000"]}`

// fastClient is a client with retry delays collapsed so tests do not sleep.
func fastClient() *HTTPClient {
	c := New()
	c.BaseDelay = time.Nanosecond
	c.MaxDelay = time.Millisecond

	return c
}

func jsonReq(url string, retryable bool) Request {
	return Request{
		Method:      http.MethodPost,
		URL:         url,
		ContentType: ContentTypeJSON,
		UserAgent:   UserAgent,
		Body:        []byte(reqBody),
		Retryable:   retryable,
	}
}

// --- what Do sends ---

func TestDoSetsHeadersAndSendsBody(t *testing.T) {
	srv := testutil.NewServer(t, testutil.Respond(http.StatusOK, `{"ok":true}`))

	if _, err := fastClient().Do(context.Background(), jsonReq(srv.URL+"/messaging/bulk", false), "secret-key"); err != nil {
		t.Fatalf("Do() = %v", err)
	}

	got := srv.LastRequest(t)
	if got.Method != http.MethodPost {
		t.Errorf("Method = %q, want POST", got.Method)
	}
	if got.Path != "/messaging/bulk" {
		t.Errorf("Path = %q, want /messaging/bulk", got.Path)
	}
	if got.ContentType != ContentTypeJSON {
		t.Errorf("Content-Type = %q, want %q", got.ContentType, ContentTypeJSON)
	}
	if got.Accept != ContentTypeJSON {
		t.Errorf("Accept = %q, want %q", got.Accept, ContentTypeJSON)
	}
	if got.UserAgent != UserAgent {
		t.Errorf("User-Agent = %q, want %q", got.UserAgent, UserAgent)
	}
	if got.APIKey != "secret-key" {
		t.Errorf("apiKey = %q, want secret-key", got.APIKey)
	}
	if string(got.Body) != reqBody {
		t.Errorf("Body = %s", got.Body)
	}
}

func TestDoFormEncodedStillAcceptsJSON(t *testing.T) {
	srv := testutil.NewServer(t, testutil.Respond(http.StatusOK, `{}`))

	r := jsonReq(srv.URL+"/airtime/send", false)
	r.ContentType = ContentTypeFormEncoded
	if _, err := fastClient().Do(context.Background(), r, "k"); err != nil {
		t.Fatalf("Do() = %v", err)
	}

	got := srv.LastRequest(t)
	if got.ContentType != ContentTypeFormEncoded {
		t.Errorf("Content-Type = %q, want form encoded", got.ContentType)
	}
	if got.Accept != ContentTypeJSON {
		t.Errorf("Accept = %q, want json regardless of what was sent", got.Accept)
	}
}

// --- what Do returns ---

func TestDoReturnsResponseDetails(t *testing.T) {
	srv := testutil.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Request-Id", "abc123")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"created":true}`))
	}))

	resp, err := fastClient().Do(context.Background(), jsonReq(srv.URL, false), "secret-key")
	if err != nil {
		t.Fatalf("Do() = %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("StatusCode = %d, want 201", resp.StatusCode)
	}
	if string(resp.Body) != `{"created":true}` {
		t.Errorf("Body = %s", resp.Body)
	}
	if got := resp.Header.Get("X-Request-Id"); got != "abc123" {
		t.Errorf("Header = %v, want the response headers", resp.Header)
	}
	// the request headers must never come back: they carry the api key
	if resp.Header.Get("apiKey") != "" {
		t.Error("response Header leaked the apiKey request header")
	}
}

func TestDoDoesNotTreatNon2xxAsError(t *testing.T) {
	srv := testutil.NewServer(t, testutil.Respond(http.StatusBadRequest, `{"errMessage":"nope"}`))

	resp, err := fastClient().Do(context.Background(), jsonReq(srv.URL, false), "k")
	if err != nil {
		t.Fatalf("Do() = %v, want nil so the caller can map the status to an APIError", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", resp.StatusCode)
	}
}

// --- failure paths ---

func TestDoHonoursContextCancellation(t *testing.T) {
	srv := testutil.NewServer(t, testutil.Respond(http.StatusOK, `{}`))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := fastClient().Do(ctx, jsonReq(srv.URL, false), "k")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want it to wrap context.Canceled", err)
	}
}

func TestDoErrorsOnUnbuildableRequest(t *testing.T) {
	_, err := fastClient().Do(context.Background(), jsonReq("http://[::1]:namedport/", false), "k")
	if err == nil {
		t.Fatal("Do() = nil error, want a build failure")
	}
	if !strings.Contains(err.Error(), "build request") {
		t.Errorf("err = %q, want it to mention building the request", err)
	}
}

func TestDoErrorsWhenServerUnreachable(t *testing.T) {
	srv := testutil.NewServer(t, testutil.Respond(http.StatusOK, `{}`))
	url := srv.URL
	srv.Close() // nothing is listening on that port now

	_, err := fastClient().Do(context.Background(), jsonReq(url, false), "k")
	if err == nil {
		t.Fatal("Do() = nil error, want a connection failure")
	}
	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("err = %q, want it to mention the failed request", err)
	}
}

// --- retry behaviour ---

func TestDoDoesNotRetryWhenNotRetryable(t *testing.T) {
	srv := testutil.NewServer(t, testutil.Sequence(
		testutil.Respond(http.StatusServiceUnavailable, `{}`),
		testutil.Respond(http.StatusOK, `{"ok":true}`),
	))

	resp, err := fastClient().Do(context.Background(), jsonReq(srv.URL, false), "k")
	if err != nil {
		t.Fatalf("Do() = %v", err)
	}
	if srv.Count() != 1 {
		t.Errorf("server saw %d requests, want exactly 1 for a non-retryable request", srv.Count())
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("StatusCode = %d, want the first response returned as-is", resp.StatusCode)
	}
}

func TestDoRetriesTransientFailures(t *testing.T) {
	srv := testutil.NewServer(t, testutil.Sequence(
		testutil.Respond(http.StatusServiceUnavailable, `{}`),
		testutil.Respond(http.StatusTooManyRequests, `{}`),
		testutil.Respond(http.StatusOK, `{"ok":true}`),
	))

	resp, err := fastClient().Do(context.Background(), jsonReq(srv.URL, true), "k")
	if err != nil {
		t.Fatalf("Do() = %v", err)
	}
	if srv.Count() != 3 {
		t.Errorf("server saw %d requests, want 3", srv.Count())
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestDoStopsAfterMaxRetries(t *testing.T) {
	srv := testutil.NewServer(t, testutil.Respond(http.StatusInternalServerError, `{"errMessage":"boom"}`))

	c := fastClient()
	c.MaxRetries = 3

	resp, err := c.Do(context.Background(), jsonReq(srv.URL, true), "k")
	if err != nil {
		t.Fatalf("Do() = %v, want the last response rather than an error", err)
	}
	if want := 4; srv.Count() != want { // 1 attempt + 3 retries
		t.Errorf("server saw %d requests, want %d", srv.Count(), want)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d, want 500", resp.StatusCode)
	}
}

func TestDoDoesNotRetryClientErrors(t *testing.T) {
	srv := testutil.NewServer(t, testutil.Respond(http.StatusUnauthorized, `{}`))

	if _, err := fastClient().Do(context.Background(), jsonReq(srv.URL, true), "k"); err != nil {
		t.Fatalf("Do() = %v", err)
	}
	if srv.Count() != 1 {
		t.Errorf("server saw %d requests, want 1: a 401 will not fix itself", srv.Count())
	}
}

func TestDoResendsTheBodyOnEveryAttempt(t *testing.T) {
	srv := testutil.NewServer(t, testutil.Sequence(
		testutil.Respond(http.StatusServiceUnavailable, `{}`),
		testutil.Respond(http.StatusServiceUnavailable, `{}`),
		testutil.Respond(http.StatusOK, `{}`),
	))

	if _, err := fastClient().Do(context.Background(), jsonReq(srv.URL, true), "k"); err != nil {
		t.Fatalf("Do() = %v", err)
	}
	reqs := srv.Requests()
	if len(reqs) != 3 {
		t.Fatalf("server saw %d requests, want 3", len(reqs))
	}
	for i, r := range reqs {
		if string(r.Body) != reqBody {
			t.Errorf("attempt %d body = %q, want the full payload resent", i+1, r.Body)
		}
	}
}

func TestDoRetryInterruptedByContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	srv := testutil.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		cancel() // cancel while the transport is about to back off
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	// real delays, so the cancellation is what ends the loop
	_, err := New().Do(ctx, jsonReq(srv.URL, true), "k")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want it to wrap context.Canceled", err)
	}
	if srv.Count() != 1 {
		t.Errorf("server saw %d requests, want 1 before the cancellation", srv.Count())
	}
}

// --- retry helpers ---

func TestRetryableStatus(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{http.StatusOK, false},
		{http.StatusCreated, false},
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
		{http.StatusForbidden, false},
		{http.StatusNotFound, false},
		{http.StatusUnprocessableEntity, false},
		{http.StatusRequestTimeout, true},
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusGatewayTimeout, true},
	}
	for _, tt := range tests {
		if got := retryableStatus(tt.status); got != tt.want {
			t.Errorf("retryableStatus(%d) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		value string
		want  time.Duration
	}{
		{"", 0},
		{"0", 0},
		{"3", 3 * time.Second},
		{"120", 120 * time.Second},
		{"-5", 0},
		{"not-a-number", 0},
		{"Wed, 21 Oct 2026 07:28:00 GMT", 0}, // the http-date form is not handled
	}
	for _, tt := range tests {
		h := http.Header{}
		if tt.value != "" {
			h.Set("Retry-After", tt.value)
		}
		if got := parseRetryAfter(h); got != tt.want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestBackoff(t *testing.T) {
	c := New()
	c.BaseDelay = 100 * time.Millisecond
	c.MaxDelay = time.Second

	t.Run("retry-after wins and is capped by MaxDelay", func(t *testing.T) {
		if got := c.backoff(0, 500*time.Millisecond); got != 500*time.Millisecond {
			t.Errorf("backoff = %v, want the Retry-After value", got)
		}
		if got := c.backoff(0, time.Hour); got != time.Second {
			t.Errorf("backoff = %v, want it capped at MaxDelay", got)
		}
	})

	t.Run("jitter stays inside the exponential ceiling", func(t *testing.T) {
		for n := range 6 {
			ceiling := min(c.BaseDelay<<n, c.MaxDelay)
			for range 50 {
				d := c.backoff(n, 0)
				if d < 0 || d >= ceiling {
					t.Fatalf("backoff(%d) = %v, want it inside [0, %v)", n, d, ceiling)
				}
			}
		}
	})

	t.Run("zero delays do not panic", func(t *testing.T) {
		zero := &HTTPClient{BaseDelay: -1, MaxDelay: -1}
		if got := zero.backoff(0, 0); got < 0 {
			t.Errorf("backoff = %v, want a non-negative duration", got)
		}
	})
}

func TestDefaultsApplyWhenUnset(t *testing.T) {
	c := New()
	if got := c.maxRetries(); got != DefaultMaxRetries {
		t.Errorf("maxRetries() = %d, want %d", got, DefaultMaxRetries)
	}
	if got := c.baseDelay(); got != DefaultBaseDelay {
		t.Errorf("baseDelay() = %v, want %v", got, DefaultBaseDelay)
	}
	if got := c.maxDelay(); got != DefaultMaxDelay {
		t.Errorf("maxDelay() = %v, want %v", got, DefaultMaxDelay)
	}

	c.MaxRetries, c.BaseDelay, c.MaxDelay = 5, time.Second, time.Minute
	if got := c.maxRetries(); got != 5 {
		t.Errorf("maxRetries() = %d, want the configured 5", got)
	}
	if got := c.baseDelay(); got != time.Second {
		t.Errorf("baseDelay() = %v, want the configured 1s", got)
	}
	if got := c.maxDelay(); got != time.Minute {
		t.Errorf("maxDelay() = %v, want the configured 1m", got)
	}
}
