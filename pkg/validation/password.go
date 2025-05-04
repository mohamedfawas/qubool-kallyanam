package validation

import (
	"unicode"
)

// PasswordPolicy defines the requirements for a valid password
type PasswordPolicy struct {
	MinLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireNumber  bool
	RequireSpecial bool
	MaxLength      int
}

// DefaultPasswordPolicy returns the default password policy
func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:      8,
		RequireUpper:   true,
		RequireLower:   true,
		RequireNumber:  true,
		RequireSpecial: true,
		MaxLength:      100,
	}
}

// ValidatePassword checks if the password meets the specified policy
func ValidatePassword(password string, policy PasswordPolicy) bool {
	if len(password) < policy.MinLength {
		return false
	}

	if policy.MaxLength > 0 && len(password) > policy.MaxLength {
		return false
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	return (!policy.RequireUpper || hasUpper) &&
		(!policy.RequireLower || hasLower) &&
		(!policy.RequireNumber || hasNumber) &&
		(!policy.RequireSpecial || hasSpecial)
}
