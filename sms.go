package africastalking

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/AllanM007/africastalking-go/internal/encoding"
	"github.com/AllanM007/africastalking-go/internal/transport"
)

// sms entity used to interact with africa's talking sms api
type SMSService struct{ at *AfricasTalking }

const smsPath string = "/messaging/bulk"

type smsPayload struct {
	Username     string   `json:"username"`
	Message      string   `json:"message"`
	PhoneNumbers []string `json:"phoneNumbers"`
	SenderId     string   `json:"senderId"`
}

func (s *SMSService) Send(ctx context.Context, sms SendSmsRequest) (SendSmsResponse, error) {
	payload := smsPayload{
		Username:     s.at.userName,
		Message:      sms.Message,
		PhoneNumbers: sms.PhoneNumbers,
		SenderId:     sms.SenderId,
	}

	body, err := encoding.EncodeJSON(payload)
	if err != nil {
		return SendSmsResponse{}, err
	}

	smsReq := transport.Request{
		Method:      http.MethodPost,
		URL:         s.at.apiURL(smsPath),
		ContentType: transport.ContentTypeJSON,
		Body:        body,
		UserAgent:   transport.UserAgent,
		Retryable:   false,
	}
	resp, err := s.at.client.Do(ctx, smsReq, s.at.apiKey)
	if err != nil {
		return SendSmsResponse{}, fmt.Errorf("africastalking: send sms: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return SendSmsResponse{}, newAPIError(smsPath, resp)
	}
	var data SendSmsResponse
	if err := json.Unmarshal(resp.Body, &data); err != nil {
		return SendSmsResponse{}, fmt.Errorf("africastalking: decode sms response: %w", err)
	}

	return data, nil
}
