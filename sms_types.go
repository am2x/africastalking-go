package africastalking

// SendSMSRequest is the input to [SMSService.Send]. Message, SenderID and at
// least one entry in PhoneNumbers are required.
type SendSMSRequest struct {
	// Message is the text to deliver.
	Message string
	// SenderID is the registered alphanumeric sender ID or short code.
	SenderID string
	// MaskedNumber is optional phone number field and only applies to Safaricom numbers in Kenya.
	MaskedNumber string
	// Telco is optional and only applies to masked/hashed safaricom numbers and should be specified when the [MaskedNumber] value is present.
	Telco string
	// Enqueue asks Africa's Talking to queue the message rather than attempt
	// immediate delivery. Useful for large bulk sends.
	Enqueue bool
	// PhoneNumbers are the recipients in international format, e.g. "+254700000000".
	PhoneNumbers []string
}

// SendPremiumSMSRequest is the input to [SMSService.SendPremium]. Message, From and at
// least one entry in To are required.
type SendPremiumSMSRequest struct {
	// Message is the text to deliver.
	Message string
	// From is the registered alphanumeric sender ID or short code.
	From string
	// Enqueue asks Africa's Talking to queue the message rather than attempt
	// immediate delivery. Useful for large bulk sends.
	Enqueue bool
	// To are the recipients in international format, e.g. "+254700000000".
	To []string
	// Keyword is used for premium service.
	Keyword string
	// LinkID is used for premium services to send OnDemand messages. The linkId is forwarded to your application when the user sends a message to your service.
	LinkID string
	// RetryDurationInHours specifies the number of hours your subscription message should be retried in case it’s not delivered to the subscriber.
	RetryDurationInHours int
	// RequestId is a client specified request identifier. We add it return it as part of the http dlr callback.
	RequestID string
}

// SendSMSResponse is the result of [SMSService.Send]. Use [SendSMSResponse.Accepted]
// and [SendSMSResponse.Failed] to inspect per-recipient outcomes rather than
// reading Recipients directly.
type SendSMSResponse struct {
	SMSMessageData SMSMessageData `json:"SMSMessageData"`
}

// SMSMessageData holds the summary message and the per-recipient results.
type SMSMessageData struct {
	Message    string         `json:"Message"`
	Recipients []SMSRecipient `json:"Recipients"`
}

// SMSRecipient is the outcome for a single recipient. StatusCode is the
// authoritative result; Status is Africa's Talking' display text for it.
type SMSRecipient struct {
	StatusCode SMSStatusCode `json:"statusCode"`
	Number     string        `json:"number"`
	Status     string        `json:"status"`
	Cost       string        `json:"cost"`
	MessageID  string        `json:"messageId"`
}

// Africa's Talking expects enqueue as 0 or 1 rather than a boolean.
func boolToInt(b bool) int8 {
	if b {
		return 1
	}
	return 0
}
