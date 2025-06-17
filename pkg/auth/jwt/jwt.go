package jwt

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
)

// Role represents user roles
type Role string

const (
	RoleUser        Role = "USER"
	RolePremiumUser Role = "PREMIUM_USER"
	RoleAdmin       Role = "ADMIN"
)

// ServiceAccess defines which parts (services) of the system a user can use.
type ServiceAccess struct {
	Auth  bool `json:"auth"`
	User  bool `json:"user"`
	Chat  bool `json:"chat"`
	Admin bool `json:"admin"`
}

// GetServiceAccessByRole returns what services a user can access based on their role.
func GetServiceAccessByRole(role Role) ServiceAccess {
	switch role {
	case RoleAdmin:
		return ServiceAccess{
			Auth:  true,
			User:  true,
			Chat:  true,
			Admin: true,
		}
	case RolePremiumUser:
		return ServiceAccess{
			Auth:  true,
			User:  true,
			Chat:  true,
			Admin: false,
		}
	case RoleUser:
		return ServiceAccess{
			Auth:  true,
			User:  true,
			Chat:  false,
			Admin: false,
		}
	default:
		return ServiceAccess{
			Auth:  false,
			User:  false,
			Chat:  false,
			Admin: false,
		}
	}
}

// Config contains configuration used to create and validate tokens.
type Config struct {
	SecretKey       string        // Secret used to sign the token (keep this safe!)
	AccessTokenTTL  time.Duration // How long the access token is valid (e.g. 15 minutes)
	RefreshTokenTTL time.Duration // How long the refresh token is valid (e.g. 7 days)
	Issuer          string        // The name or source of the token (usually your app name)
}

// Claims represents the data stored inside a JWT.
// This includes user ID, role, services allowed, etc.
type Claims struct {
	UserID               string        `json:"user_id"`  // UUID string identifier , Example: "123e4567-e89b-12d3-a456-426614174000"
	Role                 Role          `json:"role"`     // USER, PREMIUM_USER, or ADMIN
	Services             ServiceAccess `json:"services"` // Services user is allowed to access
	jwt.RegisteredClaims               // Includes standard JWT fields like expiration, issued at, etc.
}

// Manager is the main struct that handles all JWT-related operations.
type Manager struct {
	config Config
}

// NewManager creates a new JWT manager
func NewManager(config Config) *Manager {
	return &Manager{
		config: config,
	}
}

// Add this helper function
func generateTokenID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Update your GenerateAccessToken method
func (m *Manager) GenerateAccessToken(userID string,
	role Role,
	verified bool,
	premiumUntil *int64) (string, error) {

	now := indianstandardtime.Now()
	services := GetServiceAccessByRole(role)

	// Create the claims
	claims := &Claims{
		UserID:   userID,
		Role:     role,
		Services: services,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        generateTokenID(), // ADD THIS LINE
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.config.Issuer,
		},
	}

	// Create a new token using HMAC SHA256 signing method
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token using the secret key and return it
	return token.SignedString([]byte(m.config.SecretKey))
}

// Update your GenerateRefreshToken method
func (m *Manager) GenerateRefreshToken(userID string) (string, error) {
	now := indianstandardtime.Now()

	// Refresh token only stores minimal info: just the user ID
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        generateTokenID(), // ADD THIS LINE
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.RefreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.config.Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.SecretKey))
}

// ValidateToken takes a token string and verifies its validity.
// If valid, it returns the claims (user data inside).
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	// Parse and validate the token with claims
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Ensure the token was signed with HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.SecretKey), nil
	})

	// If the token failed to parse or verify (e.g., invalid signature, expired, tampered), return an error.
	if err != nil {
		return nil, err
	}

	// Additional validation in case the token parsed successfully but is still considered invalid.
	// Example: token was manually created with correct structure but wrong signature.
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// CheckServiceAccess determines if a user has permission to access a given service.
// Example: If a user tries to access "admin" panel, this function checks if allowed.
func (m *Manager) CheckServiceAccess(claims *Claims, service string) bool {
	switch service {
	case "auth":
		return claims.Services.Auth
	case "user":
		return claims.Services.User
	case "chat":
		return claims.Services.Chat
	case "admin":
		return claims.Services.Admin
	default:
		return false
	}
}
