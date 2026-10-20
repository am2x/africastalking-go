package africastalking

import (
	"errors"

	"github.com/am2x/africastalking-go/internal/transport"
)

// AfricasTalking is a client for the Africa's Talking API. Create one with
// [New] and share it; it is safe for concurrent use by multiple goroutines.
type AfricasTalking struct {
	userName string
	apiKey   string
	hosts    hosts
	client   *transport.HTTPClient
	SMS      *SMSService
	// logger
}

func (at *AfricasTalking) apiURL(path string) string     { return at.hosts.api + path }
func (at *AfricasTalking) contentURL(path string) string { return at.hosts.content + path }

// func (at *AfricasTalking) voiceURL(path string) string    { return at.hosts.voice + path }
// func (at *AfricasTalking) paymentsURL(path string) string { return at.hosts.payments + path }

// New returns a client configured by opts. It returns an error if credentials
// are missing or any option is invalid. Requests go to the production hosts
// unless [WithSandbox] is passed.
func New(opts ...Option) (*AfricasTalking, error) {
	at := &AfricasTalking{
		hosts:  productionHosts,
		client: transport.New(),
	}

	for _, opt := range opts {
		if err := opt(at); err != nil {
			return nil, err
		}
	}

	if at.userName == "" || at.apiKey == "" {
		return nil, errors.New("africastalking: missing credentials")
	}
	at.SMS = &SMSService{at: at}
	return at, nil
}
