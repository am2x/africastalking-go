package africastalking

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/allanmuiruri/africastalking-go/internal/testutil"
	"github.com/allanmuiruri/africastalking-go/internal/transport"
)

// validPremium is a premium request that passes validation, so each test can
// vary one thing at a time.
func validPremium() SendPremiumSMSRequest {
	return SendPremiumSMSRequest{
		From:    "MYSHORTCODE",
		Keyword: "QUIZ",
		Message: "Your score is 7/10",
		To:      []string{"+254 711 123 456"},
	}
}

// premiumPayload mirrors the wire shape so a test can assert what was sent
// rather than trusting the Go struct it came from.
type premiumPayload struct {
	Username             string   `json:"username"`
	To                   []string `json:"to"`
	Message              string   `json:"message"`
	From                 string   `json:"from"`
	Keyword              string   `json:"keyword"`
	Enqueue              int      `json:"enqueue"`
	LinkID               string   `json:"linkId"`
	RetryDurationInHours int      `json:"retryDurationInHours"`
	RequestID            string   `json:"requestId"`
}

func TestSMSSendPremiumRequestShape(t *testing.T) {
	at, srv := newTestClient(t, testutil.Fixture(t, http.StatusOK, "sms/premium_success.json"))

	req := validPremium()
	req.LinkID = testLinkID
	req.RetryDurationInHours = 2
	req.RequestID = "client-req-1"

	if _, err := at.SMS.SendPremium(context.Background(), req); err != nil {
		t.Fatalf("SendPremium() = %v", err)
	}

	got := srv.LastRequest(t)
	if got.Method != http.MethodPost {
		t.Errorf("Method = %q, want POST", got.Method)
	}
	if got.Path != smsPremiumPath {
		t.Errorf("Path = %q, want %q", got.Path, smsPremiumPath)
	}
	if got.ContentType != transport.ContentTypeJSON {
		t.Errorf("Content-Type = %q, want application/json", got.ContentType)
	}
	if got.Accept != transport.ContentTypeJSON {
		t.Errorf("Accept = %q, want application/json", got.Accept)
	}
	if got.APIKey != testAPIKey {
		t.Errorf("apiKey = %q, want the client's key", got.APIKey)
	}
	if got.UserAgent == "" {
		t.Error("User-Agent is empty: every request should identify the SDK")
	}

	var sent premiumPayload
	got.DecodeJSON(t, &sent)

	if sent.Username != testUser {
		t.Errorf("username = %q, want the client's username", sent.Username)
	}
	if sent.From != "MYSHORTCODE" {
		t.Errorf("from = %q, want MYSHORTCODE", sent.From)
	}
	// Keyword and From are different fields with different meanings: the
	// keyword selects the premium subscription, the short code is the sender.
	if sent.Keyword != "QUIZ" {
		t.Errorf("keyword = %q, want QUIZ", sent.Keyword)
	}
	if sent.Message != "Your score is 7/10" {
		t.Errorf("message = %q", sent.Message)
	}
	if sent.LinkID != testLinkID {
		t.Errorf("linkId = %q, want it forwarded from the request", sent.LinkID)
	}
	if sent.RetryDurationInHours != 2 {
		t.Errorf("retryDurationInHours = %d, want 2", sent.RetryDurationInHours)
	}
	if sent.RequestID != "client-req-1" {
		t.Errorf("requestId = %q, want it forwarded from the request", sent.RequestID)
	}
	// The phone number was normalized before it left.
	if len(sent.To) != 1 || sent.To[0] != "+254711123456" {
		t.Errorf("to = %v, want the normalized number", sent.To)
	}
}

func TestSMSSendPremiumDecodesSuccess(t *testing.T) {
	at, _ := newTestClient(t, testutil.Fixture(t, http.StatusOK, "sms/premium_success.json"))

	resp, err := at.SMS.SendPremium(context.Background(), validPremium())
	if err != nil {
		t.Fatalf("SendPremium() = %v", err)
	}
	if len(resp.Accepted()) != 1 {
		t.Fatalf("len(Accepted()) = %d, want 1", len(resp.Accepted()))
	}
	if resp.Accepted()[0].StatusCode != SMSStatusSent {
		t.Errorf("StatusCode = %v, want Sent", resp.Accepted()[0].StatusCode)
	}
	if resp.Err() != nil {
		t.Errorf("Err() = %v, want nil", resp.Err())
	}
}

func TestSMSSendPremiumMapsNon2xxToAPIError(t *testing.T) {
	at, _ := newTestClient(t, testutil.Fixture(t, http.StatusUnauthorized, "sms/send_invalid_key.json"))

	_, err := at.SMS.SendPremium(context.Background(), validPremium())
	if err == nil {
		t.Fatal("SendPremium() = nil error, want an APIError")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("errors.Is(err, ErrUnauthorized) = false; err = %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("errors.As could not extract *APIError from %v", err)
	}
	if apiErr.Endpoint != smsPremiumPath {
		t.Errorf("Endpoint = %q, want %q", apiErr.Endpoint, smsPremiumPath)
	}
}

func TestSMSSendPremiumValidation(t *testing.T) {
	tests := []struct {
		testName  string
		mutate    func(*SendPremiumSMSRequest)
		wantField string
	}{
		{
			testName:  "missing message",
			mutate:    func(r *SendPremiumSMSRequest) { r.Message = "" },
			wantField: "Message",
		},
		{
			testName:  "missing from",
			mutate:    func(r *SendPremiumSMSRequest) { r.From = "" },
			wantField: "From",
		},
		{
			testName:  "no recipients",
			mutate:    func(r *SendPremiumSMSRequest) { r.To = nil },
			wantField: "To",
		},
		{
			testName:  "recipient without a country code",
			mutate:    func(r *SendPremiumSMSRequest) { r.To = []string{"0711123456"} },
			wantField: "To",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			at, srv := newTestClient(t, testutil.Respond(http.StatusOK, `{}`))

			req := validPremium()
			tt.mutate(&req)

			_, err := at.SMS.SendPremium(context.Background(), req)

			var vErr *ValidationErr
			if !errors.As(err, &vErr) {
				t.Fatalf("err = %v, want *ValidationErr", err)
			}
			if vErr.Field != tt.wantField {
				t.Errorf("Field = %q, want %q", vErr.Field, tt.wantField)
			}
			if srv.Count() != 0 {
				t.Errorf("server received %d requests, want 0: validation must short-circuit", srv.Count())
			}
		})
	}
}

func TestSMSSendPremiumIsNotRetried(t *testing.T) {
	// A premium send bills the subscriber and has no idempotency key, so a
	// transient failure must not be resent.
	at, srv := newTestClient(t, testutil.Sequence(
		testutil.Respond(http.StatusServiceUnavailable, `{}`),
		testutil.Fixture(t, http.StatusOK, "sms/premium_success.json"),
	))

	if _, err := at.SMS.SendPremium(context.Background(), validPremium()); err == nil {
		t.Fatal("SendPremium() = nil error, want the 503 mapped to an APIError")
	}
	if srv.Count() != 1 {
		t.Errorf("server saw %d requests, want exactly 1: a premium send must never be retried", srv.Count())
	}
}
