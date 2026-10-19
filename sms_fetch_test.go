package africastalking

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/allanmuiruri/africastalking-go/internal/testutil"
	"github.com/allanmuiruri/africastalking-go/internal/transport"
)

func TestSMSFetchRequestShape(t *testing.T) {
	at, srv := newTestClient(t, testutil.Fixture(t, http.StatusOK, "sms/fetch_success.json"))

	if _, err := at.SMS.Fetch(context.Background(), "15070"); err != nil {
		t.Fatalf("Fetch() = %v", err)
	}

	got := srv.LastRequest(t)
	if got.Method != http.MethodGet {
		t.Errorf("Method = %q, want GET", got.Method)
	}
	if got.Path != smsFetchPath {
		t.Errorf("Path = %q, want %q", got.Path, smsFetchPath)
	}
	if got.Accept != transport.ContentTypeJSON {
		t.Errorf("Accept = %q, want application/json", got.Accept)
	}
	if got.APIKey != testAPIKey {
		t.Errorf("apiKey = %q, want the client's key", got.APIKey)
	}

	// The query string is the whole payload of a fetch: if it is dropped, the
	// request still looks well formed but asks for nothing.
	if username := got.Query.Get("username"); username != testUser {
		t.Errorf("query username = %q, want the client's username", username)
	}
	if cursor := got.Query.Get("lastReceivedId"); cursor != "15070" {
		t.Errorf("query lastReceivedId = %q, want 15070", cursor)
	}

	// A GET has no body, so it should not claim a content type.
	if got.ContentType != "" {
		t.Errorf("Content-Type = %q, want it unset on a bodyless GET", got.ContentType)
	}
	if len(got.Body) != 0 {
		t.Errorf("Body = %q, want empty on a GET", got.Body)
	}
}

func TestSMSFetchDecodesMessages(t *testing.T) {
	at, _ := newTestClient(t, testutil.Fixture(t, http.StatusOK, "sms/fetch_success.json"))

	resp, err := at.SMS.Fetch(context.Background(), "0")
	if err != nil {
		t.Fatalf("Fetch() = %v", err)
	}

	msgs := resp.SMSMessageData.Messages
	if len(msgs) != 2 {
		t.Fatalf("len(Messages) = %d, want 2", len(msgs))
	}

	// Direction is reversed relative to a send: From is the subscriber, To is
	// our own short code.
	if msgs[0].From != "+254700000000" {
		t.Errorf("From = %q, want the subscriber", msgs[0].From)
	}
	if msgs[0].To != "20880" {
		t.Errorf("To = %q, want our short code", msgs[0].To)
	}
	if msgs[0].Text != "Hello" {
		t.Errorf("Text = %q", msgs[0].Text)
	}
	if msgs[0].LinkID != testLinkID {
		t.Errorf("LinkID = %q, want the premium link id", msgs[0].LinkID)
	}
	// ID is the cursor value for the next call.
	if msgs[1].ID != 15072 {
		t.Errorf("ID = %d, want 15072", msgs[1].ID)
	}
	if msgs[1].LinkID != "" {
		t.Errorf("LinkID = %q, want empty for a non-premium message", msgs[1].LinkID)
	}
}

func TestSMSFetchEmptyInboxIsNotAnError(t *testing.T) {
	at, _ := newTestClient(t, testutil.Fixture(t, http.StatusOK, "sms/fetch_empty.json"))

	resp, err := at.SMS.Fetch(context.Background(), "15072")
	if err != nil {
		t.Fatalf("Fetch() = %v, want nil: an empty inbox is a successful call", err)
	}
	if len(resp.SMSMessageData.Messages) != 0 {
		t.Errorf("len(Messages) = %d, want 0", len(resp.SMSMessageData.Messages))
	}
}

func TestSMSFetchAdvancesTheCursor(t *testing.T) {
	// Walking the inbox means feeding the last ID back in. Two pages then an
	// empty one is the shape every caller's loop takes.
	at, srv := newTestClient(t, testutil.Sequence(
		testutil.Fixture(t, http.StatusOK, "sms/fetch_success.json"),
		testutil.Fixture(t, http.StatusOK, "sms/fetch_empty.json"),
	))

	cursor := "0"
	pages := 0
	for {
		page, err := at.SMS.Fetch(context.Background(), cursor)
		if err != nil {
			t.Fatalf("Fetch() = %v", err)
		}
		if len(page.SMSMessageData.Messages) == 0 {
			break
		}
		pages++
		for _, m := range page.SMSMessageData.Messages {
			cursor = fmt.Sprint(m.ID)
		}
	}

	if pages != 1 {
		t.Errorf("read %d non-empty pages, want 1", pages)
	}
	if srv.Count() != 2 {
		t.Errorf("server saw %d requests, want 2", srv.Count())
	}
	// The second request must carry the cursor from the first page, not "0".
	if got := srv.Requests()[1].Query.Get("lastReceivedId"); got != "15072" {
		t.Errorf("second request lastReceivedId = %q, want 15072", got)
	}
}

func TestSMSFetchMapsNon2xxToAPIError(t *testing.T) {
	at, _ := newTestClient(t, testutil.Fixture(t, http.StatusUnauthorized, "sms/send_invalid_key.json"))

	_, err := at.SMS.Fetch(context.Background(), "0")
	if err == nil {
		t.Fatal("Fetch() = nil error, want an APIError")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("errors.Is(err, ErrUnauthorized) = false; err = %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("errors.As could not extract *APIError from %v", err)
	}
	if apiErr.Endpoint != smsFetchPath {
		t.Errorf("Endpoint = %q, want %q", apiErr.Endpoint, smsFetchPath)
	}
}

func TestSMSFetchIsRetried(t *testing.T) {
	// A fetch is a read with no side effect, so unlike a send it is safe to
	// retry a transient failure.
	at, srv := newTestClient(t, testutil.Sequence(
		testutil.Respond(http.StatusServiceUnavailable, `{}`),
		testutil.Fixture(t, http.StatusOK, "sms/fetch_success.json"),
	))

	resp, err := at.SMS.Fetch(context.Background(), "0")
	if err != nil {
		t.Fatalf("Fetch() = %v, want the retry to succeed", err)
	}
	if len(resp.SMSMessageData.Messages) != 2 {
		t.Errorf("len(Messages) = %d, want 2 after the retry", len(resp.SMSMessageData.Messages))
	}
	if srv.Count() != 2 {
		t.Errorf("server saw %d requests, want 2 (one failure then a retry)", srv.Count())
	}
}
