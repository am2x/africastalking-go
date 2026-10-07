package africastalking

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/allanmuiruri/africastalking-go/internal/transport"
)

func Test_newAPIError(t *testing.T) {
	t.Run("populates every field from a json error body", func(t *testing.T) {
		got := newAPIError(smsPath, &transport.Response{
			StatusCode: 400,
			Body:       []byte(`{"errMessage":"Bad request"}`),
		})
		if got.StatusCode != 400 {
			t.Errorf("StatusCode = %d, want 400", got.StatusCode)
		}
		if got.Endpoint != smsPath {
			t.Errorf("Endpoint = %q, want %q", got.Endpoint, smsPath)
		}
		if got.Message != "Bad request" {
			t.Errorf("Message =%q, want %q", got.Message, "Bad request")
		}
		if got.Body != `{"errMessage":"Bad request"}` {
			t.Errorf("Body = %q, want the raw body", got.Body)
		}
	})
	t.Run("keeps the raw body when there is no message", func(t *testing.T) {
		got := newAPIError(smsPath, &transport.Response{
			StatusCode: 401,
			Body:       []byte("  Unauthorized  "),
		})
		if got.Message != "" {
			t.Errorf("Message = %q, want empty for a plain text body", got.Message)
		}
		if got.Body != "Unauthorized" {
			t.Errorf("Body = %q, want it trimmed", got.Body)
		}
	})
	t.Run("truncates an oversized body", func(t *testing.T) {
		got := newAPIError(smsPath, &transport.Response{
			StatusCode: 502,
			Body:       []byte(strings.Repeat("a", maxErrBody+3)),
		})
		if len(got.Body) != maxErrBody+3 {
			t.Errorf("len(Body) = %q, want %d", len(got.Body), maxErrBody+3)
		}
		if strings.HasSuffix(got.Body, "....") {
			t.Errorf("Body = %q, wanted it to end with ellipsis", got.Body[len(got.Body)-10:])
		}
	})
}

func Test_extractMessage(t *testing.T) {
	tests := []struct {
		testName string
		body     []byte
		want     string
	}{
		{
			testName: "json error message is preferred",
			body: []byte(`{ 
			    "message":     "Invalid request, missing senderId",
				"errMessage":  "Bad request",
				"description": "Payload is missing senderId value",
				"status":      "400"
			}`),
			want: "Bad request",
		}, {
			testName: "falls back to message when errMessage is absent",
			body:     []byte(`{"message":"Invalid senderId"}`),
			want:     "Invalid senderId",
		}, {
			testName: "nested sms message data",
			body:     []byte(`{"SMSMessageData":{"Message":"InvalidSenderId"}}`),
			want:     "InvalidSenderId",
		}, {
			testName: "plain text body yields no message",
			body:     []byte(`Unauthorized`),
			want:     "",
		}, {
			testName: "html body yields no message",
			body:     []byte(`<html><body>502 Bad Gateway</body></html>`),
			want:     "",
		}, {
			testName: "empty body yields no message",
			body:     nil,
			want:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			if got := extractMessage(tt.body); got != tt.want {
				t.Errorf("extractMessage() = %q, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		testName string
		err      *APIError
		resp     *transport.Response
		want     string
	}{
		{
			testName: "includes the message when there is none",
			err:      &APIError{StatusCode: 400, Endpoint: smsPath, Message: "Bad request"},
			want:     "africastalking: /messaging/bulk: 400: Bad request",
		}, {
			testName: "falls back to the status when there is no message",
			err:      &APIError{StatusCode: 401, Endpoint: smsPath},
			want:     "africastalking: /messaging/bulk: unexpected status: 401",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIErrorSentinels(t *testing.T) {
	tests := []struct {
		testName   string
		statusCode int
		want       error
	}{
		{"401 is unauthorized", 401, ErrUnauthorized},
		{"403 is unauthorized", 403, ErrUnauthorized},
		{"429 is rate limited", 429, ErrRateLimited},
		{"400 is a bad request", 400, ErrBadRequest},
		{"500 is a server error", 500, ErrServer},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			// wrapped exactly the way Send wraps it
			err := fmt.Errorf("africastalking: send sms: %w",
				newAPIError(smsPath, &transport.Response{StatusCode: tt.statusCode}))
			if !errors.Is(err, tt.want) {
				t.Errorf("errors.Is(%d, %v) = false, want true", tt.statusCode, tt.want)
			}
		})
	}
}

func TestAPIErrorUnwrapping(t *testing.T) {
	err := fmt.Errorf("africastalking: send sms: %w",
		newAPIError(smsPath, &transport.Response{
			StatusCode: 401,
			Body:       []byte(`{"errMessage":"Invalid credentials"}`),
		}))
	t.Run("errors.As reaches through the wrap", func(t *testing.T) {
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatal("errors.As could not extract *APIError")
		}
		if apiErr.StatusCode != 401 {
			t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
		}
		if apiErr.Message != "Invalid credentials" {
			t.Errorf("Message = %q, want %q", apiErr.Message, "Invalid credentials")
		}
	})
	t.Run("does not match unrelated sentinels", func(t *testing.T) {
		if errors.Is(err, ErrServer) {
			t.Error("a 401 should not match ErrServer")
		}
		if errors.Is(err, ErrRateLimited) {
			t.Error("a 401 should not match ErrRateLimited")
		}
	})
	t.Run("validation errors are a separate type", func(t *testing.T) {
		vErr := error(&ValidationErr{Field: "PhoneNumbers", Message: "at least one recipient is required"})
		var apiErr *APIError
		if errors.As(vErr, &apiErr) {
			t.Error("a ValidationErr shouldn't satisfy errors.As for *APIError")
		}
		var valErr *ValidationErr
		if !errors.As(vErr, &valErr) {
			t.Fatal("errors.As could not extract *ValidationErr")
		}
		if valErr.Field != "PhoneNumbers" {
			t.Errorf("Field = %q, want %q", valErr.Field, "PhoneNumbers")
		}
	})
}
