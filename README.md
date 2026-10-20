#  Africa's Talking Go SDK


Africa's Talking Go SDK is a comprehensive Go library for interacting with the Africa's Talking API, a platform that provides various communication and payment services across Africa.

## Features


- **SMS**: Send and receive SMS messages to and from any network in Africa
- **Voice**: Make and receive voice calls, play media, and control the call flow
- **USSD**: Create interactive menus and sessions for your users
- **Airtime**: Send airtime to mobile subscribers across Africa
- **Payments**: Accept and disburse payments via mobile money, card, and bank


## Installation

To install the SDK, you need to have Go 1.23 or higher installed on your machine. You also need to have an account with Africa's Talking and get an API key and a username.

- Install the SDK using the go get command:

```bash
go get github.com/am2x/africastalking-go
```

- Import the SDK in your code using the import statement:

```go
import "github.com/am2x/africastalking-go"
```

## Usage

To use the SDK, you need to create a client object and pass it your username and API key. You can then use the client object to access the different services and their methods. For example, you can use something like this:

```go
package main

import (
	"fmt"
	"log"
	"errors"
	"context"
	"net/http"

	"github.com/am2x/africastalking-go"
)

func main() {
	// Create a client object with your options.
	at, err := africastalking.New(
		africastalking.WithCredentials("YOUR_USERNAME", "YOUR_API_KEY"),
		africastalking.WithSandbox(), // define whether to use sandbox env or live api.
	)
	if err != nil {
		// ...
	}

	ctx := context.Background()

	resp, err := at.SMS.Send(ctx, africastalking.SendSMSRequest{
		PhoneNumbers:      []string{"+254711XXXYYY"},
		Message: "Hello",
		SenderID:    "MYSHORTCODE",
	})
	
	var apiErr *africastalking.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized {
		// ...
	}

}
```

## Testing

To test the SDK, you need to have a test account with Africa's Talking. You also need to set some environment variables for the test configuration.

- Create a test account with Africa's Talking and get a test API key and a test username
- Set the following environment variables:

```bash
export AT_TEST_API_KEY=YOUR_TEST_API_KEY # The API key for your test account
export AT_TEST_USERNAME=YOUR_TEST_USERNAME # The username for your test account
```

- Run the tests using the go test command and pass it the -cover flag:

```bash
go test -cover ./...
```
