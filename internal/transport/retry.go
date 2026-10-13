package transport

import (
	"context"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// Retry defaults, used when HTTPClient leaves them at zero.
const (
	DefaultMaxRetries = 2
	DefaultBaseDelay  = 200 * time.Millisecond
	DefaultMaxDelay   = 5 * time.Second
)

func (c *HTTPClient) maxRetries() int {
	if c.MaxRetries > 0 {
		return c.MaxRetries
	}
	return DefaultMaxRetries
}

func (c *HTTPClient) baseDelay() time.Duration {
	if c.BaseDelay > 0 {
		return c.BaseDelay
	}
	return DefaultBaseDelay
}

func (c *HTTPClient) maxDelay() time.Duration {
	if c.MaxDelay > 0 {
		return c.MaxDelay
	}
	return DefaultMaxDelay
}

// retryableStatus reports whether a status is worthy another attempt.
// 4xx are excluded apart from 408 and 429; the request itself is the problem,
// so resending it unchanged will still fail.
func retryableStatus(status int) bool {
	switch {
	case status == http.StatusRequestTimeout, status == http.StatusTooManyRequests:
		return true
	case status >= 500:
		return true
	default:
		return false
	}
}

// backoff returns how long to wait before another attempt n(0-based).
// It uses full jitter - a random point in [0,ceiling] - so that retries from many clients
// do not resynchronize into a thundering herd.
func (c *HTTPClient) backoff(n int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return min(retryAfter, c.maxDelay())
	}
	ceiling := c.baseDelay() << n
	if ceiling > c.maxDelay() {
		ceiling = c.maxDelay()
	}
	if ceiling <= 0 {
		return 0
	}

	// Jitter only needs to be unpredictable enough to spread retries out; it is
	// not a secret, so a cryptographic source would be pointless overhead.
	return rand.N(ceiling) //nolint:gosec // G404: jitter needs no crypto randomness
}

// parseRetryAfter reads the delay-seconds from the Retry-After header.
func parseRetryAfter(h http.Header) time.Duration {
	v := h.Get("Retry-After")
	if v == "" {
		return 0
	}
	secs, err := strconv.Atoi(v)
	if err != nil || secs < 0 {
		return 0 // the HTTP-date form is not handled; default to backoff.
	}
	return time.Duration(secs) * time.Second
}

// wait sleeps for d unless ctx finishes first
func wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
