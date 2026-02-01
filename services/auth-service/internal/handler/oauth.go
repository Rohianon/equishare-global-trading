package handler

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/Rohianon/equishare-global-trading/pkg/errors"
	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/pkg/oauth"
	"github.com/Rohianon/equishare-global-trading/services/auth-service/internal/types"
)

// OAuthHandler handles OAuth authentication flows.
type OAuthHandler struct {
	*Handler
	providers  oauth.ProviderRegistry
	stateStore oauth.StateStore
	magicLink  oauth.MagicLinkProvider
}

// NewOAuthHandler creates a new OAuth handler with injected dependencies.
func NewOAuthHandler(
	base *Handler,
	providers oauth.ProviderRegistry,
	stateStore oauth.StateStore,
	magicLink oauth.MagicLinkProvider,
) *OAuthHandler {
	return &OAuthHandler{
		Handler:    base,
		providers:  providers,
		stateStore: stateStore,
		magicLink:  magicLink,
	}
}

// OAuthInit initiates OAuth flow for a provider.
// POST /api/v1/auth/oauth/:provider
func (h *OAuthHandler) OAuthInit(c *fiber.Ctx) error {
	provider := c.Params("provider")
	if provider == "" {
		return apperrors.ErrValidation.WithDetails("Provider is required")
	}

	authProvider, ok := h.providers.Get(provider)
	if !ok {
		return apperrors.ErrValidation.WithDetails("Unsupported provider: " + provider)
	}

	var req types.OAuthInitRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.ErrValidation.WithDetails("Invalid request body")
	}

	if req.RedirectURI == "" {
		return apperrors.ErrValidation.WithDetails("redirect_uri is required")
	}

	ctx := c.Context()

	// Generate state for CSRF protection
	metadata := oauth.StateMetadata{
		Provider:     provider,
		RedirectURI:  req.RedirectURI,
		CodeVerifier: req.CodeVerifier,
	}

	state, err := h.stateStore.Generate(ctx, metadata)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate OAuth state")
		return apperrors.ErrInternal
	}

	// Build authorization URL
	opts := []oauth.AuthOption{
		oauth.WithRedirectURI(req.RedirectURI),
	}
	if req.CodeVerifier != "" {
		opts = append(opts, oauth.WithPKCE(req.CodeVerifier))
	}

	authURL, err := authProvider.GetAuthURL(state, opts...)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate auth URL")
		return apperrors.ErrInternal
	}

	return c.JSON(types.OAuthInitResponse{
		AuthorizationURL: authURL,
		State:            state,
	})
}

// OAuthCallback handles OAuth callback.
// POST /api/v1/auth/oauth/:provider/callback
func (h *OAuthHandler) OAuthCallback(c *fiber.Ctx) error {
	provider := c.Params("provider")
	if provider == "" {
		return apperrors.ErrValidation.WithDetails("Provider is required")
	}

	authProvider, ok := h.providers.Get(provider)
	if !ok {
		return apperrors.ErrValidation.WithDetails("Unsupported provider: " + provider)
	}

	var req types.OAuthCallbackRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.ErrValidation.WithDetails("Invalid request body")
	}

	if req.Code == "" || req.State == "" {
		return apperrors.ErrValidation.WithDetails("code and state are required")
	}

	ctx := c.Context()

	// Validate state
	metadata, err := h.stateStore.Validate(ctx, req.State)
	if err != nil {
		return apperrors.ErrValidation.WithDetails("Invalid or expired state")
	}

	if metadata.Provider != provider {
		return apperrors.ErrValidation.WithDetails("State provider mismatch")
	}

	// Exchange code for user info
	opts := []oauth.AuthOption{
		oauth.WithRedirectURI(metadata.RedirectURI),
	}
	if metadata.CodeVerifier != "" {
		opts = append(opts, oauth.WithPKCE(metadata.CodeVerifier))
	}

	userInfo, err := authProvider.ExchangeCode(ctx, req.Code, opts...)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to exchange OAuth code")
		return apperrors.ErrServiceUnavailable.WithDetails("OAuth provider error")
	}

	// Merge Apple user info if provided (Apple only sends on first auth)
	if provider == "apple" && req.User != nil {
		oauth.MergeAppleUserInfo(userInfo, &oauth.AppleUserInfo{
			Email: req.User.Email,
			Name: &oauth.AppleNameInfo{
				FirstName: req.User.Name.FirstName,
				LastName:  req.User.Name.LastName,
			},
		})
	}

	// Check if this OAuth identity already exists
	existingIdentity, err := h.userRepo.GetOAuthIdentity(ctx, provider, userInfo.ProviderID)
	if err == nil && existingIdentity != nil {
		// Existing user - login
		user, err := h.userRepo.GetByID(ctx, existingIdentity.UserID)
		if err != nil {
			return apperrors.ErrInternal
		}

		return h.completeOAuthLogin(c, user, false)
	}

	// Check if email is already registered
	if userInfo.Email != "" {
		existingUser, _ := h.userRepo.GetByEmail(ctx, userInfo.Email)
		if existingUser != nil {
			return apperrors.New(
				"AUTH_ACCOUNT_EXISTS",
				"An account with this email already exists. Please sign in with your existing method and link this provider from settings.",
				409,
			)
		}
	}

	// Create new user
	user, err := h.userRepo.CreateOAuthUser(ctx, types.CreateOAuthUserParams{
		Email:          userInfo.Email,
		EmailVerified:  userInfo.EmailVerified,
		DisplayName:    userInfo.Name,
		AvatarURL:      userInfo.Picture,
		Provider:       provider,
		ProviderUserID: userInfo.ProviderID,
	})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create OAuth user")
		return apperrors.ErrInternal
	}

	// Create default wallet
	_, _ = h.walletRepo.Create(ctx, user.ID, "KES")

	return h.completeOAuthLogin(c, user, true)
}

