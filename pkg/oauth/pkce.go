package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// PKCE contains the code verifier and challenge for PKCE flow.
// Used for mobile apps where client secrets cannot be securely stored.
type PKCE struct {
	CodeVerifier  string // Random string stored on client
	CodeChallenge string // SHA256 hash of verifier, sent to auth server
	Method        string // Always "S256"
}

// GeneratePKCE creates a new PKCE code verifier and challenge.
// The verifier should be stored client-side and sent during token exchange.
func GeneratePKCE() (*PKCE, error) {
	// Generate 32 random bytes for verifier (results in 43 base64url chars)
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, err
	}
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Generate challenge as SHA256 hash of verifier
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &PKCE{
		CodeVerifier:  verifier,
		CodeChallenge: challenge,
		Method:        "S256",
	}, nil
}

// VerifyPKCE validates that a code verifier matches its challenge.
func VerifyPKCE(verifier, challenge string) bool {
	hash := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(hash[:])
	return expected == challenge
}
