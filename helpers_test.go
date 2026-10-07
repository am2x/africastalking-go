package africastalking

import (
	"net/http"
	"testing"

	"github.com/allanmuiruri/africastalking-go/internal/testutil"
)

func newTestClient(t testing.TB, h http.Handler) (*AfricasTalking, *testutil.Server) {
	t.Helper()
	srv := testutil.NewServer(t, h)
	at, err := New(
		WithCredentials("testuser", "test-api-key"),
		WithBaseURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return at, srv
}
