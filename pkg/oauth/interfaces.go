// Package oauth provides OAuth2 authentication providers for EquiShare.
// It follows SOLID principles with dependency injection support.
package oauth

import (
	"context"
)

// AuthProvider defines the interface for OAuth2 providers (Google, Apple, etc.)
// Each provider implements this interface, allowing easy addition of new providers
// without modifying existing code (Open/Closed Principle).
type AuthProvider interface {
	// Name returns the provider identifier (e.g., "google", "apple")
	Name() string

	// GetAuthURL generates the authorization URL for the OAuth flow.
	// The state parameter is used for CSRF protection.
	GetAuthURL(state string, opts ...AuthOption) (string, error)

	// ExchangeCode exchanges an authorization code for user information.
	// This is called after the user authorizes the app.
	ExchangeCode(ctx context.Context, code string, opts ...AuthOption) (*UserInfo, error)
}

// UserInfo contains the user information returned by OAuth providers.
// This is the common structure across all providers (Liskov Substitution Principle).
type UserInfo struct {
	ProviderID    string         // Unique ID from the provider (Google sub, Apple sub)
	Email         string         // User's email address
	EmailVerified bool           // Whether the email is verified by the provider
	Name          string         // Display name
	FirstName     string         // First name (if available)
	LastName      string         // Last name (if available)
	Picture       string         // Profile picture URL
	Raw           map[string]any // Raw response from provider for debugging
}

// MagicLinkProvider handles passwordless email authentication.
// Separated from AuthProvider as it has different concerns (Interface Segregation).
type MagicLinkProvider interface {
	// GenerateToken creates a new magic link token for the given email.
	// Returns the token to be included in the magic link URL.
	GenerateToken(ctx context.Context, email string, userID *string) (token string, err error)

	// VerifyToken validates a magic link token and returns associated info.
	// Returns an error if the token is invalid or expired.
	VerifyToken(ctx context.Context, token string) (*MagicLinkInfo, error)

	// MarkUsed marks a token as used (one-time use).
	MarkUsed(ctx context.Context, token string) error
}

// MagicLinkInfo contains information about a verified magic link.
type MagicLinkInfo struct {
	Email    string  // Email address the link was sent to
	UserID   *string // User ID if this is an existing user, nil for new registration
	IssuedAt int64   // Unix timestamp when token was created
}

// ProviderRegistry manages OAuth providers.
// Allows dynamic registration of providers (Open/Closed Principle).
type ProviderRegistry interface {
	// Register adds a new provider to the registry.
	Register(provider AuthProvider)

	// Get retrieves a provider by name.
	Get(name string) (AuthProvider, bool)

	// List returns all registered provider names.
	List() []string
}

// StateStore manages OAuth state parameters for CSRF protection.
type StateStore interface {
	// Generate creates a new state with associated metadata.
	Generate(ctx context.Context, metadata StateMetadata) (state string, err error)

	// Validate checks if a state is valid and returns its metadata.
	// The state is consumed (one-time use).
	Validate(ctx context.Context, state string) (*StateMetadata, error)
}

// StateMetadata contains data associated with an OAuth state parameter.
type StateMetadata struct {
	Provider     string // Provider name (google, apple)
	RedirectURI  string // Where to redirect after auth
	CodeVerifier string // PKCE code verifier (for mobile apps)
	Nonce        string // Nonce for ID token validation (Apple)
	UserID       string // User ID if linking to existing account
}

// AuthOption is a functional option for configuring auth requests.
type AuthOption func(*authOptions)

type authOptions struct {
	RedirectURI  string
	CodeVerifier string // PKCE verifier
	Nonce        string // For Apple
}

// WithRedirectURI sets the redirect URI for the auth request.
func WithRedirectURI(uri string) AuthOption {
	return func(o *authOptions) {
		o.RedirectURI = uri
	}
}

// WithPKCE sets the PKCE code verifier for mobile apps.
func WithPKCE(verifier string) AuthOption {
	return func(o *authOptions) {
		o.CodeVerifier = verifier
	}
}

// WithNonce sets the nonce for Apple Sign-In.
func WithNonce(nonce string) AuthOption {
	return func(o *authOptions) {
		o.Nonce = nonce
	}
}

// applyOptions applies functional options to authOptions.
func applyOptions(opts ...AuthOption) *authOptions {
	o := &authOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
