// Package transport performs the HTTP calls this SDK makes to Africa's
// Talking. It owns request construction, headers, authentication, the retry
// policy and response reading, so that a service method reduces to building a
// typed request and decoding a typed response.
//
// It is internal: everything here is implementation detail and may change
// without a major version bump.
package transport

// CircuitBreaker will stop issuing requests to an endpoint that is already
// failing, so callers fail fast instead of waiting out a full timeout each
// time.
//
// It is not implemented yet: the type is a placeholder so the shape of
// HTTPClient does not have to change when it lands. Nothing constructs one,
// and no request consults it.
type CircuitBreaker struct{}
