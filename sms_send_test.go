package africastalking

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/am2x/africastalking-go/internal/testutil"
)

// validSend is a request that passes validation, so a test can focus on one
// thing at a time.
func validSend() SendSMSRequest {
	return SendSMSRequest{
		SenderID:     "MYSHORTCODE",
		Message:      "Hello from Go",
		PhoneNumbers: []string{"+254 711 123 456"},
	}
}

func TestSMSSendRequestShape(t *testing.T) {
	at, srv := newTestClient(t, testutil.Fixture(t, http.StatusOK, "sms/send_success.json"))

	if _, err := at.SMS.Send(context.Background(), validSend()); err != nil {
		t.Fatalf("Send() = %v", err)
	}

	got := srv.LastRequest(t)
	if got.Method != http.MethodPost {
		t.Errorf("Method = %q, want POST", got.Method)
	}
	if got.Path != smsPath {
		t.Errorf("Path = %q, want %q", got.Path, smsPath)
	}
	if got.ContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got.ContentType)
	}
	if got.Accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", got.Accept)
	}
	if got.APIKey != testAPIKey {
		t.Errorf("apiKey = %q, want the client's key", got.APIKey)
	}
	if got.UserAgent == "" {
		t.Error("User-Agent is empty")
	}

	var sent struct {
		Username     string   `json:"username"`
		Message      string   `json:"message"`
		SenderID     string   `json:"senderId"`
		PhoneNumbers []string `json:"phoneNumbers"`
		Enqueue      int      `json:"enqueue"`
	}
	got.DecodeJSON(t, &sent)

	// the SDK fills username from the client; users never pass it
	if sent.Username != testUser {
		t.Errorf("username = %q, want the client's username", sent.Username)
	}
	if sent.Message != "Hello from Go" {
		t.Errorf("message = %q", sent.Message)
	}
	if sent.SenderID != "MYSHORTCODE" {
		t.Errorf("senderId = %q", sent.SenderID)
	}
	if len(sent.PhoneNumbers) != 1 || sent.PhoneNumbers[0] != "+254711123456" {
		t.Errorf("phoneNumbers = %v", sent.PhoneNumbers)
	}
	if sent.Enqueue != 0 {
		t.Errorf("enqueue = %d, want 0 when the caller left it false", sent.Enqueue)
	}
}

func TestSMSSendEnqueueCrossesTheWireAsAnInt(t *testing.T) {
	at, srv := newTestClient(t, testutil.Fixture(t, http.StatusOK, "sms/send_success.json"))

	req := validSend()
	req.Enqueue = true
	if _, err := at.SMS.Send(context.Background(), req); err != nil {
		t.Fatalf("Send() = %v", err)
	}

	var sent struct {
		Enqueue int `json:"enqueue"`
	}
	srv.LastRequest(t).DecodeJSON(t, &sent)
	if sent.Enqueue != 1 {
		t.Errorf("enqueue = %d, want 1 for a true Enqueue", sent.Enqueue)
	}
}

func TestSMSSendDecodesSuccess(t *testing.T) {
	at, _ := newTestClient(t, testutil.Fixture(t, http.StatusOK, "sms/send_success.json"))

	resp, err := at.SMS.Send(context.Background(), validSend())
	if err != nil {
		t.Fatalf("Send() = %v", err)
	}
	if len(resp.Accepted()) != 1 {
		t.Fatalf("len(Accepted()) = %d, want 1", len(resp.Accepted()))
	}
	if len(resp.Failed()) != 0 {
		t.Errorf("len(Failed()) = %d, want 0", len(resp.Failed()))
	}
	if resp.Err() != nil {
		t.Errorf("Err() = %v, want nil", resp.Err())
	}

	rec := resp.Accepted()[0]
	if rec.StatusCode != SMSStatusSent {
		t.Errorf("StatusCode = %v, want Sent", rec.StatusCode)
	}
	if rec.Number != "+254700000000" {
		t.Errorf("Number = %q", rec.Number)
	}
	if rec.MessageID == "" {
		t.Error("MessageID is empty, want the id from the fixture")
	}
}

func TestSMSSendPartialFailureIsNotAnError(t *testing.T) {
	at, _ := newTestClient(t, testutil.Fixture(t, http.StatusOK, "sms/send_partial_failure.json"))

	resp, err := at.SMS.Send(context.Background(), validSend())
	if err != nil {
		t.Fatalf("Send() = %v, want nil: a partial failure is a successful call", err)
	}
	if len(resp.Accepted()) != 1 {
		t.Errorf("len(Accepted()) = %d, want 1", len(resp.Accepted()))
	}
	if len(resp.Failed()) != 2 {
		t.Errorf("len(Failed()) = %d, want 2", len(resp.Failed()))
	}
	if resp.Err() != nil {
		t.Errorf("Err() = %v, want nil while one recipient was accepted", resp.Err())
	}

	failed := resp.Failed()
	if failed[0].StatusCode != SMSStatusInvalidPhoneNumber {
		t.Errorf("failed[0] = %v, want InvalidPhoneNumber", failed[0].StatusCode)
	}
	if failed[1].StatusCode != SMSStatusInsufficientBalance {
		t.Errorf("failed[1] = %v, want InsufficientBalance", failed[1].StatusCode)
	}
}

