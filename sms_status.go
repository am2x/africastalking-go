package africastalking

import (
	"fmt"
	"slices"
	"strings"
)

// SMSStatusCode is the per-recipient delivery status Africa's Talking returns
// for each number in a send request. It is deliberately a distinct type from an HTTP status:
// the numbers overlap but mean entirely different things
type SMSStatusCode int

// The per-recipient status codes documented by Africa's Talking. 1xx means
// the message was taken for delivery, 4xx that the recipient was refused, and
// 5xx that the carrier gateway failed. Prefer the Accepted, Rejected and
// GatewayFailed helpers over comparing codes directly, and note that these
// overlap numerically with HTTP statuses while meaning something unrelated.
const (
	SMSStatusProcessed             SMSStatusCode = 100
	SMSStatusSent                  SMSStatusCode = 101
	SMSStatusQueued                SMSStatusCode = 102
	SMSStatusRiskHold              SMSStatusCode = 401
	SMSStatusInvalidSenderID       SMSStatusCode = 402
	SMSStatusInvalidPhoneNumber    SMSStatusCode = 403
	SMSStatusUnsupportedNumberType SMSStatusCode = 404
	SMSStatusInsufficientBalance   SMSStatusCode = 405
	SMSStatusUserInBlacklist       SMSStatusCode = 406
	SMSStatusCouldNotRoute         SMSStatusCode = 407
	SMSStatusDoNotDisturbRejection SMSStatusCode = 409
	SMSStatusInternalServerError   SMSStatusCode = 500
	SMSStatusGatewayError          SMSStatusCode = 501
	SMSStatusRejectedByGateway     SMSStatusCode = 502
)

var smsStatusNames = map[SMSStatusCode]string{
	SMSStatusProcessed:             "Processed",
	SMSStatusSent:                  "Sent",
	SMSStatusQueued:                "Queued",
	SMSStatusRiskHold:              "RiskHold",
	SMSStatusInvalidSenderID:       "InvalidSenderId",
	SMSStatusInvalidPhoneNumber:    "InvalidPhoneNumber",
	SMSStatusUnsupportedNumberType: "UnsupportedNumberType",
	SMSStatusInsufficientBalance:   "InsufficientBalance",
	SMSStatusUserInBlacklist:       "UserInBlacklist",
	SMSStatusCouldNotRoute:         "CouldNotRoute",
	SMSStatusDoNotDisturbRejection: "DoNotDisturbRejection",
	SMSStatusInternalServerError:   "InternalServerError",
	SMSStatusGatewayError:          "GatewayError",
	SMSStatusRejectedByGateway:     "RejectedByGateway",
}

func (c SMSStatusCode) String() string {
	if name, ok := smsStatusNames[c]; ok {
		return name
	}
	return fmt.Sprintf("SMSStatusCode(%d)", int(c))
}

// Accepted reports whether the message was taken for delivery.
func (c SMSStatusCode) Accepted() bool { return c >= 100 && c < 200 }

// Rejected reports whether the recipient was permanently refused. Resending without changes will fail again.
func (c SMSStatusCode) Rejected() bool { return c >= 400 && c < 500 }

// GatewayFailed reports a failure on the carrier's side.
func (c SMSStatusCode) GatewayFailed() bool { return c >= 500 }

// Accepted returns the recipient(s) messages that were delivered
func (r SendSMSResponse) Accepted() []SMSRecipient {
	var out []SMSRecipient
	for _, rec := range r.SMSMessageData.Recipients {
		if rec.StatusCode.Accepted() {
			out = append(out, rec)
		}
	}
	return out
}

// Failed returns the recipient(s) messages that weren't delivered
func (r SendSMSResponse) Failed() []SMSRecipient {
	var out []SMSRecipient
	for _, rec := range r.SMSMessageData.Recipients {
		if !rec.StatusCode.Accepted() {
			out = append(out, rec)
		}
	}
	return out
}

// Err reports that no recipient accepted the message. It returns nil when at
// least one recipient was accepted, and nil when the response carried no
// recipients at all.
func (r SendSMSResponse) Err() error {
	failed := r.Failed()
	if len(failed) == 0 || len(failed) != len(r.SMSMessageData.Recipients) {
		return nil
	}
	return &SMSRejectedError{Recipients: failed}
}

// SMSRejectedError reports that Africa's Talking accepted the request but refused every recipient in it.
type SMSRejectedError struct {
	Recipients []SMSRecipient
}

func (e *SMSRejectedError) Error() string {
	if len(e.Recipients) == 1 {
		r := e.Recipients[0]
		return fmt.Sprintf("africastalking: message rejected for %s: %s", r.Number, r.StatusCode)
	}

	counts := map[SMSStatusCode]int{}
	for _, r := range e.Recipients {
		counts[r.StatusCode]++
	}
	codes := make([]SMSStatusCode, 0, len(counts))
	for c := range counts {
		codes = append(codes, c)
	}
	slices.Sort(codes)

	parts := make([]string, 0, len(codes))
	for _, c := range codes {
		parts = append(parts, fmt.Sprintf("%s (%d)", c, counts[c]))
	}
	return fmt.Sprintf("africastalking: all %d recipients rejected: %s", len(e.Recipients), strings.Join(parts, ", "))
}
