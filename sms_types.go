package africastalkinggo

type SendSmsRequest struct {
	Username     string   `json:"username"`
	PhoneNumbers []string `json:"phone_numbers"`
	Message      string   `json:"message"`
	SenderId     string   `json:"senderId"`
	Enqueue      int16    `json:"enqueue"`
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