func TestSMSSendAllRejectedSurfacesViaErr(t *testing.T) {
	at, _ := newTestClient(t, testutil.Fixture(t, http.StatusOK, "sms/send_all_rejected.json"))

	resp, err := at.SMS.Send(context.Background(), validSend())
	if err != nil {
		t.Fatalf("Send() = %v, want nil: the API accepted the request", err)
	}
	if len(resp.Accepted()) != 0 {
		t.Errorf("len(Accepted()) = %d, want 0", len(resp.Accepted()))
	}

	rejErr := resp.Err()
	if rejErr == nil {
		t.Fatal("Err() = nil, want an error when every recipient is rejected")
	}
	var rejected *SMSRejectedError
	if !errors.As(rejErr, &rejected) {
		t.Fatalf("errors.As could not extract *SMSRejectedError from %v", rejErr)
	}
	if len(rejected.Recipients) != 2 {
		t.Errorf("len(Recipients) = %d, want 2", len(rejected.Recipients))
	}
}

func TestSMSSendMapsNon2xxToAPIError(t *testing.T) {
	at, _ := newTestClient(t, testutil.Fixture(t, http.StatusUnauthorized, "sms/send_invalid_key.json"))

	_, err := at.SMS.Send(context.Background(), validSend())
	if err == nil {
		t.Fatal("Send() = nil error, want an APIError")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("errors.Is(err, ErrUnauthorized) = false; err = %v", err)
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("errors.As could not extract *APIError from %v", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if apiErr.Endpoint != smsPath {
		t.Errorf("Endpoint = %q, want %q", apiErr.Endpoint, smsPath)
	}
	// if this fails, extractMessage's probe field names do not match the fixture
	if apiErr.Message != "Invalid credentials" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "Invalid credentials")
	}
}

func TestSMSSendErrorsOnMalformedBody(t *testing.T) {
	at, _ := newTestClient(t, testutil.Respond(http.StatusOK, `<html>not json</html>`))

	_, err := at.SMS.Send(context.Background(), validSend())
	if err == nil {
		t.Fatal("Send() = nil error, want a decode failure")
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Error("a 200 with a bad body should not become an APIError")
	}
}

func TestSMSSendValidationNeverReachesTheNetwork(t *testing.T) {
	tests := []struct {
		testName  string
		mutate    func(*SendSMSRequest)
		wantField string
	}{
		{
			testName:  "missing message",
			mutate:    func(r *SendSMSRequest) { r.Message = "" },
			wantField: "Message",
		},
		{
			testName:  "missing sender id",
			mutate:    func(r *SendSMSRequest) { r.SenderID = "" },
			wantField: "SenderID",
		},
		{
			testName:  "no recipients",
			mutate:    func(r *SendSMSRequest) { r.PhoneNumbers = nil },
			wantField: "PhoneNumbers",
		},
		{
			testName:  "missing + phone number",
			mutate:    func(r *SendSMSRequest) { r.PhoneNumbers = []string{"254711123456"} },
			wantField: "PhoneNumbers",
		},
		{
			testName:  "number with non-digit characters",
			mutate:    func(r *SendSMSRequest) { r.PhoneNumbers = []string{"254711ufg456"} },
			wantField: "PhoneNumbers",
		},
		{
			testName:  "too many digits number",
			mutate:    func(r *SendSMSRequest) { r.PhoneNumbers = []string{"254711111123456789"} },
			wantField: "PhoneNumbers",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			at, srv := newTestClient(t, testutil.Respond(http.StatusOK, `{}`))

			req := validSend()
			tt.mutate(&req)

			_, err := at.SMS.Send(context.Background(), req)

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
			// a validation failure is not an API failure
			var apiErr *APIError
			if errors.As(err, &apiErr) {
				t.Error("a ValidationErr should not satisfy errors.As for *APIError")
			}
		})
	}
}

func TestSMSSendHonoursContextCancellation(t *testing.T) {
	at, srv := newTestClient(t, testutil.Respond(http.StatusOK, `{}`))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := at.SMS.Send(ctx, validSend())
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want it to wrap context.Canceled", err)
	}
	if srv.Count() != 0 {
		t.Errorf("server received %d requests, want 0 for a cancelled context", srv.Count())
	}
}

func TestSMSSendIsNotRetried(t *testing.T) {
	// sending an SMS costs money and has no idempotency key, so a 503 must not
	// be resent even though the transport is capable of it
	at, srv := newTestClient(t, testutil.Sequence(
		testutil.Respond(http.StatusServiceUnavailable, `{}`),
		testutil.Fixture(t, http.StatusOK, "sms/send_success.json"),
	))

	if _, err := at.SMS.Send(context.Background(), validSend()); err == nil {
		t.Fatal("Send() = nil error, want the 503 mapped to an APIError")
	}
	if srv.Count() != 1 {
		t.Errorf("server saw %d requests, want exactly 1: an SMS send must never be retried", srv.Count())
	}
}
