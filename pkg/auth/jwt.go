package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// =============================================================================
// JWT Token Management
// =============================================================================
// This package provides JWT token generation and validation with support for:
// - Access tokens (short-lived, for API authentication)
// - Refresh tokens (long-lived, for obtaining new access tokens)
// - Session tracking (refresh tokens are bound to sessions for revocation)
// =============================================================================

// Config holds JWT configuration
type Config struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// Claims represents JWT token claims
type Claims struct {
	UserID    string `json:"user_id"`
	Phone     string `json:"phone"`
	SessionID string `json:"session_id,omitempty"` // Only in refresh tokens
	TokenType string `json:"token_type"`           // "access" or "refresh"
	jwt.RegisteredClaims
}

// TokenPair contains access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // Access token TTL in seconds
	SessionID    string `json:"-"`          // Internal use for session tracking
}

// JWTManager handles JWT operations
type JWTManager struct {
	config       *Config
	sessionStore SessionStore
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(cfg *Config) *JWTManager {
	if cfg.AccessTokenTTL == 0 {
		cfg.AccessTokenTTL = 15 * time.Minute
	}
	if cfg.RefreshTokenTTL == 0 {
		cfg.RefreshTokenTTL = 7 * 24 * time.Hour
	}
	return &JWTManager{config: cfg}
}

// WithSessionStore adds session store for token revocation support
func (m *JWTManager) WithSessionStore(store SessionStore) *JWTManager {
	m.sessionStore = store
	return m
}

// GenerateTokenPair creates a new access/refresh token pair
func (m *JWTManager) GenerateTokenPair(userID, phone string) (*TokenPair, error) {
	return m.GenerateTokenPairWithSession(context.Background(), userID, phone, "", "")
}

// GenerateTokenPairWithSession creates tokens with session tracking
func (m *JWTManager) GenerateTokenPairWithSession(ctx context.Context, userID, phone, userAgent, ipAddress string) (*TokenPair, error) {
	// Generate session ID for tracking
	sessionID, err := GenerateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Generate access token (no session ID, short-lived)
	accessToken, err := m.generateAccessToken(userID, phone)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token (with session ID, long-lived)
	refreshToken, err := m.generateRefreshToken(userID, phone, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store session if store is configured
	if m.sessionStore != nil {
		session := &Session{
			ID:        sessionID,
			UserID:    userID,
			TokenHash: HashToken(refreshToken),
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(m.config.RefreshTokenTTL),
			UserAgent: userAgent,
			IPAddress: ipAddress,
		}
		if err := m.sessionStore.Create(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(m.config.AccessTokenTTL.Seconds()),
		SessionID:    sessionID,
	}, nil
}

// RefreshTokens validates a refresh token and issues new tokens
func (m *JWTManager) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Validate the refresh token
	claims, err := m.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// If session store is configured, validate and rotate session
	if m.sessionStore != nil {
		tokenHash := HashToken(refreshToken)
		session, err := m.sessionStore.Validate(ctx, claims.SessionID, tokenHash)
		if err != nil {
			return nil, fmt.Errorf("failed to validate session: %w", err)
		}
		if session == nil {
			return nil, fmt.Errorf("session not found or token already rotated")
		}

		// Generate new refresh token for rotation
		newRefreshToken, err := m.generateRefreshToken(claims.UserID, claims.Phone, claims.SessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to generate new refresh token: %w", err)
		}

		// Rotate the session with new token hash
		newExpiry := time.Now().Add(m.config.RefreshTokenTTL)
		if err := m.sessionStore.Rotate(ctx, claims.SessionID, HashToken(newRefreshToken), newExpiry); err != nil {
			return nil, fmt.Errorf("failed to rotate session: %w", err)
		}

		// Generate new access token
		accessToken, err := m.generateAccessToken(claims.UserID, claims.Phone)
		if err != nil {
			return nil, fmt.Errorf("failed to generate access token: %w", err)
		}

		return &TokenPair{
			AccessToken:  accessToken,
			RefreshToken: newRefreshToken,
			ExpiresIn:    int(m.config.AccessTokenTTL.Seconds()),
			SessionID:    claims.SessionID,
		}, nil
	}

	// Without session store, just generate new tokens (less secure)
	return m.GenerateTokenPair(claims.UserID, claims.Phone)
}

// RevokeSession invalidates a session (logout)
func (m *JWTManager) RevokeSession(ctx context.Context, sessionID string) error {
	if m.sessionStore == nil {
		return nil // No session store, nothing to revoke
	}
	return m.sessionStore.Revoke(ctx, sessionID)
}

// RevokeAllSessions invalidates all sessions for a user (logout everywhere)
func (m *JWTManager) RevokeAllSessions(ctx context.Context, userID string) error {
	if m.sessionStore == nil {
		return nil
	}
	return m.sessionStore.RevokeAllForUser(ctx, userID)
}

// ListSessions returns all active sessions for a user
func (m *JWTManager) ListSessions(ctx context.Context, userID string) ([]*Session, error) {
	if m.sessionStore == nil {
		return nil, nil
	}
	return m.sessionStore.ListForUser(ctx, userID)
}

// ValidateToken validates an access token and returns claims
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.Secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token specifically
func (m *JWTManager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("not a refresh token")
	}

	return claims, nil
}

// =============================================================================
// Internal Token Generation
// =============================================================================

func (m *JWTManager) generateAccessToken(userID, phone string) (string, error) {
	claims := &Claims{
		UserID:    userID,
		Phone:     phone,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "equishare",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.Secret))
}

func (m *JWTManager) generateRefreshToken(userID, phone, sessionID string) (string, error) {
	// Generate unique JWT ID for token uniqueness (prevents identical tokens when rotated)
	jti, err := GenerateSessionID()
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT ID: %w", err)
	}

	claims := &Claims{
		UserID:    userID,
		Phone:     phone,
		SessionID: sessionID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti, // Unique token identifier
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.RefreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "equishare",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.Secret))
}

// GetConfig returns the JWT configuration (for TTL info)
func (m *JWTManager) GetConfig() *Config {
	return m.config
}
