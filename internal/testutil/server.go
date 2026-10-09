// Package testutil provides helpers for testing the SDK against a local HTTP
// server instead of the live API. It is internal and test-only: it must never
// import the root package, so that the root package can import it.
package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// Request is a recording of one request the SDK sent.
type Request struct {
	Method      string
	Path        string
	Query       url.Values
	ContentType string
	Accept      string
	UserAgent   string
	APIKey      string
	Body        []byte
}

// DecodeJSON unmarshals the recorded body into v
func (r Request) DecodeJSON(t testing.TB, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("request body is not valid json: %v\nbody: %s", err, r.Body)
	}
}

// Server is an httptest.Server that records every request it receives.
type Server struct {
	*httptest.Server
	mu       sync.Mutex
	requests []Request
}

// New Server starts a server that answers every request with h and records what
// it was sent. It is closed automatically when the test finishes.
func NewServer(t testing.TB, h http.Handler) *Server {
	t.Helper()
	s := &Server{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("testutil: reading request body: %v", err)
		}
		s.mu.Lock()
		s.requests = append(s.requests, Request{
			Method:      r.Method,
			Path:        r.URL.Path,
			Query:       r.URL.Query(),
			ContentType: r.Header.Get("Content-Type"),
			Accept:      r.Header.Get("Accept"),
			UserAgent:   r.Header.Get("User-Agent"),
			APIKey:      r.Header.Get("apiKey"),
			Body:        body,
		})
		s.mu.Unlock()

		r.Body = io.NopCloser(bytes.NewReader(body)) //let the handler read it too
		h.ServeHTTP(w, r)
	}))
	t.Cleanup(s.Server.Close)
	return s
}

// Requests returns every request the server received in order.
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}

// Count returns how many requests the server received.
func (s *Server) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

// LastRequest returns the most recent request, failing the test if none arrived.
func (s *Server) LastRequest(t testing.TB) Request {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.requests) == 0 {
		t.Fatal("testutil: no request reached the server")
	}
	return s.requests[len(s.requests)-1]
}

// Respond replies with status and body for every request
func Respond(status int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	})
}

// Fixture replies with status and the contents of testdata/<name>, resolved
// relative to the package under test.
func Fixture(t testing.TB, status int, name string) http.Handler {
	t.Helper()
	body, err := readFixture(name)
	if err != nil {
		t.Fatalf("testutil: %v", err)
	}
	return Respond(status, string(body))
}

// readFixture holds the part of Fixture that can fail, so it can be tested
// without a testing.TB (which cannot be faked outside the testing package).
func readFixture(name string) ([]byte, error) {
	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		return nil, fmt.Errorf("reading fixture %s: %w", name, err)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, fmt.Errorf("fixture %s is empty", name)
	}
	return body, nil
}

// Sequence replies with each handler in turn, so a test can exercise a retry
// or a paged read. The final handler answers any extra requests.
func Sequence(handlers ...http.Handler) http.Handler {
	if len(handlers) == 0 {
		panic("testutil: Sequence needs at least one handler")
	}
	var (
		mu sync.Mutex
		i  int
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		h := handlers[min(i, len(handlers)-1)]
		i++
		mu.Unlock()
		h.ServeHTTP(w, r)
	})
}
