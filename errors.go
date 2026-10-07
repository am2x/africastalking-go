package africastalking

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/allanmuiruri/africastalking-go/internal/transport"
)

// APIError is returned when Africa's Talking rejects a request.
type APIError struct {
	StatusCode int    // HTTP status returned by the API
	Endpoint   string // path that was called, e.g. "/messaging/bulk"
	Message    string // message extracted from the response, when there is one
	Body       string // raw response body, truncated
}

// Sentinel errors for matching broad API failure conditions with errors.Is.
// Use errors.As with [APIError] when you need the status code or body.
var (
	ErrUnauthorized = errors.New("africastalking: unauthorized")
	ErrRateLimited  = errors.New("africastalking: rate limited")
	ErrBadRequest   = errors.New("africastalking: bad request")
	ErrServer       = errors.New("africastalking: server error")
)

const maxErrBody = 512

func newAPIError(endpoint string, resp *transport.Response) *APIError {
	body := strings.TrimSpace(string(resp.Body))
	if len(body) > maxErrBody {
		body = body[:maxErrBody] + "..."
	}
	return &APIError{
		StatusCode: resp.StatusCode,
		Endpoint:   endpoint,
		Message:    extractMessage(resp.Body),
		Body:       body,
	}
}

// extractMessage pulls a human-readable message out of an error response.
// Africa's Talking is not consistent here: some endpoints return JSON, others
// plain text, and occasionally HTML — so this is best-effort by design.
func extractMessage(body []byte) string {
	var probe struct {
		Message        string `json:"message"`
		ErrMessage     string `json:"errMessage"`
		Description    string `json:"description"`
		Status         string `json:"status"`
		SMSMessageData struct {
			Message string `json:"Message"`
		} `json:"SMSMessageData"`
	}
	if json.Unmarshal(body, &probe) == nil {
		for _, s := range []string{
			probe.ErrMessage, probe.Message, probe.Description, probe.SMSMessageData.Message, probe.Status,
		} {
			if s != "" {
				return s
			}
		}
	}
	return "" // not JSON — Body still carries the raw text
}

// Error returns an well formatted error string.
func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("africastalking: %s: %d: %s", e.Endpoint, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("africastalking: %s: unexpected status: %d", e.Endpoint, e.StatusCode)
}

// Is lets callers match broad conditions with errors.Is while still being able
// to pull the full detail out with errors.As.
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrUnauthorized:
		return e.StatusCode == http.StatusUnauthorized || e.StatusCode == http.StatusForbidden
	case ErrRateLimited:
		return e.StatusCode == http.StatusTooManyRequests
	case ErrBadRequest:
		return e.StatusCode == http.StatusBadRequest
	case ErrServer:
		return e.StatusCode == http.StatusInternalServerError
	}
	return false
}

// ValidationErr reports input rejected by the SDK before any request was sent.
type ValidationErr struct {
	Field   string
	Message string
}

func (e *ValidationErr) Error() string {
	return fmt.Sprintf("africastalking: invalid %s: %s", e.Field, e.Message)
}
