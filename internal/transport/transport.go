package transport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// http client used to interact with africa's talking(internal use only!)
type HTTPClient struct {
	Client     *http.Client
	Breaker    *CircuitBreaker
	MaxRetries int
	// BaseDelay  time.Duration
	// MaxDelay   time.Duration
	// Wait func(time.Duration)
}

const (
	ContentTypeJSON        = "application/json"
	ContentTypeFormEncoded = "application/x-www-form-urlencoded"
	UserAgent              = "africastalking-go-sdk"
)

type Request struct {
	Method      string
	URL         string
	ContentType string
	UserAgent   string
	Body        []byte
	// Retryable   bool //TODO: set up once retry.go is live: false for anything involving money transfer
}

type Response struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

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

func (c *HTTPClient) Do(ctx context.Context, r Request, apiKey string) (*Response, error) {
	request, err := http.NewRequestWithContext(ctx, r.Method, r.URL, bytes.NewReader(r.Body))
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
	defer response.Body.Close()
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
