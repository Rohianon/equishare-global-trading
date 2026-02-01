package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Rohianon/equishare-global-trading/pkg/config"
)

const (
	googleAuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL    = "https://oauth2.googleapis.com/token"
	googleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
)

// GoogleProvider implements AuthProvider for Google OAuth2.
type GoogleProvider struct {
	clientID     string
	clientSecret string
	redirectURIs []string
	scopes       []string
	httpClient   *http.Client
}

// NewGoogleProvider creates a new Google OAuth provider.
func NewGoogleProvider(cfg config.GoogleOAuthConfig) *GoogleProvider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}

	return &GoogleProvider{
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		redirectURIs: cfg.RedirectURIs,
		scopes:       scopes,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider identifier.
func (p *GoogleProvider) Name() string {
	return "google"
}

// GetAuthURL generates the Google authorization URL.
func (p *GoogleProvider) GetAuthURL(state string, opts ...AuthOption) (string, error) {
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
		"response_type": {"code"},
		"scope":         {strings.Join(p.scopes, " ")},
		"state":         {state},
		"access_type":   {"offline"}, // Get refresh token
		"prompt":        {"consent"}, // Force consent screen for refresh token
	}

	// Add PKCE if provided
	if options.CodeVerifier != "" {
		hash := sha256.Sum256([]byte(options.CodeVerifier))
		challenge := base64.RawURLEncoding.EncodeToString(hash[:])
		params.Set("code_challenge", challenge)
		params.Set("code_challenge_method", "S256")
	}

	return googleAuthURL + "?" + params.Encode(), nil
}

// ExchangeCode exchanges an authorization code for user information.
func (p *GoogleProvider) ExchangeCode(ctx context.Context, code string, opts ...AuthOption) (*UserInfo, error) {
	options := applyOptions(opts...)

	redirectURI := options.RedirectURI
	if redirectURI == "" && len(p.redirectURIs) > 0 {
		redirectURI = p.redirectURIs[0]
	}

	// Exchange code for tokens
	tokenResp, err := p.exchangeToken(ctx, code, redirectURI, options.CodeVerifier)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	// Get user info
	userInfo, err := p.getUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	return userInfo, nil
}

type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"`
}

func (p *GoogleProvider) exchangeToken(ctx context.Context, code, redirectURI, codeVerifier string) (*googleTokenResponse, error) {
	data := url.Values{
		"client_id":     {p.clientID},
		"client_secret": {p.clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURI},
	}

	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(data.Encode()))
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

	var tokenResp googleTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

type googleUserInfoResponse struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

func (p *GoogleProvider) getUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

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
		return nil, fmt.Errorf("user info request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var googleUser googleUserInfoResponse
	if err := json.Unmarshal(body, &googleUser); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	// Convert to common UserInfo format
	raw := make(map[string]any)
	_ = json.Unmarshal(body, &raw)

	return &UserInfo{
		ProviderID:    googleUser.Sub,
		Email:         googleUser.Email,
		EmailVerified: googleUser.EmailVerified,
		Name:          googleUser.Name,
		FirstName:     googleUser.GivenName,
		LastName:      googleUser.FamilyName,
		Picture:       googleUser.Picture,
		Raw:           raw,
	}, nil
}
