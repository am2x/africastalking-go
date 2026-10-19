package africastalking

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Option configures a client during [New]. Options are applied in order and
// may return an error to reject their input.
type Option func(*AfricasTalking) error

// WithCredentials sets the Africa's Talking username and API key. It is
// required: [New] fails without it.
func WithCredentials(username, apiKey string) Option {
	return func(at *AfricasTalking) error {
		if username == "" {
			return errors.New("africastalking: missing username")
		}
		if apiKey == "" {
			return errors.New("africastalking: missing api key")
		}
		at.userName, at.apiKey = username, apiKey
		return nil
	}
}

// WithSandbox points every product at the Africa's Talking sandbox hosts.
// Without it, requests go to production.
func WithSandbox() Option {
	return func(at *AfricasTalking) error {
		at.hosts = sandboxHosts
		return nil
	}
}

// WithBaseURL overrides the base URL used for every product. It exists for
// tests and for routing through a proxy; prefer [WithSandbox] for normal use.
func WithBaseURL(rawURL string) Option {
	return func(at *AfricasTalking) error {
		u, err := url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("africastalking: invalid base url %q: %w", rawURL, err)
		}
		if u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("africastalking: base url %q needs a scheme and host", rawURL)
		}
		base := strings.TrimSuffix(rawURL, "/")
		at.hosts = hosts{api: base, voice: base, payments: base, content: base}
		return nil
	}
} // escape hatch/testing
// func WithRetry()     {}
// func WithUserAgent() Option{}
// func WithLogger() Option {}
