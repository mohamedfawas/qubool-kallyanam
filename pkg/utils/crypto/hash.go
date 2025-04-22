package crypto

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinCost is the minimum bcrypt cost factor
	MinCost = bcrypt.MinCost
	// DefaultCost is the recommended cost factor for most applications
	DefaultCost = 12
	// MaxCost is the maximum bcrypt cost factor
	MaxCost = bcrypt.MaxCost
)

var (
	// ErrHashFailed is returned when password hashing fails
	ErrHashFailed = errors.New("failed to hash password")
	// ErrMismatchedPassword is returned when a password doesn't match the hash
	ErrMismatchedPassword = errors.New("password does not match hash")
	// ErrInvalidHash is returned when a hash is in an invalid format
	ErrInvalidHash = errors.New("invalid hash format")
)

// HashPassword hashes a plaintext password using bcrypt with the default cost factor
func HashPassword(password string) (string, error) {
	return HashPasswordWithCost(password, DefaultCost)
}

// HashPasswordWithCost hashes a plaintext password using bcrypt with a specified cost factor
func HashPasswordWithCost(password string, cost int) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	// Validate cost factor
	if cost < MinCost {
		cost = MinCost
	} else if cost > MaxCost {
		cost = MaxCost
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrHashFailed, err)
	}

	return string(hash), nil
}

// VerifyPassword compares a plaintext password against a bcrypt hash
func VerifyPassword(hashedPassword, password string) error {
	if hashedPassword == "" || password == "" {
		return errors.New("password or hash cannot be empty")
	}

	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrMismatchedPassword
		}
		return fmt.Errorf("%w: %v", ErrInvalidHash, err)
	}

	return nil
}

// NeedsRehash checks if a password hash should be rehashed with a new cost factor
func NeedsRehash(hashedPassword string, cost int) bool {
	if hashedPassword == "" {
		return false
	}

	// Extract the cost from the hash
	hashCost, err := bcrypt.Cost([]byte(hashedPassword))
	if err != nil {
		return true // If we can't determine the cost, rehash
	}

	return hashCost != cost
}
