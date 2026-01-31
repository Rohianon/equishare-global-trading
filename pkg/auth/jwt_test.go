package auth

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestJWTManager_GenerateTokenPair(t *testing.T) {
	manager := NewJWTManager(&Config{
		Secret:          "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})

	tokens, err := manager.GenerateTokenPair("user-123", "+254712345678")
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	if tokens.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}

	if tokens.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}

	if tokens.ExpiresIn != 900 {
		t.Errorf("ExpiresIn = %d, want 900", tokens.ExpiresIn)
	}

	if tokens.SessionID == "" {
		t.Error("SessionID should not be empty")
	}
}

func TestJWTManager_ValidateToken(t *testing.T) {
	manager := NewJWTManager(&Config{
		Secret:          "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})

	tokens, _ := manager.GenerateTokenPair("user-123", "+254712345678")

	claims, err := manager.ValidateToken(tokens.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("UserID = %s, want user-123", claims.UserID)
	}

	if claims.Phone != "+254712345678" {
		t.Errorf("Phone = %s, want +254712345678", claims.Phone)
	}

	if claims.TokenType != "access" {
		t.Errorf("TokenType = %s, want access", claims.TokenType)
	}
}

func TestJWTManager_ValidateRefreshToken(t *testing.T) {
	manager := NewJWTManager(&Config{
		Secret:          "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})

	tokens, _ := manager.GenerateTokenPair("user-123", "+254712345678")

	claims, err := manager.ValidateRefreshToken(tokens.RefreshToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken() error = %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("UserID = %s, want user-123", claims.UserID)
	}

	if claims.TokenType != "refresh" {
		t.Errorf("TokenType = %s, want refresh", claims.TokenType)
	}

	if claims.SessionID == "" {
		t.Error("SessionID should not be empty in refresh token")
	}
}

func TestJWTManager_ValidateRefreshToken_NotRefresh(t *testing.T) {
	manager := NewJWTManager(&Config{
		Secret:          "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})

	tokens, _ := manager.GenerateTokenPair("user-123", "+254712345678")

	// Try to validate access token as refresh token
	_, err := manager.ValidateRefreshToken(tokens.AccessToken)
	if err == nil {
		t.Error("ValidateRefreshToken() should return error for access token")
	}
}

func TestJWTManager_ValidateToken_Invalid(t *testing.T) {
	manager := NewJWTManager(&Config{
		Secret:          "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})

	_, err := manager.ValidateToken("invalid-token")
	if err == nil {
		t.Error("ValidateToken() should return error for invalid token")
	}
}

func TestJWTManager_ValidateToken_WrongSecret(t *testing.T) {
	manager1 := NewJWTManager(&Config{Secret: "secret1"})
	manager2 := NewJWTManager(&Config{Secret: "secret2"})

	tokens, _ := manager1.GenerateTokenPair("user-123", "+254712345678")

	_, err := manager2.ValidateToken(tokens.AccessToken)
	if err == nil {
		t.Error("ValidateToken() should return error for token signed with different secret")
	}
}

func TestJWTManager_DefaultTTLs(t *testing.T) {
	manager := NewJWTManager(&Config{Secret: "test-secret"})
	cfg := manager.GetConfig()

	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Errorf("Default AccessTokenTTL = %v, want 15m", cfg.AccessTokenTTL)
	}

	if cfg.RefreshTokenTTL != 7*24*time.Hour {
		t.Errorf("Default RefreshTokenTTL = %v, want 7 days", cfg.RefreshTokenTTL)
	}
}

// =============================================================================
// Session-based Tests
// =============================================================================

func TestJWTManager_WithSessionStore(t *testing.T) {
	store := NewMockSessionStore()
	manager := NewJWTManager(&Config{
		Secret:          "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	}).WithSessionStore(store)

	ctx := context.Background()
	tokens, err := manager.GenerateTokenPairWithSession(ctx, "user-123", "+254712345678", "Test Agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("GenerateTokenPairWithSession() error = %v", err)
	}

	// Verify session was created
	session, err := store.Get(ctx, tokens.SessionID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if session == nil {
		t.Fatal("Session should have been created")
	}
	if session.UserID != "user-123" {
		t.Errorf("Session.UserID = %s, want user-123", session.UserID)
	}
	if session.UserAgent != "Test Agent" {
		t.Errorf("Session.UserAgent = %s, want Test Agent", session.UserAgent)
	}
}

func TestJWTManager_RefreshTokens_WithRotation(t *testing.T) {
	store := NewMockSessionStore()
	manager := NewJWTManager(&Config{
		Secret:          "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	}).WithSessionStore(store)

	ctx := context.Background()
	tokens1, _ := manager.GenerateTokenPairWithSession(ctx, "user-123", "+254712345678", "", "")

	// Refresh tokens
	tokens2, err := manager.RefreshTokens(ctx, tokens1.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshTokens() error = %v", err)
	}

	if tokens2.RefreshToken == tokens1.RefreshToken {
		t.Error("Refresh token should be rotated")
	}

	if tokens2.SessionID != tokens1.SessionID {
		t.Error("Session ID should remain the same after rotation")
	}

	// Old refresh token should no longer work
	_, err = manager.RefreshTokens(ctx, tokens1.RefreshToken)
	if err == nil {
		t.Error("Old refresh token should be invalid after rotation")
	}

	// New refresh token should work
	tokens3, err := manager.RefreshTokens(ctx, tokens2.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshTokens() with new token error = %v", err)
	}
	if tokens3.AccessToken == "" {
		t.Error("Should get new access token")
	}
}

func TestJWTManager_RevokeSession(t *testing.T) {
	store := NewMockSessionStore()
	manager := NewJWTManager(&Config{
		Secret:          "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	}).WithSessionStore(store)

	ctx := context.Background()
	tokens, _ := manager.GenerateTokenPairWithSession(ctx, "user-123", "+254712345678", "", "")

	// Revoke session
	err := manager.RevokeSession(ctx, tokens.SessionID)
	if err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}

	// Refresh should fail after revocation
	_, err = manager.RefreshTokens(ctx, tokens.RefreshToken)
	if err == nil {
		t.Error("RefreshTokens() should fail after session is revoked")
	}
}

func TestJWTManager_RevokeAllSessions(t *testing.T) {
	store := NewMockSessionStore()
	manager := NewJWTManager(&Config{
		Secret:          "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	}).WithSessionStore(store)

	ctx := context.Background()

	// Create multiple sessions
	tokens1, _ := manager.GenerateTokenPairWithSession(ctx, "user-123", "+254712345678", "", "")
	tokens2, _ := manager.GenerateTokenPairWithSession(ctx, "user-123", "+254712345678", "", "")

	// Verify both sessions exist
	sessions, _ := manager.ListSessions(ctx, "user-123")
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}

	// Revoke all sessions
	err := manager.RevokeAllSessions(ctx, "user-123")
	if err != nil {
		t.Fatalf("RevokeAllSessions() error = %v", err)
	}

	// Both refreshes should fail
	_, err = manager.RefreshTokens(ctx, tokens1.RefreshToken)
	if err == nil {
		t.Error("RefreshTokens() should fail for token 1")
	}

	_, err = manager.RefreshTokens(ctx, tokens2.RefreshToken)
	if err == nil {
		t.Error("RefreshTokens() should fail for token 2")
	}
}

func TestJWTManager_ListSessions(t *testing.T) {
	store := NewMockSessionStore()
	manager := NewJWTManager(&Config{
		Secret:          "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	}).WithSessionStore(store)

	ctx := context.Background()

	// Create sessions for multiple users
	manager.GenerateTokenPairWithSession(ctx, "user-1", "+254712345678", "Agent 1", "1.1.1.1")
	manager.GenerateTokenPairWithSession(ctx, "user-1", "+254712345678", "Agent 2", "2.2.2.2")
	manager.GenerateTokenPairWithSession(ctx, "user-2", "+254787654321", "Agent 3", "3.3.3.3")

	sessions1, _ := manager.ListSessions(ctx, "user-1")
	if len(sessions1) != 2 {
		t.Errorf("User 1 should have 2 sessions, got %d", len(sessions1))
	}

	sessions2, _ := manager.ListSessions(ctx, "user-2")
	if len(sessions2) != 1 {
		t.Errorf("User 2 should have 1 session, got %d", len(sessions2))
	}
}

// =============================================================================
// Helper Functions Tests
// =============================================================================

func TestGenerateSessionID(t *testing.T) {
	id1, err := GenerateSessionID()
	if err != nil {
		t.Fatalf("GenerateSessionID() error = %v", err)
	}

	if len(id1) != 64 { // 32 bytes = 64 hex characters
		t.Errorf("SessionID length = %d, want 64", len(id1))
	}

	id2, _ := GenerateSessionID()
	if id1 == id2 {
		t.Error("Session IDs should be unique")
	}
}

func TestHashToken(t *testing.T) {
	token := "my-secret-token"
	hash1 := HashToken(token)
	hash2 := HashToken(token)

	if hash1 != hash2 {
		t.Error("Same token should produce same hash")
	}

	if len(hash1) != 64 { // SHA-256 = 64 hex characters
		t.Errorf("Hash length = %d, want 64", len(hash1))
	}

	hash3 := HashToken("different-token")
	if hash1 == hash3 {
		t.Error("Different tokens should produce different hashes")
	}
}

// =============================================================================
// Mock Session Store
// =============================================================================

type MockSessionStore struct {
	sessions     map[string]*Session
	userSessions map[string][]string
	mu           sync.RWMutex
}

func NewMockSessionStore() *MockSessionStore {
	return &MockSessionStore{
		sessions:     make(map[string]*Session),
		userSessions: make(map[string][]string),
	}
}

func (m *MockSessionStore) Create(ctx context.Context, session *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions[session.ID] = session
	m.userSessions[session.UserID] = append(m.userSessions[session.UserID], session.ID)
	return nil
}

func (m *MockSessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, nil
	}
	return session, nil
}

func (m *MockSessionStore) Validate(ctx context.Context, sessionID, tokenHash string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, nil
	}
	if session.TokenHash != tokenHash {
		return nil, nil
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, nil
	}
	return session, nil
}

func (m *MockSessionStore) Rotate(ctx context.Context, sessionID, newTokenHash string, newExpiry time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil
	}
	session.TokenHash = newTokenHash
	session.ExpiresAt = newExpiry
	return nil
}

func (m *MockSessionStore) Revoke(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if exists {
		// Remove from user sessions
		userSessions := m.userSessions[session.UserID]
		for i, id := range userSessions {
			if id == sessionID {
				m.userSessions[session.UserID] = append(userSessions[:i], userSessions[i+1:]...)
				break
			}
		}
	}
	delete(m.sessions, sessionID)
	return nil
}

func (m *MockSessionStore) RevokeAllForUser(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessionIDs := m.userSessions[userID]
	for _, id := range sessionIDs {
		delete(m.sessions, id)
	}
	delete(m.userSessions, userID)
	return nil
}

func (m *MockSessionStore) ListForUser(ctx context.Context, userID string) ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessionIDs := m.userSessions[userID]
	sessions := make([]*Session, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		if session, exists := m.sessions[id]; exists {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}
