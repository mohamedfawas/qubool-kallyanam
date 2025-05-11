package validation

import (
	"regexp"
	"strings"
)

// Regular expression for validating email format
// This regex ensures the email follows a standard format like "example@domain.com"
// Breakdown of the regex:
// ^                   : Start of the string
// [a-zA-Z0-9._%+\-]+  : Matches one or more of letters, digits, dot (.), underscore (_), percent (%), plus (+), or hyphen (-)
// @                   : Matches the '@' symbol
// [a-zA-Z0-9.\-]+     : Matches the domain part before the dot, e.g., "gmail", "yahoo", etc.
// \.                  : Escaped dot character
// [a-zA-Z]{2,}        : Matches the top-level domain with at least 2 letters (e.g., "com", "org")
// $                   : End of the string
//
// ✅ Examples of valid emails:
// - user@example.com
// - my.name123@domain.co
// - a_b-c+xyz@mail-server.org
//
// ❌ Examples of invalid emails:
// - user@.com        (invalid domain name)
// - @domain.com      (missing username)
// - user@domain      (missing top-level domain)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail validates if the provided email has a valid format
func ValidateEmail(email string) bool {
	email = strings.TrimSpace(email)
	return emailRegex.MatchString(email)
}
