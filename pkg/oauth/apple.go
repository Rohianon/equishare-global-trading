package oauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Rohianon/equishare-global-trading/pkg/config"
)

const (
	appleAuthURL  = "https://appleid.apple.com/auth/authorize"
	appleTokenURL = "https://appleid.apple.com/auth/token"
	appleKeysURL  = "https://appleid.apple.com/auth/keys"
)

// AppleProvider implements AuthProvider for Apple Sign-In.
type AppleProvider struct {
	clientID     string // Service ID
	teamID       string
	keyID        string
	privateKey   *ecdsa.PrivateKey
	redirectURIs []string
	httpClient   *http.Client
}

// NewAppleProvider creates a new Apple Sign-In provider.
func NewAppleProvider(cfg config.AppleOAuthConfig) (*AppleProvider, error) {
	// Load private key
	privateKey, err := loadApplePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load Apple private key: %w", err)
	}

	return &AppleProvider{
		clientID:     cfg.ClientID,
		teamID:       cfg.TeamID,
		keyID:        cfg.KeyID,
		privateKey:   privateKey,
		redirectURIs: cfg.RedirectURIs,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// loadApplePrivateKey loads an ECDSA private key from file path or PEM content.
func loadApplePrivateKey(keyPathOrContent string) (*ecdsa.PrivateKey, error) {
	var pemData []byte
	var err error

	// Check if it's a file path or direct PEM content
	if strings.HasPrefix(keyPathOrContent, "-----BEGIN") {
		pemData = []byte(keyPathOrContent)
	} else {
		pemData, err = os.ReadFile(keyPathOrContent)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file: %w", err)
		}
	}

	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	ecdsaKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not an ECDSA private key")
	}

	return ecdsaKey, nil
}

// Name returns the provider identifier.
func (p *AppleProvider) Name() string {
	return "apple"
}

// GetAuthURL generates the Apple authorization URL.
func (p *AppleProvider) GetAuthURL(state string, opts ...AuthOption) (string, error) {
	options := applyOptions(opts...)

	redirectURI := options.RedirectURI
	if redirectURI == "" && len(p.redirectURIs) > 0 {
		redirectURI = p.redirectURIs[0]
	}
	if redirectURI == "" {
		return "", fmt.Errorf("redirect_uri is required")
	}

	params := url.Values{
		"client_id":     {p.clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code id_token"},
		"response_mode": {"form_post"},
		"scope":         {"name email"},
		"state":         {state},
	}

	// Add nonce if provided (for ID token validation)
	if options.Nonce != "" {
		params.Set("nonce", options.Nonce)
	}

	return appleAuthURL + "?" + params.Encode(), nil
}

// ExchangeCode exchanges an authorization code for user information.
// Note: Apple only sends user info (name, email) on the FIRST authorization.
// Subsequent logins only return the ID token with the user's sub (unique ID).
func (p *AppleProvider) ExchangeCode(ctx context.Context, code string, opts ...AuthOption) (*UserInfo, error) {
	options := applyOptions(opts...)

	redirectURI := options.RedirectURI
	if redirectURI == "" && len(p.redirectURIs) > 0 {
		redirectURI = p.redirectURIs[0]
	}

	// Generate client secret (JWT signed with private key)
	clientSecret, err := p.generateClientSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate client secret: %w", err)
	}

	// Exchange code for tokens
	tokenResp, err := p.exchangeToken(ctx, code, redirectURI, clientSecret)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	// Parse ID token to get user info
	userInfo, err := p.parseIDToken(tokenResp.IDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID token: %w", err)
	}

	return userInfo, nil
}

// generateClientSecret creates a JWT client secret for Apple.
// Apple requires a signed JWT instead of a static client secret.
func (p *AppleProvider) generateClientSecret() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": p.teamID,
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"aud": "https://appleid.apple.com",
		"sub": p.clientID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = p.keyID

	return token.SignedString(p.privateKey)
}

type appleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
}

func (p *AppleProvider) exchangeToken(ctx context.Context, code, redirectURI, clientSecret string) (*appleTokenResponse, error) {
	data := url.Values{
		"client_id":     {p.clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, appleTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp appleTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// appleIDTokenClaims represents the claims in Apple's ID token.
type appleIDTokenClaims struct {
	jwt.RegisteredClaims
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"` // Can be bool or string "true"/"false"
	IsPrivateEmail any   `json:"is_private_email"`
	AuthTime      int64  `json:"auth_time"`
	NonceSupported bool  `json:"nonce_supported"`
}

func (p *AppleProvider) parseIDToken(idToken string) (*UserInfo, error) {
	// Parse without verification for now (in production, verify with Apple's public keys)
	// For full security, fetch keys from appleKeysURL and verify signature
	token, _, err := new(jwt.Parser).ParseUnverified(idToken, &appleIDTokenClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID token: %w", err)
	}

	claims, ok := token.Claims.(*appleIDTokenClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Handle email_verified which can be bool or string
	emailVerified := false
	switch v := claims.EmailVerified.(type) {
	case bool:
		emailVerified = v
	case string:
		emailVerified = v == "true"
	}

	return &UserInfo{
		ProviderID:    claims.Subject,
		Email:         claims.Email,
		EmailVerified: emailVerified,
		Raw: map[string]any{
			"sub":              claims.Subject,
			"email":            claims.Email,
			"email_verified":   emailVerified,
			"is_private_email": claims.IsPrivateEmail,
		},
	}, nil
}

// AppleUserInfo represents the user info sent by Apple on first authorization.
// This is sent as a form post parameter, not in the token.
type AppleUserInfo struct {
	Name  *AppleNameInfo `json:"name,omitempty"`
	Email string         `json:"email,omitempty"`
}

// AppleNameInfo contains the user's name from Apple.
type AppleNameInfo struct {
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
}

// MergeAppleUserInfo merges Apple's first-auth user info into UserInfo.
// Call this when you receive user info from Apple's form post callback.
func MergeAppleUserInfo(userInfo *UserInfo, appleUser *AppleUserInfo) {
	if appleUser == nil {
		return
	}
	if appleUser.Email != "" && userInfo.Email == "" {
		userInfo.Email = appleUser.Email
	}
	if appleUser.Name != nil {
		userInfo.FirstName = appleUser.Name.FirstName
		userInfo.LastName = appleUser.Name.LastName
		if userInfo.Name == "" {
			userInfo.Name = strings.TrimSpace(appleUser.Name.FirstName + " " + appleUser.Name.LastName)
		}
	}
}
