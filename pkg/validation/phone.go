package validation

import (
	"regexp"
	"strings"
)

// Regular expression for validating international phone number format
var phoneRegex = regexp.MustCompile(`^\+[1-9]{1}[0-9]{3,14}$`)

// ValidatePhone validates if the provided phone has a valid international format
func ValidatePhone(phone string) bool {
	phone = strings.TrimSpace(phone)
	return phoneRegex.MatchString(phone)
}
