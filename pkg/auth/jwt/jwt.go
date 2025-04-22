package jwt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// Common errors
var (
	ErrInvalidToken    = errors.New("invalid token")
	ErrExpiredToken    = errors.New("token has expired")
	ErrMissingClaims   = errors.New("missing required claims")
	ErrInvalidIssuer   = errors.New("invalid token issuer")
	ErrInvalidAudience = errors.New("invalid token audience")
)

// Role defines user roles in the system
type Role string

// Available roles
const (
	RoleAdmin Role = "ADMIN"
	RoleUser  Role = "USER"
)

// Claims represents the custom JWT claims for our application
type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
	Role   Role   `json:"role"`
}

// ClaimsKey is used to store/retrieve claims from context
type ClaimsKey struct{}

// Config holds JWT configuration
type Config struct {
	SigningKey     string        `yaml:"signing_key"`
	Issuer         string        `yaml:"issuer"`
	Audience       []string      `yaml:"audience"`
	AccessTimeout  time.Duration `yaml:"access_timeout"`
	RefreshTimeout time.Duration `yaml:"refresh_timeout"`
}

// Manager handles JWT operations
type Manager struct {
	config Config
	logger logging.Logger
}

// NewManager creates a new JWT manager
func NewManager(config Config, logger logging.Logger) *Manager {
	if logger == nil {
		logger = logging.Get().Named("jwt")
	}

	// Set default timeouts if not specified
	if config.AccessTimeout == 0 {
		config.AccessTimeout = 15 * time.Minute
	}
	if config.RefreshTimeout == 0 {
		config.RefreshTimeout = 7 * 24 * time.Hour
	}

	return &Manager{
		config: config,
		logger: logger,
	}
}

// GenerateAccessToken creates a new JWT access token
func (m *Manager) GenerateAccessToken(userID string, role Role) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.config.AccessTimeout)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.Issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings(m.config.Audience),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateTokenID(),
		},
		UserID: userID,
		Role:   role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(m.config.SigningKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// GenerateRefreshToken creates a new JWT refresh token
func (m *Manager) GenerateRefreshToken(userID string, role Role) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.config.RefreshTimeout)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.Issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings(m.config.Audience),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateTokenID(),
		},
		UserID: userID,
		Role:   role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(m.config.SigningKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates a JWT token and returns the claims
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.SigningKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrMissingClaims
	}

	// Validate issuer if configured
	if m.config.Issuer != "" && claims.Issuer != m.config.Issuer {
		return nil, ErrInvalidIssuer
	}

	// Validate audience if configured
	if len(m.config.Audience) > 0 {
		// Just check if any configured audience is in the token
		validAudience := false
		for _, aud := range m.config.Audience {
			if contains(claims.Audience, aud) {
				validAudience = true
				break
			}
		}
		if !validAudience {
			return nil, ErrInvalidAudience
		}
	}

	return claims, nil
}

// RefreshAccessToken generates a new access token from a valid refresh token
func (m *Manager) RefreshAccessToken(refreshToken string) (string, error) {
	claims, err := m.ValidateToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Generate new access token
	return m.GenerateAccessToken(claims.UserID, claims.Role)
}

// GenerateTokenPair generates both access and refresh tokens
func (m *Manager) GenerateTokenPair(userID string, role Role) (accessToken, refreshToken string, err error) {
	accessToken, err = m.GenerateAccessToken(userID, role)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = m.GenerateRefreshToken(userID, role)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// ClaimsFromContext extracts claims from context
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsKey{}).(*Claims)
	return claims, ok
}

// ContextWithClaims adds claims to context
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, ClaimsKey{}, claims)
}

// helper functions

// generateTokenID creates a unique token ID
func generateTokenID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
