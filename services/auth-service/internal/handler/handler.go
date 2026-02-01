package handler

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Rohianon/equishare-global-trading/pkg/auth"
	"github.com/Rohianon/equishare-global-trading/pkg/cache"
	"github.com/Rohianon/equishare-global-trading/pkg/crypto"
	apperrors "github.com/Rohianon/equishare-global-trading/pkg/errors"
	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/services/auth-service/internal/repository"
	"github.com/Rohianon/equishare-global-trading/services/auth-service/internal/types"
)

type SMSClient interface {
	Send(to, message string) error
}

type Handler struct {
	userRepo   *repository.UserRepository
	walletRepo *repository.WalletRepository
	cache      *cache.RedisCache
	sms        SMSClient
	jwt        *auth.JWTManager
}

func New(
	userRepo *repository.UserRepository,
	walletRepo *repository.WalletRepository,
	cache *cache.RedisCache,
	sms SMSClient,
	jwt *auth.JWTManager,
) *Handler {
	return &Handler{
		userRepo:   userRepo,
		walletRepo: walletRepo,
		cache:      cache,
		sms:        sms,
		jwt:        jwt,
	}
}

var kenyanPhoneRegex = regexp.MustCompile(`^\+254[17]\d{8}$`)

func isValidKenyanPhone(phone string) bool {
	return kenyanPhoneRegex.MatchString(phone)
}

func maskPhone(phone string) string {
	if len(phone) < 8 {
		return phone
	}
	return phone[:7] + "XXXX" + phone[len(phone)-2:]
}

func otpKey(phone string) string {
	return fmt.Sprintf("otp:%s", phone)
}

func rateLimitKey(phone string) string {
	return fmt.Sprintf("ratelimit:otp:%s", phone)
}

func (h *Handler) Register(c *fiber.Ctx) error {
	var req types.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.ErrValidation.WithDetails("Invalid request body")
	}

	req.Phone = strings.TrimSpace(req.Phone)
	if !isValidKenyanPhone(req.Phone) {
		return apperrors.ErrInvalidPhone.WithDetails("Phone must be in format +254XXXXXXXXX")
	}

	ctx := c.Context()

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

	exists, err := h.userRepo.ExistsByPhone(ctx, req.Phone)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to check user existence")
		return apperrors.ErrInternal
	}
	if exists {
		return apperrors.ErrConflict.WithDetails("Phone number already registered")
	}

	otp, err := crypto.GenerateOTP(6)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate OTP")
		return apperrors.ErrInternal
	}

	if err := h.cache.Set(ctx, otpKey(req.Phone), otp, 5*time.Minute); err != nil {
		logger.Error().Err(err).Msg("Failed to store OTP")
		return apperrors.ErrInternal
	}

	message := fmt.Sprintf("Your EquiShare verification code is: %s. Valid for 5 minutes.", otp)
	if err := h.sms.Send(req.Phone, message); err != nil {
		logger.Error().Err(err).Msg("Failed to send SMS")
		return apperrors.ErrServiceUnavailable.WithDetails("Failed to send verification code")
	}

	logger.Info().Str("phone", maskPhone(req.Phone)).Msg("OTP sent for registration")

	return c.Status(fiber.StatusOK).JSON(types.RegisterResponse{
		Message:   fmt.Sprintf("OTP sent to %s", maskPhone(req.Phone)),
		ExpiresIn: 300,
	})
}

