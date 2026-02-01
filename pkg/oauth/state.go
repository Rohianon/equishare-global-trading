package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	stateKeyPrefix = "oauth_state:"
	stateTTL       = 10 * time.Minute
)

// redisStateStore implements StateStore using Redis.
type redisStateStore struct {
	client *redis.Client
}

// NewStateStore creates a new Redis-backed state store.
func NewStateStore(client *redis.Client) StateStore {
	return &redisStateStore{client: client}
}

// Generate creates a new cryptographically secure state with metadata.
func (s *redisStateStore) Generate(ctx context.Context, metadata StateMetadata) (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(bytes)

	// Serialize metadata
	data, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal state metadata: %w", err)
	}

	// Store in Redis with TTL
	key := stateKeyPrefix + state
	if err := s.client.Set(ctx, key, string(data), stateTTL).Err(); err != nil {
		return "", fmt.Errorf("failed to store state: %w", err)
	}

	return state, nil
}

// Validate checks if a state is valid and returns its metadata.
// The state is consumed (deleted) after validation.
func (s *redisStateStore) Validate(ctx context.Context, state string) (*StateMetadata, error) {
	key := stateKeyPrefix + state

	// Get and delete atomically using GETDEL
	data, err := s.client.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("invalid or expired state")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to validate state: %w", err)
	}

	// Deserialize metadata
	var metadata StateMetadata
	if err := json.Unmarshal([]byte(data), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state metadata: %w", err)
	}

	return &metadata, nil
}

// GenerateNonce creates a random nonce for ID token validation.
func GenerateNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
