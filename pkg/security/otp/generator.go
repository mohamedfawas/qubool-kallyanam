package otp

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// Config holds configuration for OTP generation and validation
type Config struct {
	Length     int
	ExpiryTime time.Duration // How long the OTP is valid for
}

// DefaultConfig returns a default configuration for OTP
func DefaultConfig() Config {
	return Config{
		Length:     6,
		ExpiryTime: 5 * time.Minute,
	}
}

// Generator handles OTP generation
type Generator struct {
	config Config
}

// NewGenerator creates a new OTP generator with provided config
func NewGenerator(config Config) *Generator {
	return &Generator{
		config: config,
	}
}

// Generate creates a new numeric OTP of specified length
func (g *Generator) Generate() (string, error) {
	maxNum := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(g.config.Length)), nil)
	n, err := rand.Int(rand.Reader, maxNum)
	if err != nil {
		return "", err
	}

	// Format with leading zeros if needed
	format := fmt.Sprintf("%%0%dd", g.config.Length)
	return fmt.Sprintf(format, n), nil
}
