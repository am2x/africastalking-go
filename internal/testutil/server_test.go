package testutil

import (
	"net/http"
	"strings"
	"testing"

	"github.com/am2x/africastalking-go/internal/transport"
)

func get(t testing.TB, url string) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	t.Cleanup(func() {
		_ = resp.Body.Close()
	})
	return resp
}

func TestRespond(t *testing.T) {
	srv := NewServer(t, Respond(http.StatusTeapot, `{"brewing":false}`))

	resp := get(t, srv.URL)
	if resp.StatusCode != http.StatusTeapot {
		t.Errorf("StatusCode = %d, want 418", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != transport.ContentTypeJSON {
		t.Errorf("Content-Type = %q, want %s", got, transport.ContentTypeJSON)
	}
}

func TestServerRecordsRequests(t *testing.T) {
	srv := NewServer(t, Respond(http.StatusOK, `{}`))

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/messaging/bulk?from=sdk", strings.NewReader(`{"message":"hi"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", transport.ContentTypeJSON)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("apiKey", "secret-key")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	got := srv.LastRequest(t)
	if got.Method != http.MethodPost {
		t.Errorf("Method = %q, want POST", got.Method)
	}
	if got.Path != "/messaging/bulk" {
		t.Errorf("Path = %q, want /messaging/bulk (query excluded)", got.Path)
	}
	if got.Query.Get("from") != "sdk" {
		t.Errorf("Query = %v, want from=sdk", got.Query)
	}
	if got.ContentType != "application/json" {
		t.Errorf("ContentType = %q", got.ContentType)
	}
	if got.Accept != "application/json" {
		t.Errorf("Accept = %q", got.Accept)
	}
	if got.UserAgent != "test-agent" {
		t.Errorf("UserAgent = %q", got.UserAgent)
	}
	if got.APIKey != "secret-key" {
		t.Errorf("APIKey = %q", got.APIKey)
	}
	if string(got.Body) != `{"message":"hi"}` {
		t.Errorf("Body = %q", got.Body)
	}
}

func TestServerPassesBodyToTheHandler(t *testing.T) {
	// the recorder reads the body first, so it must put it back for the handler
	var seen string
	srv := NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 64)
		n, _ := r.Body.Read(buf)
		seen = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))

	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if seen != `{"a":1}` {
		t.Errorf("handler saw %q, want the body to be readable downstream", seen)
	}
}

func TestServerCountAndRequests(t *testing.T) {
	srv := NewServer(t, Respond(http.StatusOK, `{}`))
	if srv.Count() != 0 {
		t.Errorf("Count() = %d on a fresh server, want 0", srv.Count())
	}

	for range 3 {
		_ = get(t, srv.URL)
	}

	if srv.Count() != 3 {
		t.Errorf("Count() = %d, want 3", srv.Count())
	}
	reqs := srv.Requests()
	if len(reqs) != 3 {
		t.Fatalf("len(Requests()) = %d, want 3", len(reqs))
	}

	// Requests returns a copy: mutating it must not affect the server
	reqs[0].Path = "/mutated"
	if srv.Requests()[0].Path == "/mutated" {
		t.Error("Requests() handed out the internal slice")
	}
}

func TestRequestDecodeJSON(t *testing.T) {
	srv := NewServer(t, Respond(http.StatusOK, `{}`))
	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(`{"username":"testuser","count":2}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	var got struct {
		Username string `json:"username"`
		Count    int    `json:"count"`
	}
	// chained off LastRequest, which returns a value: this only compiles with
	// a value receiver on DecodeJSON
	srv.LastRequest(t).DecodeJSON(t, &got)

	if got.Username != "testuser" {
		t.Errorf("Username = %q, want testuser", got.Username)
	}
	if got.Count != 2 {
		t.Errorf("Count = %d, want 2", got.Count)
	}
}

func TestSequence(t *testing.T) {
	t.Run("replies with each handler in turn", func(t *testing.T) {
		srv := NewServer(t, Sequence(
			Respond(http.StatusServiceUnavailable, `{"n":1}`),
			Respond(http.StatusTooManyRequests, `{"n":2}`),
			Respond(http.StatusOK, `{"n":3}`),
		))

		want := []int{
			http.StatusServiceUnavailable,
			http.StatusTooManyRequests,
			http.StatusOK,
		}
		for i, wantStatus := range want {
			if got := get(t, srv.URL).StatusCode; got != wantStatus {
				t.Errorf("request %d status = %d, want %d", i+1, got, wantStatus)
			}
		}
	})

	t.Run("the last handler answers any extra requests", func(t *testing.T) {
		srv := NewServer(t, Sequence(
			Respond(http.StatusServiceUnavailable, `{}`),
			Respond(http.StatusOK, `{}`),
		))

		_ = get(t, srv.URL) // 503
		for i := range 3 {
			if got := get(t, srv.URL).StatusCode; got != http.StatusOK {
				t.Errorf("extra request %d status = %d, want the final handler's 200", i+1, got)
			}
		}
	})

	t.Run("a single handler answers everything", func(t *testing.T) {
		srv := NewServer(t, Sequence(Respond(http.StatusCreated, `{}`)))
		for range 2 {
			if got := get(t, srv.URL).StatusCode; got != http.StatusCreated {
				t.Errorf("status = %d, want 201", got)
			}
		}
	})

	t.Run("panics with no handlers", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("Sequence() with no handlers did not panic")
			}
		}()
		Sequence()
	})
}

func TestReadFixture(t *testing.T) {
	t.Run("reads a file from testdata", func(t *testing.T) {
		got, err := readFixture("sample.json")
		if err != nil {
			t.Fatalf("readFixture() = %v", err)
		}
		if !strings.Contains(string(got), `"hello"`) {
			t.Errorf("body = %q, want the fixture contents", got)
		}
	})

	t.Run("errors on a missing file", func(t *testing.T) {
		_, err := readFixture("does_not_exist.json")
		if err == nil {
			t.Fatal("readFixture() = nil error, want a read failure")
		}
		if !strings.Contains(err.Error(), "reading fixture") {
			t.Errorf("err = %q", err)
		}
	})

	t.Run("errors on an empty file", func(t *testing.T) {
		// an unfilled placeholder is the common case this guards against
		_, err := readFixture("empty.json")
		if err == nil {
			t.Fatal("readFixture() = nil error, want an empty-fixture failure")
		}
		if !strings.Contains(err.Error(), "is empty") {
			t.Errorf("err = %q, want it to name the empty fixture", err)
		}
	})
}

func TestFixtureServesTheFile(t *testing.T) {
	srv := NewServer(t, Fixture(t, http.StatusOK, "sample.json"))

	resp := get(t, srv.URL)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	buf := make([]byte, 128)
	n, _ := resp.Body.Read(buf)
	if !strings.Contains(string(buf[:n]), `"fixture"`) {
		t.Errorf("body = %q, want the fixture contents", buf[:n])
	}
}

func TestNewServerClosesOnCleanup(t *testing.T) {
	var srv *Server
	t.Run("inner", func(t *testing.T) {
		srv = NewServer(t, Respond(http.StatusOK, `{}`))
		_ = get(t, srv.URL)
	})
	// the subtest finished, so its t.Cleanup has closed the server
	if _, err := http.Get(srv.URL); err == nil {
		t.Error("server still accepting requests after its test finished")
	}
}
