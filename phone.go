package africastalking

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Separators callers commonly include; stripping them changes formatting, not data.
	phoneCleaner = strings.NewReplacer("-", "", " ", "", "(", "", ")", "", ".", "")

	// E.164: a leading non-zero digit then up to 14 more, 15 digits maximum.
	phoneE164 = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
)

func normalizePhoneNumber(s string) (string, error) {
	cleaned := phoneCleaner.Replace(s)

	if strings.HasPrefix(cleaned, "00") {
		cleaned = "+" + cleaned[2:]
	}

	if !phoneE164.MatchString(cleaned) {
		return "", fmt.Errorf("africastalking: incorrect phone number format: %s (expected E.164, e.g. +254711123456)", s)
	}

	return cleaned, nil
}
