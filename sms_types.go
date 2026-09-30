package africastalking

type SendSmsRequest struct {
	Message      string   `json:"message"`
	MaskedNumber string   `json:"maskedNumber,omitempty"`
	Telco        string   `json:"telco,omitempty"`
	SenderId     string   `json:"senderId"`
	Enqueue      int16    `json:"enqueue,omitempty"`
	PhoneNumbers []string `json:"phoneNumbers"`
}

type SendSmsResponse struct {
	SmsMessageData struct {
		Message    string `json:"Message"`
		Recipients []struct {
			StatusCode int32  `json:"statusCode"`
			Number     string `json:"number"`
			Status     string `json:"status"`
			Cost       string `json:"cost"`
			MessageId  string `json:"messageId"`
		} `json:"Recipients"`
	} `json:"SMSMessageData"`
}
