package africastalking

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/allanmuiruri/africastalking-go/internal/encoding"
	"github.com/allanmuiruri/africastalking-go/internal/transport"
)

// SMSService is used to interact with africa's talking sms api
type SMSService struct{ at *AfricasTalking }

const smsPath string = "/messaging/bulk"

type smsPayload struct {
	Username     string   `json:"username"`
	Message      string   `json:"message"`
	PhoneNumbers []string `json:"phoneNumbers,omitempty"`
	SenderID     string   `json:"senderId"`
	MaskedNumber string   `json:"maskedNumber,omitempty"`
	Telco        string   `json:"telco,omitempty"`
	Enqueue      int8     `json:"enqueue"`
}

// Send submits a message to one or more recipients.
//
// A nil error means Africa's Talking accepted the request, not that every
// recipient received the message. Each recipient carries its own
// SMSStatusCode — use Accepted() and Failed() on the response to inspect
// them. Final delivery is only known via the delivery-report webhook.
func (s *SMSService) Send(ctx context.Context, sms SendSMSRequest) (SendSMSResponse, error) {

	if sms.Message == "" {
		return SendSMSResponse{}, &ValidationErr{Field: "Message", Message: "message is required"}
	}
	if sms.SenderID == "" {
		return SendSMSResponse{}, &ValidationErr{Field: "SenderID", Message: "senderId is required"}
	}

	if len(sms.PhoneNumbers) == 0 {
		return SendSMSResponse{}, &ValidationErr{Field: "PhoneNumbers", Message: "at least one recipient is required"}
	}

	payload := smsPayload{
		Username:     s.at.userName,
		Message:      sms.Message,
		PhoneNumbers: sms.PhoneNumbers,
		SenderID:     sms.SenderID,
		MaskedNumber: sms.MaskedNumber,
		Telco:        sms.Telco,
		Enqueue:      boolToInt(sms.Enqueue),
	}

	body, err := encoding.EncodeJSON(payload)
	if err != nil {
		return SendSMSResponse{}, err
	}

	smsReq := transport.Request{
		Method:      http.MethodPost,
		URL:         s.at.apiURL(smsPath),
		ContentType: transport.ContentTypeJSON,
		Body:        body,
		UserAgent:   transport.UserAgent,
		Retryable:   false, // sms sends should not be retryable to avoid double-sending/spending
	}
	resp, err := s.at.client.Do(ctx, smsReq, s.at.apiKey)
	if err != nil {
		return SendSMSResponse{}, fmt.Errorf("africastalking: send sms: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return SendSMSResponse{}, newAPIError(smsPath, resp)
	}
	var data SendSMSResponse
	if err := json.Unmarshal(resp.Body, &data); err != nil {
		return SendSMSResponse{}, fmt.Errorf("africastalking: decode sms response: %w", err)
	}

	return data, nil
}
