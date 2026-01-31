package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Rohianon/equishare-global-trading/pkg/cache"
	"github.com/Rohianon/equishare-global-trading/services/ussd-service/internal/types"
)

const (
	SessionTTL    = 3 * time.Minute
	SessionPrefix = "ussd:session:"
)

type Manager struct {
	cache *cache.RedisCache
}

func NewManager(cache *cache.RedisCache) *Manager {
	return &Manager{cache: cache}
}

func (m *Manager) Get(ctx context.Context, sessionID string) (*types.Session, error) {
	key := SessionPrefix + sessionID

	data, err := m.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if data == "" {
		return nil, nil
	}

	var session types.Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

func (m *Manager) Save(ctx context.Context, session *types.Session) error {
	key := SessionPrefix + session.SessionID
	session.UpdatedAt = time.Now()

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := m.cache.Set(ctx, key, string(data), SessionTTL); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func (m *Manager) Delete(ctx context.Context, sessionID string) error {
	key := SessionPrefix + sessionID
	return m.cache.Delete(ctx, key)
}

func (m *Manager) GetOrCreate(ctx context.Context, sessionID, phoneNumber string) (*types.Session, error) {
	session, err := m.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if session != nil {
		return session, nil
	}

	session = types.NewSession(sessionID, phoneNumber)
	if err := m.Save(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}
