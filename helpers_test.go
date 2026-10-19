package africastalking

import (
	"net/http"
	"testing"

	"github.com/allanmuiruri/africastalking-go/internal/testutil"
)

const (
	testAPIKey string = "test-api-key"
	testUser   string = "testuser"
	testLinkID string = "SampleLinkId123"
)

func newTestClient(t testing.TB, h http.Handler) (*AfricasTalking, *testutil.Server) {
	t.Helper()
	srv := testutil.NewServer(t, h)
	at, err := New(
		WithCredentials(testUser, testAPIKey),
		WithBaseURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return at, srv
}
