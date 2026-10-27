package africastalking_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	africastalking "github.com/am2x/africastalking-go"
)

// Build the client once and share it. It is safe for concurrent use and holds
// no mutable state after construction, so a long-lived program should keep one
// rather than build a client per request.
func Example() {
	at, err := africastalking.New(
		africastalking.WithCredentials(os.Getenv("AT_USERNAME"), os.Getenv("AT_API_KEY")),
		africastalking.WithSandbox(),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := at.SMS.Send(ctx, africastalking.SendSMSRequest{
		SenderID:     "MYSHORTCODE",
		Message:      "Hello from Go",
		PhoneNumbers: []string{"+254700000000"},
	})
	if err != nil {
		// Return rather than log.Fatal: an exit here would skip the deferred
		// cancel and leak the context.
		log.Print(err)
		return
	}

	fmt.Println(resp.SMSMessageData.Message)
}

// A nil error means Africa's Talking accepted the request, not that every
// recipient received the message. Inspect the per-recipient outcomes.
func ExampleSMSService_Send() {
	at, err := africastalking.New(
		africastalking.WithCredentials(os.Getenv("AT_USERNAME"), os.Getenv("AT_API_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := at.SMS.Send(context.Background(), africastalking.SendSMSRequest{
		SenderID:     "MYSHORTCODE",
		Message:      "Your order has shipped",
		PhoneNumbers: []string{"+254700000000", "+254711111111"},
	})
	if err != nil {
		log.Print(err)
		return
	}

	// Some recipients can be refused inside an otherwise successful response.
	for _, r := range resp.Accepted() {
		fmt.Printf("queued for %s (%s)\n", r.Number, r.MessageID)
	}
	for _, r := range resp.Failed() {
		fmt.Printf("not sent to %s: %s\n", r.Number, r.StatusCode)
	}

	// Err reports the case where no recipient at all was accepted.
	if err := resp.Err(); err != nil {
		log.Print(err)
	}
}

// Enqueue asks Africa's Talking to buffer a large batch rather than attempt
// immediate delivery, which is the right choice for bulk sends.
func ExampleSMSService_Send_bulk() {
	at, err := africastalking.New(
		africastalking.WithCredentials(os.Getenv("AT_USERNAME"), os.Getenv("AT_API_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	recipients := make([]string, 0, 500)
	// ... fill recipients from your own data

	resp, err := at.SMS.Send(context.Background(), africastalking.SendSMSRequest{
		SenderID:     "MYSHORTCODE",
		Message:      "Polls close at 18:00",
		PhoneNumbers: recipients,
		Enqueue:      true,
	})
	if err != nil {
		log.Print(err)
		return
	}

	fmt.Printf("%d accepted, %d refused\n", len(resp.Accepted()), len(resp.Failed()))
}

// Premium messages are billed to the subscriber and require a keyword mapped to
// your short code. LinkID arrives on the subscription callback and ties the
// reply to an OnDemand request.
func ExampleSMSService_SendPremium() {
	at, err := africastalking.New(
		africastalking.WithCredentials(os.Getenv("AT_USERNAME"), os.Getenv("AT_API_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := at.SMS.SendPremium(context.Background(), africastalking.SendPremiumSMSRequest{
		From:    "MYSHORTCODE",
		Keyword: "QUIZ",
		Message: "Your score is 7/10",
		To:      []string{"+254700000000"},
		// LinkID comes from the subscription callback for this subscriber.
		LinkID:               "SomeLinkId",
		RetryDurationInHours: 2,
	})
	if err != nil {
		log.Print(err)
		return
	}

	fmt.Println(resp.SMSMessageData.Message)
}

// The inbox is cursor paged. Persist the cursor only after the messages are
// safely stored, since it cannot be rewound.
func ExampleSMSService_Fetch() {
	at, err := africastalking.New(
		africastalking.WithCredentials(os.Getenv("AT_USERNAME"), os.Getenv("AT_API_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	lastReceivedID := "0" // "0" starts from the beginning

	for {
		page, err := at.SMS.Fetch(context.Background(), lastReceivedID)
		if err != nil {
			log.Print(err)
			return
		}
		if len(page.SMSMessageData.Messages) == 0 {
			break // nothing new
		}

		for _, m := range page.SMSMessageData.Messages {
			fmt.Printf("%s -> %s: %s\n", m.From, m.To, m.Text)
			lastReceivedID = fmt.Sprint(m.ID)
		}
		// Store lastReceivedID here, after the messages are handled.
	}
}

// Failures separate into input this SDK rejected before sending, and a request
// Africa's Talking rejected.
func ExampleAPIError() {
	at, err := africastalking.New(
		africastalking.WithCredentials(os.Getenv("AT_USERNAME"), os.Getenv("AT_API_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = at.SMS.Send(context.Background(), africastalking.SendSMSRequest{
		SenderID:     "MYSHORTCODE",
		Message:      "Hello",
		PhoneNumbers: []string{"+254700000000"},
	})

	// Bad input never reaches the network.
	var valErr *africastalking.ValidationErr
	if errors.As(err, &valErr) {
		log.Printf("fix the request: %s", valErr.Field)
		return
	}

	// Broad conditions match with errors.Is.
	switch {
	case errors.Is(err, africastalking.ErrUnauthorized):
		log.Print("check your username and API key")
	case errors.Is(err, africastalking.ErrRateLimited):
		log.Print("back off and retry later")
	}

	// The full detail is on APIError.
	var apiErr *africastalking.APIError
	if errors.As(err, &apiErr) {
		log.Printf("%s returned %d: %s", apiErr.Endpoint, apiErr.StatusCode, apiErr.Message)
	}
}

// A recipient's status code is the authoritative result. Compare against the
// SMSStatus constants rather than the display string in Status.
func ExampleSMSStatusCode() {
	var r africastalking.SMSRecipient // from resp.Failed()

	switch {
	case r.StatusCode == africastalking.SMSStatusInsufficientBalance:
		log.Print("top up the account before retrying")
	case r.StatusCode.Rejected():
		// 4xx: resending unchanged will fail the same way.
		log.Printf("%s permanently refused: %s", r.Number, r.StatusCode)
	case r.StatusCode.GatewayFailed():
		// 5xx: the carrier failed, but the message may still have gone out.
		log.Printf("%s failed at the gateway: %s", r.Number, r.StatusCode)
	}
}
