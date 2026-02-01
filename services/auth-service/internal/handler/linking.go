package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Rohianon/equishare-global-trading/pkg/crypto"
	apperrors "github.com/Rohianon/equishare-global-trading/pkg/errors"
	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/pkg/middleware"
	"github.com/Rohianon/equishare-global-trading/pkg/oauth"
	"github.com/Rohianon/equishare-global-trading/services/auth-service/internal/types"
)

// LinkPhoneRequest initiates phone linking for social users.
type LinkPhoneRequest struct {
	Phone string `json:"phone" validate:"required"`
}

// LinkPhoneVerifyRequest verifies OTP and sets PIN.
type LinkPhoneVerifyRequest struct {
	Phone string `json:"phone" validate:"required"`
	OTP   string `json:"otp" validate:"required,len=6"`
	PIN   string `json:"pin" validate:"required,len=4"`
}

// LinkPhone sends OTP to link phone to existing account.
// POST /api/v1/auth/link/phone
func (h *OAuthHandler) LinkPhone(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return apperrors.ErrUnauthorized
	}

	var req LinkPhoneRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.ErrValidation.WithDetails("Invalid request body")
	}

	req.Phone = strings.TrimSpace(req.Phone)
	if !isValidKenyanPhone(req.Phone) {
		return apperrors.ErrInvalidPhone.WithDetails("Phone must be in format +254XXXXXXXXX")
	}

	ctx := c.Context()

	// Check user doesn't already have a verified phone
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return apperrors.ErrNotFound
	}
	if user.Phone != nil && user.PhoneVerified {
		return apperrors.ErrConflict.WithDetails("Phone already linked")
	}

	// Check phone not already taken
	existingUser, _ := h.userRepo.GetByPhone(ctx, req.Phone)
	if existingUser != nil && existingUser.ID != userID {
		return apperrors.ErrConflict.WithDetails("Phone number already registered to another account")
	}

	// Rate limit
	count, err := h.cache.Incr(ctx, rateLimitKey(req.Phone))
	if err != nil {
		logger.Error().Err(err).Msg("Failed to check rate limit")
	}
	if count == 1 {
		h.cache.Expire(ctx, rateLimitKey(req.Phone), time.Hour)
	}
	if count > 3 {
		return apperrors.ErrRateLimited.WithDetails("Too many OTP requests. Try again later.")
	}

	// Generate and send OTP
	otp, err := crypto.GenerateOTP(6)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate OTP")
		return apperrors.ErrInternal
	}

	// Store OTP with user context
	otpData := fmt.Sprintf("%s:%s", userID, otp)
	if err := h.cache.Set(ctx, linkPhoneOTPKey(req.Phone), otpData, 5*time.Minute); err != nil {
		logger.Error().Err(err).Msg("Failed to store OTP")
		return apperrors.ErrInternal
	}

	message := fmt.Sprintf("Your EquiShare verification code is: %s. Valid for 5 minutes.", otp)
	if err := h.sms.Send(req.Phone, message); err != nil {
		logger.Error().Err(err).Msg("Failed to send SMS")
		return apperrors.ErrServiceUnavailable.WithDetails("Failed to send verification code")
	}

	return c.JSON(fiber.Map{
		"message":    fmt.Sprintf("OTP sent to %s", maskPhone(req.Phone)),
		"expires_in": 300,
	})
}

