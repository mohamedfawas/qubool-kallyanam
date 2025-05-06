package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Role represents user roles
type Role string

const (
	RoleUser        Role = "USER"
	RolePremiumUser Role = "PREMIUM_USER"
	RoleAdmin       Role = "ADMIN"
)

// ServiceAccess defines which services a user can access
type ServiceAccess struct {
	Auth  bool `json:"auth"`
	User  bool `json:"user"`
	Chat  bool `json:"chat"`
	Admin bool `json:"admin"`
}

// GetServiceAccessByRole returns service access permissions based on role
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
			Chat:  true,
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

// Config holds JWT configuration
type Config struct {
	SecretKey       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Issuer          string
}

// Claims represents JWT claims with user info and permissions
type Claims struct {
	UserID       uint          `json:"user_id"`
	Role         Role          `json:"role"`
	Services     ServiceAccess `json:"services"`
	Verified     bool          `json:"verified"`
	PremiumUntil *int64        `json:"premium_until,omitempty"`
	UserIDString string        `json:"user_id_string,omitempty"` // Add this field
	jwt.RegisteredClaims
}

// Manager handles JWT operations
type Manager struct {
	config Config
}

// NewManager creates a new JWT manager
func NewManager(config Config) *Manager {
	return &Manager{
		config: config,
	}
}

// GenerateAccessToken generates a new JWT access token
func (m *Manager) GenerateAccessToken(userID uint, role Role, verified bool, premiumUntil *int64, userIDString string) (string, error) {
	now := time.Now()
	services := GetServiceAccessByRole(role)

	claims := &Claims{
		UserID:       userID,
		Role:         role,
		Services:     services,
		Verified:     verified,
		PremiumUntil: premiumUntil,
		UserIDString: userIDString, // Set the UUID string in claims
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.config.Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.SecretKey))
}

// GenerateRefreshToken generates a new JWT refresh token
func (m *Manager) GenerateRefreshToken(userID uint, userIDString string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:       userID,
		UserIDString: userIDString, // Set the UUID string in claims
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.RefreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.config.Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.SecretKey))
}

// ValidateToken validates a JWT token and returns claims
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// CheckServiceAccess checks if the user has access to a specific service
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