func (h *OAuthHandler) completeOAuthLogin(c *fiber.Ctx, user *types.User, isNewUser bool) error {
	phone := ""
	if user.Phone != nil {
		phone = *user.Phone
	}

	tokens, err := h.jwt.GenerateTokenPair(user.ID, phone)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate tokens")
		return apperrors.ErrInternal
	}

	needsPhone := user.Phone == nil || !user.PhoneVerified
	needsUsername := user.Username == nil || *user.Username == ""

	return c.JSON(types.OAuthCallbackResponse{
		User: types.UserResponse{
			ID:            user.ID,
			Phone:         user.Phone,
			Email:         user.Email,
			Username:      user.Username,
			DisplayName:   user.DisplayName,
			AvatarURL:     user.AvatarURL,
			KYCStatus:     user.KYCStatus,
			KYCTier:       user.KYCTier,
			PhoneVerified: user.PhoneVerified,
			EmailVerified: user.EmailVerified,
		},
		AccessToken:   tokens.AccessToken,
		RefreshToken:  tokens.RefreshToken,
		ExpiresIn:     tokens.ExpiresIn,
		IsNewUser:     isNewUser,
		NeedsPhone:    needsPhone,
		NeedsUsername: needsUsername,
	})
}

// MagicLinkSend sends a magic link email.
// POST /api/v1/auth/magic-link
func (h *OAuthHandler) MagicLinkSend(c *fiber.Ctx) error {
	var req types.MagicLinkRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.ErrValidation.WithDetails("Invalid request body")
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !isValidEmail(req.Email) {
		return apperrors.ErrValidation.WithDetails("Invalid email address")
	}

	ctx := c.Context()

	// Check if email exists
	var userID *string
	existingUser, _ := h.userRepo.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		userID = &existingUser.ID
	}

	// Generate magic link token
	token, err := h.magicLink.GenerateToken(ctx, req.Email, userID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate magic link")
		return apperrors.ErrInternal
	}

	// TODO: Send email with magic link
	// For now, log the token (in production, send via email service)
	logger.Info().
		Str("email", req.Email).
		Str("token", token).
		Msg("Magic link generated")

	return c.JSON(types.MagicLinkResponse{
		Message:   fmt.Sprintf("Magic link sent to %s", maskEmail(req.Email)),
		ExpiresIn: 900, // 15 minutes
	})
}

// MagicLinkVerify verifies a magic link token.
// POST /api/v1/auth/magic-link/verify
func (h *OAuthHandler) MagicLinkVerify(c *fiber.Ctx) error {
	var req types.MagicLinkVerifyRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.ErrValidation.WithDetails("Invalid request body")
	}

	if req.Token == "" {
		return apperrors.ErrValidation.WithDetails("Token is required")
	}

	ctx := c.Context()

	// Verify token
	info, err := h.magicLink.VerifyToken(ctx, req.Token)
	if err != nil {
		return apperrors.ErrValidation.WithDetails("Invalid or expired token")
	}

	// Mark token as used
	if err := h.magicLink.MarkUsed(ctx, req.Token); err != nil {
		logger.Error().Err(err).Msg("Failed to mark magic link as used")
	}

	var user *types.User
	isNewUser := false

	if info.UserID != nil {
		// Existing user
		user, err = h.userRepo.GetByID(ctx, *info.UserID)
		if err != nil {
			return apperrors.ErrInternal
		}
	} else {
		// New user - create account
		user, err = h.userRepo.CreateEmailUser(ctx, info.Email)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to create email user")
			return apperrors.ErrInternal
		}
		isNewUser = true

		// Create default wallet
		_, _ = h.walletRepo.Create(ctx, user.ID, "KES")
	}

	return h.completeOAuthLogin(c, user, isNewUser)
}

// UsernameCheck checks if a username is available.
// GET /api/v1/auth/username/check?username=xxx
func (h *OAuthHandler) UsernameCheck(c *fiber.Ctx) error {
	username := strings.TrimSpace(c.Query("username"))
	if username == "" {
		return apperrors.ErrValidation.WithDetails("Username is required")
	}

	if !isValidUsername(username) {
		return c.JSON(types.UsernameCheckResponse{
			Available:   false,
			Suggestions: generateUsernameSuggestions(username),
		})
	}

	ctx := c.Context()
	exists, err := h.userRepo.UsernameExists(ctx, username)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to check username")
		return apperrors.ErrInternal
	}

	if exists {
		return c.JSON(types.UsernameCheckResponse{
			Available:   false,
			Suggestions: generateUsernameSuggestions(username),
		})
	}

	return c.JSON(types.UsernameCheckResponse{
		Available: true,
	})
}

// Helper functions

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
var usernameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{2,29}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func isValidUsername(username string) bool {
	return usernameRegex.MatchString(username)
}

func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	local := parts[0]
	if len(local) <= 2 {
		return email
	}
	return local[:2] + "***@" + parts[1]
}

func generateUsernameSuggestions(base string) []string {
	// Clean the base username
	clean := strings.ToLower(base)
	clean = regexp.MustCompile(`[^a-z0-9]`).ReplaceAllString(clean, "")
	if len(clean) < 3 {
		clean = "user"
	}
	if len(clean) > 20 {
		clean = clean[:20]
	}

	return []string{
		clean + "123",
		clean + "_trader",
		clean + "2024",
	}
}
