package strings

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
)

var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phoneRegex    = regexp.MustCompile(`^\+?[0-9]{10,15}$`)
	urlRegex      = regexp.MustCompile(`^(https?:\/\/)?(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,30}$`)

	// Common weak passwords (this is just a small sample - in production use a proper library)
	weakPasswords = map[string]bool{
		"password": true,
		"123456":   true,
		"qwerty":   true,
		"admin":    true,
	}
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// IsEmpty checks if a string is empty or contains only whitespace
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsEmail validates an email address
func IsEmail(email string) bool {
	if IsEmpty(email) {
		return false
	}

	// Use regex for basic validation
	if !emailRegex.MatchString(email) {
		return false
	}

	// Additional validation using mail.ParseAddress
	_, err := mail.ParseAddress(email)
	return err == nil
}

// IsValidPhone validates a phone number
func IsValidPhone(phone string) bool {
	if IsEmpty(phone) {
		return false
	}

	// Clean the phone number
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")

	return phoneRegex.MatchString(phone)
}

// IsValidURL validates a URL
func IsValidURL(url string) bool {
	if IsEmpty(url) {
		return false
	}

	return urlRegex.MatchString(url)
}

// IsValidUsername validates a username
func IsValidUsername(username string) bool {
	if IsEmpty(username) {
		return false
	}

	return usernameRegex.MatchString(username)
}

// IsStrongPassword validates password strength
func IsStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	// Check if it's a common weak password
	if weakPasswords[strings.ToLower(password)] {
		return false
	}

	// Check for character variety
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	// Require at least 3 out of 4 character types
	count := 0
	if hasUpper {
		count++
	}
	if hasLower {
		count++
	}
	if hasDigit {
		count++
	}
	if hasSpecial {
		count++
	}

	return count >= 3
}

// ValidatePasswordWithReason validates a password and returns a reason if invalid
func ValidatePasswordWithReason(password string) (bool, string) {
	if len(password) < 8 {
		return false, "Password must be at least 8 characters long"
	}

	if weakPasswords[strings.ToLower(password)] {
		return false, "Password is too common and easily guessed"
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	missing := []string{}
	if !hasUpper {
		missing = append(missing, "uppercase letter")
	}
	if !hasLower {
		missing = append(missing, "lowercase letter")
	}
	if !hasDigit {
		missing = append(missing, "number")
	}
	if !hasSpecial {
		missing = append(missing, "special character")
	}

	if len(missing) > 1 {
		return false, fmt.Sprintf("Password must include at least one %s and one %s",
			strings.Join(missing[:len(missing)-1], ", "), missing[len(missing)-1])
	} else if len(missing) == 1 {
		return false, fmt.Sprintf("Password must include at least one %s", missing[0])
	}

	return true, ""
}

// IsValidName validates a person's name
func IsValidName(name string) bool {
	if IsEmpty(name) {
		return false
	}

	// Name must be at least 2 characters long
	if len(strings.TrimSpace(name)) < 2 {
		return false
	}

	// Name should only contain letters, spaces, hyphens, and apostrophes
	for _, r := range name {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' && r != '\'' {
			return false
		}
	}

	return true
}