// LinkPhoneVerify verifies OTP and links phone to account.
// POST /api/v1/auth/link/phone/verify
func (h *OAuthHandler) LinkPhoneVerify(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return apperrors.ErrUnauthorized
	}

	var req LinkPhoneVerifyRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.ErrValidation.WithDetails("Invalid request body")
	}

	req.Phone = strings.TrimSpace(req.Phone)
	if !isValidKenyanPhone(req.Phone) {
		return apperrors.ErrInvalidPhone
	}
	if len(req.OTP) != 6 {
		return apperrors.ErrValidation.WithDetails("OTP must be 6 digits")
	}
	if len(req.PIN) != 4 {
		return apperrors.ErrValidation.WithDetails("PIN must be 4 digits")
	}

	ctx := c.Context()

	// Get stored OTP
	storedData, err := h.cache.Get(ctx, linkPhoneOTPKey(req.Phone))
	if err != nil || storedData == "" {
		return apperrors.ErrValidation.WithDetails("OTP expired or not found")
	}

	// Parse stored data (userID:otp)
	parts := strings.SplitN(storedData, ":", 2)
	if len(parts) != 2 || parts[0] != userID {
		return apperrors.ErrValidation.WithDetails("Invalid OTP")
	}
	if parts[1] != req.OTP {
		return apperrors.ErrValidation.WithDetails("Invalid OTP")
	}

	// Delete OTP
	h.cache.Delete(ctx, linkPhoneOTPKey(req.Phone))

	// Hash PIN
	pinHash, err := crypto.HashPIN(req.PIN)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to hash PIN")
		return apperrors.ErrInternal
	}

	// Link phone to user
	if err := h.userRepo.LinkPhone(ctx, userID, req.Phone, pinHash); err != nil {
		logger.Error().Err(err).Msg("Failed to link phone")
		return apperrors.ErrInternal
	}

	// Create wallet if not exists
	h.walletRepo.Create(ctx, userID, "KES")

	logger.Info().Str("user_id", userID).Str("phone", maskPhone(req.Phone)).Msg("Phone linked successfully")

	return c.JSON(fiber.Map{
		"message": "Phone linked successfully. M-Pesa deposits now available.",
		"phone":   req.Phone,
	})
}

// LinkOAuth initiates OAuth linking for existing users.
// POST /api/v1/auth/link/:provider
func (h *OAuthHandler) LinkOAuth(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return apperrors.ErrUnauthorized
	}

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

	// Check user doesn't already have this provider linked
	identities, err := h.userRepo.GetOAuthIdentitiesByUser(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get OAuth identities")
		return apperrors.ErrInternal
	}
	for _, identity := range identities {
		if identity.Provider == provider {
			return apperrors.ErrConflict.WithDetails("Provider already linked")
		}
	}

	// Generate state with user ID for linking
	metadata := oauth.StateMetadata{
		Provider:     provider,
		RedirectURI:  req.RedirectURI,
		CodeVerifier: req.CodeVerifier,
		UserID:       userID, // Important: links to existing user
	}

	state, err := h.stateStore.Generate(ctx, metadata)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate OAuth state")
		return apperrors.ErrInternal
	}

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

// LinkOAuthCallback handles OAuth callback for account linking.
// POST /api/v1/auth/link/:provider/callback
func (h *OAuthHandler) LinkOAuthCallback(c *fiber.Ctx) error {
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

	ctx := c.Context()

	// Validate state
	metadata, err := h.stateStore.Validate(ctx, req.State)
	if err != nil {
		return apperrors.ErrValidation.WithDetails("Invalid or expired state")
	}

	if metadata.Provider != provider {
		return apperrors.ErrValidation.WithDetails("State provider mismatch")
	}

	if metadata.UserID == "" {
		return apperrors.ErrValidation.WithDetails("Invalid linking state")
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

	// Check if this OAuth identity already exists for another user
	existingIdentity, _ := h.userRepo.GetOAuthIdentity(ctx, provider, userInfo.ProviderID)
	if existingIdentity != nil {
		if existingIdentity.UserID != metadata.UserID {
			return apperrors.ErrConflict.WithDetails("This account is already linked to another user")
		}
		// Already linked to this user
		return c.JSON(fiber.Map{
			"message":  "Provider already linked",
			"provider": provider,
		})
	}

	// Link OAuth identity to user
	var email, name *string
	if userInfo.Email != "" {
		email = &userInfo.Email
	}
	if userInfo.Name != "" {
		name = &userInfo.Name
	}

	err = h.userRepo.CreateOAuthIdentity(ctx, metadata.UserID, provider, userInfo.ProviderID, email, name)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create OAuth identity")
		return apperrors.ErrInternal
	}

	logger.Info().Str("user_id", metadata.UserID).Str("provider", provider).Msg("OAuth provider linked")

	return c.JSON(fiber.Map{
		"message":  "Provider linked successfully",
		"provider": provider,
	})
}

