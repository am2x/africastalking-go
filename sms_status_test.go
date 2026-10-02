package africastalking

import (
	"encoding/json"
	"errors"
	"testing"
)

const mixedStatusBody = `{"SMSMessageData":{
      "Message":"Sent to 1/3 Total Cost: KES 0.8000",
      "Recipients":[
              {"statusCode":101,"number":"+254700000000","status":"Success","cost":"KES 0.8000","messageId":"ATXid_1"},
              {"statusCode":403,"number":"+254711111111","status":"InvalidPhoneNumber","cost":"0","messageId":""},
              {"statusCode":405,"number":"+254722222222","status":"InsufficientBalance","cost":"0","messageId":""}
			  ]
		}}`

func TestSMSStatusCode(t *testing.T) {
	t.Run("String names every documented code", func(t *testing.T) {
		for code, want := range smsStatusNames {
			if got := code.String(); got != want {
				t.Errorf("SMSStatusCode(%d).String() = %q, want %q", int(code), got, want)
			}
		}
	})
	t.Run("String falls back for unknown codes", func(t *testing.T) {
		if got := SMSStatusCode(999).String(); got != "SMSStatusCode(999)" {
			t.Errorf("String() = %q, want %q", got, "SMSStatusCode(999)")
		}
	})
	t.Run("buckets", func(t *testing.T) {
		tests := []struct {
			code          SMSStatusCode
			accepted      bool
			rejected      bool
			gatewayFailed bool
		}{
			{SMSStatusProcessed, true, false, false},
			{SMSStatusSent, true, false, false},
			{SMSStatusQueued, true, false, false},
			{SMSStatusRiskHold, false, true, false},
			{SMSStatusInvalidPhoneNumber, false, true, false},
			{SMSStatusInsufficientBalance, false, true, false},
			{SMSStatusDoNotDisturbRejection, false, true, false},
			{SMSStatusInternalServerError, false, false, true},
			{SMSStatusGatewayError, false, false, true},
			{SMSStatusRejectedByGateway, false, false, true},
		}
		for _, tt := range tests {
			t.Run(tt.code.String(), func(t *testing.T) {
				if got := tt.code.Accepted(); got != tt.accepted {
					t.Errorf("Accepted() = %v, want %v", got, tt.accepted)
				}
				if got := tt.code.Rejected(); got != tt.rejected {
					t.Errorf("Rejected() = %v, want %v", got, tt.rejected)
				}
				if got := tt.code.GatewayFailed(); got != tt.gatewayFailed {
					t.Errorf("GatewayFailed() = %v, want %v", got, tt.gatewayFailed)
				}
			})
		}
	})
}

func TestSMSResponsePartitioning(t *testing.T) {
	var resp SendSMSResponse
	if err := json.Unmarshal([]byte(mixedStatusBody), &resp); err != nil {
		t.Fatalf("unmarshall: %v", err)
	}

	accepted, failed := resp.Accepted(), resp.Failed()
	if len(accepted) != 1 {
		t.Fatalf("len(Accepted()) = %d, want 1", len(accepted))
	}
	if len(failed) != 2 {
		t.Fatalf("len(Failed()) = %d, want 2", len(failed))
	}
	if accepted[0].Number != "+254700000000" {
		t.Errorf("accepted number = %q", accepted[0].Number)
	}
	if accepted[0].StatusCode != SMSStatusSent {
		t.Errorf("accepted code = %v, want Sent", accepted[0].StatusCode)
	}
	if failed[0].StatusCode != SMSStatusInvalidPhoneNumber {
		t.Errorf("failed[1] code = %v, want InvalidPhoneNumber", failed[0].StatusCode)
	}
	if failed[1].StatusCode != SMSStatusInsufficientBalance {
		t.Errorf("failed[2] code = %v, want InsufficientBalance", failed[1].StatusCode)
	}

	//a partial failure is not an error
	if err := resp.Err(); err != nil {
		t.Errorf("Err() = %v, want nil for a partial failure", err)
	}
}

func TestSendSMSResponseErr(t *testing.T) {
	tests := []struct {
		testName string
		body     string
		wantErr  string
	}{
		{
			testName: "nil when one recipient was accepted",
			body:     `{"SMSMessageData":{"Recipients":[{"statusCode":101,"number":"+254700000000"},{"statusCode":403,"number":"+254711111111"}]}}`,
		},
		{
			testName: "nil when there are no recipients",
			body:     `{"SMSMessageData":{"Recipients":[]}}`,
		},
		{
			testName: "single rejection names the number",
			body:     `{"SMSMessageData":{"Recipients":[{"statusCode":403,"number":"+254711111111"}]}}`,
			wantErr:  "africastalking: message rejected for +254711111111: InvalidPhoneNumber",
		},
		{
			testName: "many rejections summarise by code, sorted",
			body: `{"SMSMessageData":{"Recipients":[
                              {"statusCode":405,"number":"+254700000000"},
                              {"statusCode":403,"number":"+254711111111"},
                              {"statusCode":405,"number":"+254722222222"},
                              {"statusCode":405,"number":"+254733333333"}]}}`,
			wantErr: "africastalking: all 4 recipients rejected: InvalidPhoneNumber (1), InsufficientBalance (3)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			var resp SendSMSResponse
			if err := json.Unmarshal([]byte(tt.body), &resp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			err := resp.Err()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Err() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Err() = nil, want an error")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("Err()\n got = %q\nwant = %q", err, tt.wantErr)
			}
			var rejected *SMSRejectedError
			if !errors.As(err, &rejected) {
				t.Fatal("errors.As could not extract *SMSRejectedError")
			}
			if len(rejected.Recipients) != len(resp.SMSMessageData.Recipients) {
				t.Errorf("len(Recipients) = %d, want %d", len(rejected.Recipients), len(resp.SMSMessageData.Recipients))
			}
		})
	}
}
