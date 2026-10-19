package transport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// HTTPClient performs HTTP requests against Africa's Talking. It is internal
// to this module: the retry policy, headers and encoding it applies are
// implementation detail and may change without a major version bump.
type HTTPClient struct {
	Client *http.Client
	// Breaker    *CircuitBreaker //TODO: implement circuit breaker.
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	// Wait func(time.Duration)
}

// Content types accepted by Africa's Talking, and the User-Agent this SDK
// identifies itself with. The v1 endpoints on the api host are form encoded;
// Payments, Mobile Data and parts of Voice take JSON.
const (
	ContentTypeJSON        = "application/json"
	ContentTypeFormEncoded = "application/x-www-form-urlencoded"
	UserAgent              = "africastalking-go-sdk"
)

// Request is a single call to an Africa's Talking endpoint, already encoded.
// Services build one of these; only the transport knows how it is sent.
type Request struct {
	Method      string
	URL         string
	ContentType string
	UserAgent   string
	Query       url.Values
	Body        []byte
	// Retryable allows the transport to resend this request on a transient failure.
	// Leave it false for anything that creates, sends or moves money:
	// a resend after a timeout can duplicate the effect.
	Retryable bool
}

// Response is the outcome of a Request. Any HTTP status is reported here
// rather than as an error: mapping a status to an APIError is the caller's
// job, because several Africa's Talking endpoints describe a failure in the
// body of an otherwise successful response.
type Response struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

// New returns an HTTPClient with sane connection pooling and a one minute
// timeout. Retry tuning is left at zero, so the Default values in retry.go
// apply until a caller overrides them.
func New() *HTTPClient {
	return &HTTPClient{
		Client: &http.Client{
			Timeout: 1 * time.Minute,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
			},
		},
	}
}

// Do performs retryable attempts.
func (c *HTTPClient) Do(ctx context.Context, r Request, apiKey string) (*Response, error) {
	attempts := 1
	if r.Retryable {
		attempts += c.maxRetries()
	}

	var resp *Response
	for n := range attempts {
		if n > 0 {
			if err := wait(ctx, c.backoff(n-1, parseRetryAfter(resp.Header))); err != nil {
				return resp, fmt.Errorf("transport: retry interrupted: %w", err)
			}
		}

		var err error
		resp, err = c.do(ctx, r, apiKey)
		if err != nil {
			return nil, err
		}
		if !retryableStatus(resp.StatusCode) {
			return resp, nil
		}
	}
	// every attempt was retryable; hand back the last response and let the
	// caller map it to an error.
	return resp, nil
}

// do performs a single attempt.
func (c *HTTPClient) do(ctx context.Context, r Request, apiKey string) (*Response, error) {
	var fullURL string
	if len(r.Query) > 0 {
		fullURL = r.URL + "?" + r.Query.Encode()
	} else {
		fullURL = r.URL
	}

	request, err := http.NewRequestWithContext(ctx, r.Method, fullURL, bytes.NewReader(r.Body))
	if err != nil {
		return &Response{}, fmt.Errorf("transport: build request: %w", err)
	}
	request.Header.Set("Accept", ContentTypeJSON)
	request.Header.Set("Content-Type", r.ContentType)
	request.Header.Set("User-Agent", r.UserAgent)
	request.Header.Set("apiKey", apiKey)

	response, err := c.Client.Do(request)
	if err != nil {
		return &Response{}, fmt.Errorf("transport: request failed: %w", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err.Error())
		}
	}()
	bodyText, err := io.ReadAll(response.Body)
	if err != nil {
		return &Response{}, fmt.Errorf("transport: parsing request: %w", err)
	}

	resp := Response{
		StatusCode: response.StatusCode,
		Body:       bodyText,
		Header:     response.Header,
	}

	return &resp, nil
}
