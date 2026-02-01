package types

import "time"

type RegisterRequest struct {
	Phone string `json:"phone" validate:"required"`
}

type RegisterResponse struct {
	Message   string `json:"message"`
	ExpiresIn int    `json:"expires_in"`
}

type VerifyRequest struct {
	Phone    string `json:"phone" validate:"required"`
	OTP      string `json:"otp" validate:"required,len=6"`
	PIN      string `json:"pin" validate:"required,len=4"`
	Password string `json:"password,omitempty"`
}

type VerifyResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int          `json:"expires_in"`
}

type UserResponse struct {
	ID            string  `json:"id"`
	Phone         *string `json:"phone,omitempty"`
	Email         *string `json:"email,omitempty"`
	Username      *string `json:"username,omitempty"`
	DisplayName   *string `json:"display_name,omitempty"`
	AvatarURL     *string `json:"avatar_url,omitempty"`
	KYCStatus     string  `json:"kyc_status"`
	KYCTier       string  `json:"kyc_tier"`
	PhoneVerified bool    `json:"phone_verified"`
	EmailVerified bool    `json:"email_verified"`
}

type User struct {
	ID                  string
	Phone               *string // Nullable for social-first users
	Email               *string
	Username            *string
	DisplayName         *string
	AvatarURL           *string
	PasswordHash        *string
	PINHash             *string
	FirstName           *string
	LastName            *string
	KYCStatus           string
	KYCTier             string
	PrimaryAuthProvider string
	PhoneVerified       bool
	EmailVerified       bool
	AlpacaAccountID     *string
	IsActive            bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type Wallet struct {
	ID            string
	UserID        string
	Currency      string
	Balance       float64
	LockedBalance float64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type LoginRequest struct {
	Phone    string `json:"phone" validate:"required"`
	PIN      string `json:"pin,omitempty"`
	Password string `json:"password,omitempty"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// =============================================================================
// OAuth Types
// =============================================================================

// OAuthInitRequest initiates OAuth flow.
type OAuthInitRequest struct {
	RedirectURI  string `json:"redirect_uri" validate:"required,url"`
	CodeVerifier string `json:"code_verifier,omitempty"` // PKCE for mobile
}

// OAuthInitResponse returns the authorization URL.
type OAuthInitResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	State            string `json:"state"`
}

// OAuthCallbackRequest handles OAuth callback.
type OAuthCallbackRequest struct {
	Code         string          `json:"code" validate:"required"`
	State        string          `json:"state" validate:"required"`
	CodeVerifier string          `json:"code_verifier,omitempty"` // PKCE
	User         *AppleUserInfo  `json:"user,omitempty"`          // Apple first-auth only
}

// AppleUserInfo contains user info from Apple's first authorization.
type AppleUserInfo struct {
	Name  *AppleNameInfo `json:"name,omitempty"`
	Email string         `json:"email,omitempty"`
}

// AppleNameInfo contains name from Apple.
type AppleNameInfo struct {
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
}

// OAuthCallbackResponse returns after successful OAuth.
type OAuthCallbackResponse struct {
	User          UserResponse `json:"user"`
	AccessToken   string       `json:"access_token"`
	RefreshToken  string       `json:"refresh_token"`
	ExpiresIn     int          `json:"expires_in"`
	IsNewUser     bool         `json:"is_new_user"`
	NeedsPhone    bool         `json:"needs_phone"`    // True if M-Pesa not available
	NeedsUsername bool         `json:"needs_username"` // True if username not set
}

// MagicLinkRequest sends a magic link email.
type MagicLinkRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// MagicLinkResponse confirms magic link sent.
type MagicLinkResponse struct {
	Message   string `json:"message"`
	ExpiresIn int    `json:"expires_in"`
}

// MagicLinkVerifyRequest verifies a magic link token.
type MagicLinkVerifyRequest struct {
	Token string `json:"token" validate:"required"`
}

// UsernameCheckRequest checks username availability.
type UsernameCheckRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30"`
}

// UsernameCheckResponse returns availability status.
type UsernameCheckResponse struct {
	Available   bool     `json:"available"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// SetUsernameRequest sets the user's username.
type SetUsernameRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30"`
}

// AuthProvidersResponse lists linked auth providers.
type AuthProvidersResponse struct {
	Providers []AuthProviderInfo `json:"providers"`
	Primary   string             `json:"primary"`
}

// AuthProviderInfo describes a linked provider.
type AuthProviderInfo struct {
	Provider  string    `json:"provider"`
	Email     string    `json:"email,omitempty"`
	LinkedAt  time.Time `json:"linked_at"`
	CanUnlink bool      `json:"can_unlink"`
}

// OAuthIdentity represents a linked OAuth identity.
type OAuthIdentity struct {
	ID             string
	UserID         string
	Provider       string
	ProviderUserID string
	ProviderEmail  *string
	ProviderName   *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateOAuthUserParams contains parameters for creating an OAuth user.
type CreateOAuthUserParams struct {
	Email          string
	EmailVerified  bool
	DisplayName    string
	AvatarURL      string
	Provider       string
	ProviderUserID string
}
