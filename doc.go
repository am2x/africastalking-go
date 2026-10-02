// Package africastalking is a Go SDK for the Africa's Talking API.
//
// It is an unofficial, community-maintained library and is not affiliated with
// or endorsed by Africa's Talking.
//
// # Getting started
//
// Build a client once and share it. It is safe for concurrent use by multiple
// goroutines, and holds no mutable state after construction.
//
//	at, err := africastalking.New(
//	        africastalking.WithCredentials(os.Getenv("AT_USERNAME"), os.Getenv("AT_API_KEY")),
//	        africastalking.WithSandbox(),
//	)
//	if err != nil {
//	        log.Fatal(err)
//	}
//
//	resp, err := at.SMS.Send(ctx, africastalking.SendSMSRequest{
//	        SenderID:     "MYSHORTCODE",
//	        Message:      "Hello from Go",
//	        PhoneNumbers: []string{"+254700000000"},
//	})
//
// WithSandbox points every product at Africa's Talking' sandbox hosts; omit it
// for production. Every call takes a context and honours its deadline and
// cancellation.
//
// # Errors
//
// Input rejected before any network call returns [ValidationErr]. A request the
// API rejects returns [APIError], which carries the status code, endpoint and
// response body:
//
//	var apiErr *africastalking.APIError
//	if errors.As(err, &apiErr) {
//	        log.Printf("status %d: %s", apiErr.StatusCode, apiErr.Message)
//	}
//
// Broad conditions can be matched with errors.Is against [ErrUnauthorized],
// [ErrRateLimited], [ErrBadRequest] and [ErrServer].
//
// # Per-recipient results
//
// A nil error from [SMSService.Send] means Africa's Talking accepted the
// request, not that every recipient received the message. Each recipient
// carries its own [SMSStatusCode]:
//
//	for _, r := range resp.Failed() {
//	        log.Printf("%s: %s", r.Number, r.StatusCode)
//	}
//
// Final delivery is reported asynchronously through a delivery-report webhook.
package africastalking
