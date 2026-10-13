package encoding

import (
	"net/url"
	"strings"
	"testing"
)

type recipient struct {
	PhoneNumber string `json:"phoneNumber"`
	Amount      string `json:"amount"`
}

func TestEncodeForm(t *testing.T) {
	type formPayload struct {
		UserName string `form:"username"`
		SenderID string `form:"senderId"`
		Numbers  string `form:"numbers"`
	}

	type optionalPayload struct {
		UserName string `form:"username"`
		SenderID string `form:"senderId,omitempty"`
		Enqueue  *bool  `form:"enqueue,omitempty"`
	}

	type listPayload struct {
		UserName string   `form:"username"`
		Numbers  []string `form:"phoneNumbers"`
	}

	type skipPayload struct {
		UserName string `form:"username"`
		Internal string `form:"-"`
		secret   string //nolint:unused // asserts unexported fields are skipped
	}

	type badTagPayload struct {
		SenderID string `form:"senderId,omitemtpy"`
	}

	type noJSONOptionPayload struct {
		Username   string      `form:"username"`
		Recipients []recipient `form:"recipients"`
	}

	type noJSONTagPayload struct {
		Username     string
		SenderID     string
		PhoneNumbers string
	}

	type floatPayload struct {
		Amount float64 `form:"amount"`
	}
	falseVal := false

	tests := []struct {
		testName string // description of this test case
		// Named input parameters for target function.
		v       any
		want    string
		wantErr string
	}{
		{
			testName: "all fields encoded, keys sorted",
			v:        formPayload{"user1", "CODE123", "254712345678"},
			want:     "numbers=254712345678&senderId=CODE123&username=user1",
		}, {
			testName: "omitempty skips zero values",
			v:        optionalPayload{UserName: "user1"},
			want:     "username=user1",
		}, {
			testName: "omitempty keeps a pointer to false",
			v:        optionalPayload{UserName: "user1", Enqueue: &falseVal},
			want:     "enqueue=false&username=user1",
		}, {
			testName: "string slice is comma joined",
			v:        listPayload{"user1", []string{"+254700000000", "+254711111111"}},
			want:     "phoneNumbers=%2B254700000000%2C%2B254711111111&username=user1",
		}, {
			testName: "dash tag and unexported fields are skipped",
			v:        skipPayload{UserName: "user1", Internal: "nope", secret: "nope"},
			want:     "username=user1",
		}, {
			testName: "non-struct input",
			v:        "not a struct",
			wantErr:  "expects a struct, got string",
		}, {
			testName: "nil pointer input",
			v:        (*formPayload)(nil),
			wantErr:  "form encode of nil pointer",
		}, {
			testName: "unknown tag option",
			v:        badTagPayload{SenderID: "CODE123"},
			wantErr:  `unknown form tag option "omitemtpy"`,
		}, {
			testName: "float is rejected so money never drifts",
			v:        floatPayload{0.1},
			wantErr:  "unsupported kind: float64",
		}, {
			testName: "no json tags in payload",
			v:        noJSONOptionPayload{"user1", []recipient{{"+254711111111", "KES 100"}}},
			wantErr:  "requires the `json` tag option",
		}, {
			testName: "untagged fields fall back to the Go field name",
			v:        noJSONTagPayload{Username: "user1", SenderID: "CODE123", PhoneNumbers: "+254711111111"},
			want:     "PhoneNumbers=%2B254711111111&SenderID=CODE123&Username=user1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			got, err := EncodeForm(tt.v)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("EncodeForm() succeeded, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("EncodeForm error =%q, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("EncodeForm() unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("EncodeForm() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestEncodeFormJSONOption(t *testing.T) {
	type airtimePayload struct {
		UserName   string      `form:"username"`
		Recipients []recipient `form:"recipients,json"`
	}

	got, err := EncodeForm(airtimePayload{
		UserName: "user1",
		Recipients: []recipient{
			{"+254700000000", "KES 100"},
			{"+254711111111", "KES 50"},
		},
	})
	if err != nil {
		t.Fatalf("EncodeForm() unexpected error: %v", err)
	}

	values, err := url.ParseQuery(string(got))
	if err != nil {
		t.Fatalf("output is not valid form encoding: %v", err)
	}
	if username := values.Get("username"); username != "user1" {
		t.Errorf("username = %q, want %q", username, "user1")
	}

	// the whole point of the `json` option: the field arrives as a JSON string
	const wantJSON = `[{"phoneNumber":"+254700000000","amount":"KES 100"},` +
		`{"phoneNumber":"+254711111111","amount":"KES 50"}]`
	if gotJSON := values.Get("recipients"); gotJSON != wantJSON {
		t.Errorf("recipients JSON\n got = %s\nwant = %s", gotJSON, wantJSON)
	}
}
