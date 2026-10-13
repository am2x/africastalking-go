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

const (
	smsPath        string = "/version1/messaging/bulk"
	smsPremiumPath string = "/version1/messaging"
)

type smsPayload struct {
	Username     string   `json:"username"`
	Message      string   `json:"message"`
	PhoneNumbers []string `json:"phoneNumbers,omitempty"`
	SenderID     string   `json:"senderId"`
	MaskedNumber string   `json:"maskedNumber,omitempty"`
	Telco        string   `json:"telco,omitempty"`
	Enqueue      int8     `json:"enqueue"`
}

type smsPremiumPayload struct {
	Username             string   `json:"username"`
	To                   []string `json:"to"`
	Message              string   `json:"message"`
	From                 string   `json:"from"`
	Keyword              string   `json:"keyword,omitempty"`
	Enqueue              int8     `json:"enqueue,omitempty"`
	LinkID               string   `json:"linkId,omitempty"`
	RetryDurationInHours int      `json:"retryDurationInHours,omitempty"`
	RequestID            string   `json:"requestId,omitempty"`
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
		return SendSMSResponse{}, &ValidationErr{Field: "SenderID", Message: "senderID is required"}
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

// SendPremium submits a message to one or more recipients
func (s *SMSService) SendPremium(ctx context.Context, sms SendPremiumSMSRequest) (SendSMSResponse, error) {
	if sms.Message == "" {
		return SendSMSResponse{}, &ValidationErr{Field: "Message", Message: "message is required"}
	}

	if sms.From == "" {
		return SendSMSResponse{}, &ValidationErr{Field: "From", Message: "from is required"}
	}

	if len(sms.To) == 0 {
		return SendSMSResponse{}, &ValidationErr{Field: "To", Message: "at least one recipient is required"}
	}

	payload := smsPremiumPayload{
		Username:             s.at.userName,
		To:                   sms.To,
		Message:              sms.Message,
		From:                 sms.From,
		Keyword:              sms.From,
		Enqueue:              boolToInt(sms.Enqueue),
		LinkID:               sms.LinkID,
		RetryDurationInHours: sms.RetryDurationInHours,
		RequestID:            sms.RequestID,
	}

	body, err := encoding.EncodeJSON(payload)
	if err != nil {
		return SendSMSResponse{}, err
	}

	smsPremiumReq := transport.Request{
		Method:      http.MethodPost,
		URL:         s.at.apiURL(smsPremiumPath),
		ContentType: transport.ContentTypeJSON,
		Body:        body,
		Retryable:   false,
	}

	resp, err := s.at.client.Do(ctx, smsPremiumReq, s.at.apiKey)
	if err != nil {
		return SendSMSResponse{}, fmt.Errorf("africastalking: send premium sms: %w", err)
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
