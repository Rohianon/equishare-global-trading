package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// =============================================================================
// Session Management
// =============================================================================
// Sessions track active refresh tokens for users. When a refresh token is used,
// it is rotated (old one invalidated, new one issued). On logout, the session
// is revoked immediately.
//
// Redis Key Schema:
//   - session:{session_id} -> user_id (TTL = refresh token TTL)
//   - user_sessions:{user_id} -> set of session_ids (for logout all)
// =============================================================================

// Session represents an active user session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"token_hash"` // Hash of refresh token
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	UserAgent string    `json:"user_agent,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`
}

// SessionStore defines the interface for session storage
type SessionStore interface {
	// Create creates a new session and returns the session ID
	Create(ctx context.Context, session *Session) error

	// Get retrieves a session by ID
	Get(ctx context.Context, sessionID string) (*Session, error)

	// Validate checks if a session exists and the token hash matches
	Validate(ctx context.Context, sessionID, tokenHash string) (*Session, error)

	// Rotate updates the token hash for a session (token rotation)
	Rotate(ctx context.Context, sessionID, newTokenHash string, newExpiry time.Time) error

	// Revoke invalidates a specific session
	Revoke(ctx context.Context, sessionID string) error

	// RevokeAllForUser invalidates all sessions for a user
	RevokeAllForUser(ctx context.Context, userID string) error

	// ListForUser returns all active sessions for a user
	ListForUser(ctx context.Context, userID string) ([]*Session, error)
}

// RedisSessionStore implements SessionStore using Redis
type RedisSessionStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisSessionStore creates a new Redis-based session store
func NewRedisSessionStore(client *redis.Client, refreshTokenTTL time.Duration) *RedisSessionStore {
	return &RedisSessionStore{
		client: client,
		ttl:    refreshTokenTTL,
	}
}

func sessionKey(sessionID string) string {
	return fmt.Sprintf("session:%s", sessionID)
}

func userSessionsKey(userID string) string {
	return fmt.Sprintf("user_sessions:%s", userID)
}

func (s *RedisSessionStore) Create(ctx context.Context, session *Session) error {
	pipe := s.client.Pipeline()

	// Store session data
	key := sessionKey(session.ID)
	pipe.HSet(ctx, key,
		"user_id", session.UserID,
		"token_hash", session.TokenHash,
		"created_at", session.CreatedAt.Unix(),
		"expires_at", session.ExpiresAt.Unix(),
		"user_agent", session.UserAgent,
		"ip_address", session.IPAddress,
	)
	pipe.ExpireAt(ctx, key, session.ExpiresAt)

	// Add to user's session set
	pipe.SAdd(ctx, userSessionsKey(session.UserID), session.ID)
	pipe.ExpireAt(ctx, userSessionsKey(session.UserID), session.ExpiresAt.Add(24*time.Hour))

	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisSessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	key := sessionKey(sessionID)
	data, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil // Session not found
	}

	return parseSessionData(sessionID, data)
}

func (s *RedisSessionStore) Validate(ctx context.Context, sessionID, tokenHash string) (*Session, error) {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil // Session not found
	}

	// Check token hash matches
	if session.TokenHash != tokenHash {
		return nil, nil // Token doesn't match (possibly reused after rotation)
	}

	// Check expiry
	if time.Now().After(session.ExpiresAt) {
		// Clean up expired session
		s.Revoke(ctx, sessionID)
		return nil, nil
	}

	return session, nil
}

func (s *RedisSessionStore) Rotate(ctx context.Context, sessionID, newTokenHash string, newExpiry time.Time) error {
	key := sessionKey(sessionID)

	pipe := s.client.Pipeline()
	pipe.HSet(ctx, key, "token_hash", newTokenHash, "expires_at", newExpiry.Unix())
	pipe.ExpireAt(ctx, key, newExpiry)
	_, err := pipe.Exec(ctx)

	return err
}

func (s *RedisSessionStore) Revoke(ctx context.Context, sessionID string) error {
	// Get session to find user ID
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	pipe := s.client.Pipeline()

	// Delete session
	pipe.Del(ctx, sessionKey(sessionID))

	// Remove from user's session set
	if session != nil {
		pipe.SRem(ctx, userSessionsKey(session.UserID), sessionID)
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisSessionStore) RevokeAllForUser(ctx context.Context, userID string) error {
	// Get all session IDs for user
	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey(userID)).Result()
	if err != nil {
		return err
	}

	if len(sessionIDs) == 0 {
		return nil
	}

	pipe := s.client.Pipeline()

	// Delete all sessions
	for _, sessionID := range sessionIDs {
		pipe.Del(ctx, sessionKey(sessionID))
	}

	// Delete the set
	pipe.Del(ctx, userSessionsKey(userID))

	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisSessionStore) ListForUser(ctx context.Context, userID string) ([]*Session, error) {
	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey(userID)).Result()
	if err != nil {
		return nil, err
	}

	sessions := make([]*Session, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		session, err := s.Get(ctx, sessionID)
		if err != nil {
			continue
		}
		if session != nil {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

func parseSessionData(sessionID string, data map[string]string) (*Session, error) {
	var createdAt, expiresAt int64
	fmt.Sscanf(data["created_at"], "%d", &createdAt)
	fmt.Sscanf(data["expires_at"], "%d", &expiresAt)

	return &Session{
		ID:        sessionID,
		UserID:    data["user_id"],
		TokenHash: data["token_hash"],
		CreatedAt: time.Unix(createdAt, 0),
		ExpiresAt: time.Unix(expiresAt, 0),
		UserAgent: data["user_agent"],
		IPAddress: data["ip_address"],
	}, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// GenerateSessionID creates a cryptographically secure session ID
func GenerateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken creates a SHA-256 hash of a token for storage
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
