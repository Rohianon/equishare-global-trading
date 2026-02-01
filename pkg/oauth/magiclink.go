package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

// MagicLinkStore defines the storage interface for magic link tokens.
// This allows different storage backends (Redis, PostgreSQL, etc.).
type MagicLinkStore interface {
	// Store saves a magic link token with its metadata.
	Store(ctx context.Context, tokenHash string, info *MagicLinkInfo, expiry time.Duration) error

	// Get retrieves magic link info by token hash.
	Get(ctx context.Context, tokenHash string) (*MagicLinkInfo, error)

	// Delete removes a token (after use or expiry).
	Delete(ctx context.Context, tokenHash string) error
}

// magicLinkProvider implements MagicLinkProvider for passwordless email auth.
type magicLinkProvider struct {
	store   MagicLinkStore
	baseURL string
	expiry  time.Duration
}

// NewMagicLinkProvider creates a new magic link provider.
func NewMagicLinkProvider(store MagicLinkStore, baseURL string, expiry time.Duration) MagicLinkProvider {
	if expiry == 0 {
		expiry = 15 * time.Minute
	}
	return &magicLinkProvider{
		store:   store,
		baseURL: baseURL,
		expiry:  expiry,
	}
}

// GenerateToken creates a new magic link token for the given email.
func (p *magicLinkProvider) GenerateToken(ctx context.Context, email string, userID *string) (string, error) {
	// Generate 32 random bytes for token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)

	// Hash token for storage (don't store raw token)
	tokenHash := hashToken(token)

	// Create token info
	info := &MagicLinkInfo{
		Email:    email,
		UserID:   userID,
		IssuedAt: time.Now().Unix(),
	}

	// Store hashed token
	if err := p.store.Store(ctx, tokenHash, info, p.expiry); err != nil {
		return "", fmt.Errorf("failed to store token: %w", err)
	}

	return token, nil
}

// VerifyToken validates a magic link token and returns associated info.
func (p *magicLinkProvider) VerifyToken(ctx context.Context, token string) (*MagicLinkInfo, error) {
	tokenHash := hashToken(token)

	info, err := p.store.Get(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired token")
	}

	return info, nil
}

// MarkUsed marks a token as used (deletes it for one-time use).
func (p *magicLinkProvider) MarkUsed(ctx context.Context, token string) error {
	tokenHash := hashToken(token)
	return p.store.Delete(ctx, tokenHash)
}

// GetMagicLinkURL returns the full magic link URL for the given token.
func (p *magicLinkProvider) GetMagicLinkURL(token string) string {
	return fmt.Sprintf("%s?token=%s", p.baseURL, token)
}

// hashToken creates a SHA256 hash of the token for secure storage.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// RedisMagicLinkStore implements MagicLinkStore using Redis.
type RedisMagicLinkStore struct {
	client interface {
		Set(ctx context.Context, key string, value any, expiration time.Duration) error
		Get(ctx context.Context, key string) (string, error)
		Del(ctx context.Context, keys ...string) error
	}
	keyPrefix string
}

// redisClient is a minimal interface for Redis operations.
type redisClient interface {
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, keys ...string) error
}

// NewRedisMagicLinkStore creates a Redis-backed magic link store.
func NewRedisMagicLinkStore(client redisClient) *RedisMagicLinkStore {
	return &RedisMagicLinkStore{
		client:    client,
		keyPrefix: "magic_link:",
	}
}

func (s *RedisMagicLinkStore) key(tokenHash string) string {
	return s.keyPrefix + tokenHash
}

func (s *RedisMagicLinkStore) Store(ctx context.Context, tokenHash string, info *MagicLinkInfo, expiry time.Duration) error {
	// Serialize info as JSON-like string
	value := fmt.Sprintf("%s|%d", info.Email, info.IssuedAt)
	if info.UserID != nil {
		value = fmt.Sprintf("%s|%d|%s", info.Email, info.IssuedAt, *info.UserID)
	}
	return s.client.Set(ctx, s.key(tokenHash), value, expiry)
}

func (s *RedisMagicLinkStore) Get(ctx context.Context, tokenHash string) (*MagicLinkInfo, error) {
	value, err := s.client.Get(ctx, s.key(tokenHash))
	if err != nil {
		return nil, err
	}

	// Parse the stored value
	info := &MagicLinkInfo{}
	parts := splitN(value, "|", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid stored value")
	}

	info.Email = parts[0]
	fmt.Sscanf(parts[1], "%d", &info.IssuedAt)
	if len(parts) == 3 && parts[2] != "" {
		userID := parts[2]
		info.UserID = &userID
	}

	return info, nil
}

func (s *RedisMagicLinkStore) Delete(ctx context.Context, tokenHash string) error {
	return s.client.Del(ctx, s.key(tokenHash))
}

// splitN splits a string by separator into at most n parts.
func splitN(s, sep string, n int) []string {
	result := make([]string, 0, n)
	for i := 0; i < n-1; i++ {
		idx := indexOf(s, sep)
		if idx < 0 {
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	result = append(result, s)
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
