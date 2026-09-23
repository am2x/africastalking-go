package transport

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// http client used to interact with africa's talking(internal use only!)
type HttpClient struct {
	Client     *http.Client
	Breaker    *CircuitBreaker
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Wait       func(time.Duration)
}

func New() *HttpClient {
	return &HttpClient{
		Client: &http.Client{
			Timeout: 1 * time.Minute,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
			},
		},
	}
}

func (c *HttpClient) Do(baseUrl, apiKey string, data io.Reader) []byte {
	req, err := http.NewRequest("POST", baseUrl, data)
	if err != nil {
		// log.Fatal(err)
		log.Printf("error building new request: %v", err.Error())
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apiKey", apiKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		// log.Fatal(err)
		log.Printf("error making http request: %v", err.Error())
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		// log.Fatal(err)
		log.Printf("error parsing request body: %v", err.Error())
	}
	fmt.Printf("%s\n", bodyText)
	return bodyText
}
