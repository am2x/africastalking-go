package africastalkinggo

import "errors"

const sandboxUrl string = "https://api.sandbox.africastalking.com/version1"
const productionUrl string = "https://api.africastalking.com/version1"

const (
	SmsPath     string = "/messaging/bulk"
	AirtimePath string = "/airtime/send"
)

type Credentials struct {
	Username string
	ApiKey   string
}

// verify credentials
func WithCredentials(username, apiKey string) (Credentials, error) {
	if username == "" || apiKey == "" {
		return Credentials{}, errors.New("missing africa's talking credentials")
	}

	return Credentials{Username: username, ApiKey: apiKey}, nil
}

// set base url based on user's sandbox option
func WithSandbox(sandbox bool) string {
	if sandbox {
		return sandboxUrl
	} else {
		return productionUrl
	}
}
func WithHTTPClient() {}
func WithBaseURL() {

}                    //escape hatch/testing
func WithRetry()     {}
func WithUserAgent() {}
func WithLogger()    {}
