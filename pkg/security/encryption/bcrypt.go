package encryption

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrHashFailed = errors.New("failed to hash password")
)

// Default cost for bcrypt
const DefaultCost = 12

// HashPassword generates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", ErrHashFailed
	}
	return string(hashedBytes), nil
}

// VerifyPassword checks if the provided password matches the hashed password
func VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
