package africastalking

import "errors"

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

// func WithBaseURL() Option {} //escape hatch/testing
// func WithRetry()     {}
// func WithUserAgent() Option{}
// func WithLogger() Option {}
