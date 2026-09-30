package africastalking

import (
	"errors"

	"github.com/AllanM007/africastalking-go/internal/transport"
)

// africa's talking object used to interact with api services
type AfricasTalking struct {
	userName string
	apiKey   string
	hosts    hosts
	client   *transport.HTTPClient
	SMS      *SMSService
	// logger
}

func (at *AfricasTalking) apiURL(path string) string      { return at.hosts.api + path }
func (at *AfricasTalking) voiceURL(path string) string    { return at.hosts.voice + path }
func (at *AfricasTalking) paymentsURL(path string) string { return at.hosts.payments + path }

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
