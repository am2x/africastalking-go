package africastalkinggo

import (
	"github.com/AllanM007/africastalking-go/internal/transport"
)

// africa's talking object used to interact with api services
type AfricasTalking struct {
	Username    string
	ApiKey      string
	Environment string
	BaseUrl     string
	Client      *transport.HttpClient
}

func New() *AfricasTalking {
	return &AfricasTalking{}
}
