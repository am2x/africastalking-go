package africastalking

import "errors"

type Option func(*AfricasTalking) error

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

// set base url based on user's sandbox option
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
