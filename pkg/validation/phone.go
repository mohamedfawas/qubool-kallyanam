package validation

import (
	"regexp"
	"strings"
)

// Regular expression for validating international phone number format
//
// Regex Explanation:
// ^              : Start of the string
// \+             : Matches the literal '+' sign (required for international format)
// [1-9]{1}       : The first digit after '+' must be 1-9 (avoids leading 0)
// [0-9]{3,14}    : Matches the remaining digits (minimum 3, maximum 14) to allow country codes and phone numbers
// $              : End of the string
//
// ✅ Valid examples:
// - +14155552671     (US number)
// - +919876543210    (Indian number)
// - +447911123456    (UK number)
//
// ❌ Invalid examples:
// - 14155552671      (missing '+')
// - +0123456789      (starts with 0 after '+')
// - +123             (too short to be a valid number)
// - +1234567890123456 (too long, exceeds 15 digits)
var phoneRegex = regexp.MustCompile(`^\+[1-9]{1}[0-9]{3,14}$`)

// ValidatePhone validates if the provided phone has a valid international format
func ValidatePhone(phone string) bool {
	phone = strings.TrimSpace(phone)
	return phoneRegex.MatchString(phone)
}