func (h *Handler) Verify(c *fiber.Ctx) error {
	var req types.VerifyRequest
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

	storedOTP, err := h.cache.Get(ctx, otpKey(req.Phone))
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get OTP from cache")
		return apperrors.ErrInternal
	}
	if storedOTP == "" {
		return apperrors.ErrValidation.WithDetails("OTP expired or not found")
	}
	if storedOTP != req.OTP {
		return apperrors.ErrValidation.WithDetails("Invalid OTP")
	}

	h.cache.Delete(ctx, otpKey(req.Phone))

	pinHash, err := crypto.HashPIN(req.PIN)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to hash PIN")
		return apperrors.ErrInternal
	}

	var passwordHash *string
	if req.Password != "" {
		hash, err := crypto.HashPassword(req.Password)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to hash password")
			return apperrors.ErrInternal
		}
		passwordHash = &hash
	}

	user, err := h.userRepo.Create(ctx, req.Phone, pinHash, passwordHash)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create user")
		return apperrors.ErrInternal
	}

	_, err = h.walletRepo.Create(ctx, user.ID, "KES")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create wallet")
	}

	phone := ""
	if user.Phone != nil {
		phone = *user.Phone
	}
	tokens, err := h.jwt.GenerateTokenPair(user.ID, phone)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate tokens")
		return apperrors.ErrInternal
	}

	logger.Info().Str("user_id", user.ID).Msg("User registered successfully")

	return c.Status(fiber.StatusCreated).JSON(types.VerifyResponse{
		User: types.UserResponse{
			ID:            user.ID,
			Phone:         user.Phone,
			KYCStatus:     user.KYCStatus,
			KYCTier:       user.KYCTier,
			PhoneVerified: user.PhoneVerified,
			EmailVerified: user.EmailVerified,
		},
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req types.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.ErrValidation.WithDetails("Invalid request body")
	}

	req.Phone = strings.TrimSpace(req.Phone)
	if !isValidKenyanPhone(req.Phone) {
		return apperrors.ErrInvalidPhone
	}

	if req.PIN == "" && req.Password == "" {
		return apperrors.ErrValidation.WithDetails("PIN or password required")
	}

	ctx := c.Context()

	user, err := h.userRepo.GetByPhone(ctx, req.Phone)
	if err != nil {
		return apperrors.ErrInvalidCredentials
	}

	if !user.IsActive {
		return apperrors.ErrForbidden.WithDetails("Account is deactivated")
	}

	if req.PIN != "" && user.PINHash != nil {
		if !crypto.CheckPIN(req.PIN, *user.PINHash) {
			return apperrors.ErrInvalidCredentials
		}
	} else if req.Password != "" && user.PasswordHash != nil {
		if !crypto.CheckPassword(req.Password, *user.PasswordHash) {
			return apperrors.ErrInvalidCredentials
		}
	} else {
		return apperrors.ErrInvalidCredentials
	}

	phone := ""
	if user.Phone != nil {
		phone = *user.Phone
	}
	tokens, err := h.jwt.GenerateTokenPair(user.ID, phone)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate tokens")
		return apperrors.ErrInternal
	}

	logger.Info().Str("user_id", user.ID).Msg("User logged in")

	return c.JSON(types.VerifyResponse{
		User: types.UserResponse{
			ID:            user.ID,
			Phone:         user.Phone,
			KYCStatus:     user.KYCStatus,
			KYCTier:       user.KYCTier,
			PhoneVerified: user.PhoneVerified,
			EmailVerified: user.EmailVerified,
		},
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

func (h *Handler) RefreshToken(c *fiber.Ctx) error {
	var req types.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.ErrValidation.WithDetails("Invalid request body")
	}

	claims, err := h.jwt.ValidateToken(req.RefreshToken)
	if err != nil {
		return apperrors.ErrUnauthorized.WithDetails("Invalid refresh token")
	}

	ctx := c.Context()
	user, err := h.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return apperrors.ErrUnauthorized
	}

	if !user.IsActive {
		return apperrors.ErrForbidden.WithDetails("Account is deactivated")
	}

	phone := ""
	if user.Phone != nil {
		phone = *user.Phone
	}
	tokens, err := h.jwt.GenerateTokenPair(user.ID, phone)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate tokens")
		return apperrors.ErrInternal
	}

	return c.JSON(types.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

var _ context.Context
