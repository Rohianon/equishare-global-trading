package auth

import (
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