// GetProviders returns linked auth providers for the current user.
// GET /api/v1/auth/providers
func (h *OAuthHandler) GetProviders(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return apperrors.ErrUnauthorized
	}

	ctx := c.Context()

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return apperrors.ErrNotFound
	}

	identities, err := h.userRepo.GetOAuthIdentitiesByUser(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get OAuth identities")
		return apperrors.ErrInternal
	}

	var providers []types.AuthProviderInfo

	// Add phone if verified
	if user.Phone != nil && user.PhoneVerified {
		providers = append(providers, types.AuthProviderInfo{
			Provider:  "phone",
			LinkedAt:  user.CreatedAt,
			CanUnlink: len(identities) > 0, // Can unlink if has other providers
		})
	}

	// Add OAuth providers
	for _, identity := range identities {
		email := ""
		if identity.ProviderEmail != nil {
			email = *identity.ProviderEmail
		}
		providers = append(providers, types.AuthProviderInfo{
			Provider:  identity.Provider,
			Email:     email,
			LinkedAt:  identity.CreatedAt,
			CanUnlink: len(providers) > 0 || len(identities) > 1, // Can unlink if has other providers
		})
	}

	return c.JSON(types.AuthProvidersResponse{
		Providers: providers,
		Primary:   user.PrimaryAuthProvider,
	})
}

// UnlinkProvider removes a linked auth provider.
// DELETE /api/v1/auth/unlink/:provider
func (h *OAuthHandler) UnlinkProvider(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return apperrors.ErrUnauthorized
	}

	provider := c.Params("provider")
	if provider == "" {
		return apperrors.ErrValidation.WithDetails("Provider is required")
	}

	ctx := c.Context()

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return apperrors.ErrNotFound
	}

	identities, err := h.userRepo.GetOAuthIdentitiesByUser(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get OAuth identities")
		return apperrors.ErrInternal
	}

	// Count total providers
	totalProviders := len(identities)
	if user.Phone != nil && user.PhoneVerified {
		totalProviders++
	}

	if totalProviders <= 1 {
		return apperrors.ErrValidation.WithDetails("Cannot unlink the only authentication method")
	}

	if provider == "phone" {
		// Cannot unlink phone (needed for M-Pesa)
		return apperrors.ErrValidation.WithDetails("Phone cannot be unlinked as it's required for M-Pesa")
	}

	// Find and delete the OAuth identity
	found := false
	for _, identity := range identities {
		if identity.Provider == provider {
			found = true
			break
		}
	}

	if !found {
		return apperrors.ErrNotFound.WithDetails("Provider not linked")
	}

	if err := h.userRepo.DeleteOAuthIdentity(ctx, userID, provider); err != nil {
		logger.Error().Err(err).Msg("Failed to delete OAuth identity")
		return apperrors.ErrInternal
	}

	logger.Info().Str("user_id", userID).Str("provider", provider).Msg("OAuth provider unlinked")

	return c.JSON(fiber.Map{
		"message":  "Provider unlinked successfully",
		"provider": provider,
	})
}

// SetUsername sets the username for the current user.
// POST /api/v1/auth/username
func (h *OAuthHandler) SetUsername(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return apperrors.ErrUnauthorized
	}

	var req types.SetUsernameRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.ErrValidation.WithDetails("Invalid request body")
	}

	req.Username = strings.TrimSpace(strings.ToLower(req.Username))
	if !isValidUsername(req.Username) {
		return apperrors.ErrValidation.WithDetails("Username must be 3-30 characters, start with a letter, and contain only letters, numbers, and underscores")
	}

	ctx := c.Context()

	// Check username not taken
	exists, err := h.userRepo.UsernameExists(ctx, req.Username)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to check username")
		return apperrors.ErrInternal
	}
	if exists {
		return apperrors.ErrConflict.WithDetails("Username already taken")
	}

	// Update username
	if err := h.userRepo.UpdateUsername(ctx, userID, req.Username); err != nil {
		logger.Error().Err(err).Msg("Failed to update username")
		return apperrors.ErrInternal
	}

	return c.JSON(fiber.Map{
		"message":  "Username set successfully",
		"username": req.Username,
	})
}

func linkPhoneOTPKey(phone string) string {
	return fmt.Sprintf("link_phone_otp:%s", phone)
}
