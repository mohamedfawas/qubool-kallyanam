package helpers

import (
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
	autherrors "github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/errors"
)

// ValidateRegistrationInput validates user registration input
func ValidateRegistrationInput(email, phone, password string) error {
	if !validation.ValidateEmail(email) {
		return autherrors.ErrInvalidInput
	}
	if !validation.ValidatePhone(phone) {
		return autherrors.ErrInvalidInput
	}
	if !validation.ValidatePassword(password, validation.DefaultPasswordPolicy()) {
		return autherrors.ErrInvalidInput
	}
	return nil
}

// ValidateLoginInput validates login input
func ValidateLoginInput(email, password string) error {
	if email == "" || password == "" {
		return autherrors.ErrInvalidInput
	}
	if !validation.ValidateEmail(email) {
		return autherrors.ErrInvalidInput
	}
	return nil
}
