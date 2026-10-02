package africastalking

type SendSMSRequest struct {
	Message      string
	MaskedNumber string
	Telco        string
	SenderID     string
	Enqueue      bool
	PhoneNumbers []string
}

type SendSMSResponse struct {
	SMSMessageData SMSMessageData `json:"SMSMessageData"`
}

type SMSMessageData struct {
	Message    string         `json:"Message"`
	Recipients []SMSRecipient `json:"Recipients"`
}

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
