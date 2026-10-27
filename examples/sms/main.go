// Main package contains a sample sdk integration a user would setup on their client config
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/am2x/africastalking-go"
)

func main() {
	at, err := africastalking.New(
		africastalking.WithCredentials(os.Getenv("USERNAME"), os.Getenv("APIKEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := at.SMS.Send(context.Background(), africastalking.SendSMSRequest{
		Message:      "Hello",
		SenderID:     "MYSHORTCODE",
		PhoneNumbers: []string{"+254712345678"},
	})
	if err != nil {
		log.Print("error sending message: %w", err)
		return
	}

	for _, v := range resp.Accepted() {
		fmt.Printf("queued for %s(%s)\n", v.Number, v.MessageID)
	}
	for _, v := range resp.Failed() {
		fmt.Printf("not sent to %s: %s\n", v.Number, v.StatusCode)
	}

	if err := resp.Err(); err != nil {
		log.Print(err)
	}
}
