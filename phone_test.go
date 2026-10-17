package africastalking

import "testing"

func Test_normalizePhoneNumbers(t *testing.T) {
	tests := []struct {
		testName string // description of this test case
		// Named input parameters for target function.
		s       string
		want    string
		wantErr bool
	}{
		{
			testName: "well formatted number",
			s:        "+254711123456",
			want:     "+254711123456",
			wantErr:  false,
		}, {
			testName: "number with spaces",
			s:        "+254 711 123 456",
			want:     "+254711123456",
			wantErr:  false,
		}, {
			testName: "hyphenated number",
			s:        "+254-711-123-456",
			want:     "+254711123456",
			wantErr:  false,
		}, {
			testName: "additional prefix number",
			s:        "00254711123456",
			want:     "+254711123456",
			wantErr:  false,
		}, {
			testName: "no country code number",
			s:        "0711123456",
			wantErr:  true,
		}, {
			testName: "missing + number",
			s:        "254711123456",
			wantErr:  true,
		}, {
			testName: "number with non-digit characters",
			s:        "+254711abc456",
			wantErr:  true,
		}, {
			testName: "empty number",
			s:        "",
			wantErr:  true,
		}, {
			testName: "only + character number",
			s:        "+",
			wantErr:  true,
		}, {
			testName: "too many digits number",
			s:        "+2547111123456789",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			got, gotErr := normalizePhoneNumber(tt.s)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("normalizePhoneNumbers() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Errorf("normalizedPhoneNumber() = %q, want an error", got)
			}
			if got != tt.want {
				t.Errorf("normalizePhoneNumbers() = %v, want %v", got, tt.want)
			}
		})
	}
}
