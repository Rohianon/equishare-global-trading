package handler

import (
	"regexp"

	"github.com/gofiber/fiber/v2"

	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/services/user-service/internal/repository"
	"github.com/Rohianon/equishare-global-trading/services/user-service/internal/types"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Handler handles user HTTP requests
type Handler struct {
	repo *repository.Repository
}

// NewHandler creates a new user handler
func NewHandler(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// GetProfile retrieves the current user's profile
// GET /users/me
func (h *Handler) GetProfile(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(types.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID required",
			Code:    401,
		})
	}

	ctx := c.Context()

	user, err := h.repo.GetUserByID(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user")
		return c.Status(fiber.StatusNotFound).JSON(types.ErrorResponse{
			Error:   "not_found",
			Message: "User not found",
			Code:    404,
		})
	}

	// Build full name
	fullName := ""
	if user.FirstName != nil {
		fullName = *user.FirstName
	}
	if user.LastName != nil {
		if fullName != "" {
			fullName += " "
		}
		fullName += *user.LastName
	}

	return c.JSON(types.UserProfile{
		ID:        user.ID,
		Phone:     user.Phone,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		FullName:  fullName,
		KYCStatus: user.KYCStatus,
		KYCTier:   user.KYCTier,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	})
}

// UpdateProfile updates the current user's profile
// PUT /users/me
func (h *Handler) UpdateProfile(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(types.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID required",
			Code:    401,
		})
	}

	var req types.UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
			Code:    400,
		})
	}

	ctx := c.Context()

	// Validate email if provided
	if req.Email != nil && *req.Email != "" {
		if !emailRegex.MatchString(*req.Email) {
			return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
				Error:   "bad_request",
				Message: "Invalid email format",
				Code:    400,
			})
		}

		// Check if email is already in use
		exists, err := h.repo.CheckEmailExists(ctx, *req.Email, userID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to check email")
			return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate email",
				Code:    500,
			})
		}
		if exists {
			return c.Status(fiber.StatusConflict).JSON(types.ErrorResponse{
				Error:   "conflict",
				Message: "Email already in use",
				Code:    409,
			})
		}
	}

	// Update profile
	if err := h.repo.UpdateProfile(ctx, userID, &req); err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("Failed to update profile")
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update profile",
			Code:    500,
		})
	}

	// Return updated profile
	return h.GetProfile(c)
}

// GetSettings retrieves the current user's settings
// GET /users/me/settings
func (h *Handler) GetSettings(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(types.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID required",
			Code:    401,
		})
	}

	ctx := c.Context()

	settings, err := h.repo.GetUserSettings(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get settings")
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get settings",
			Code:    500,
		})
	}

	return c.JSON(settings)
}

// UpdateSettings updates the current user's settings
// PUT /users/me/settings
func (h *Handler) UpdateSettings(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(types.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID required",
			Code:    401,
		})
	}

	var req types.UpdateSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
			Code:    400,
		})
	}

	// For now, just return the settings since we don't have a settings table
	// In a real implementation, you'd persist these
	return h.GetSettings(c)
}

// GetKYCStatus retrieves the current user's KYC status
// GET /users/me/kyc
func (h *Handler) GetKYCStatus(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(types.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID required",
			Code:    401,
		})
	}

	ctx := c.Context()

	user, err := h.repo.GetUserByID(ctx, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(types.ErrorResponse{
			Error:   "not_found",
			Message: "User not found",
			Code:    404,
		})
	}

	// Define limits based on KYC tier
	limits := getKYCLimits(user.KYCTier)

	return c.JSON(types.KYCStatusResponse{
		Status:      user.KYCStatus,
		Tier:        user.KYCTier,
		SubmittedAt: user.KYCSubmittedAt,
		VerifiedAt:  user.KYCVerifiedAt,
		Limits:      limits,
	})
}

// DeleteAccount deactivates the current user's account
// DELETE /users/me
func (h *Handler) DeleteAccount(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(types.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID required",
			Code:    401,
		})
	}

	ctx := c.Context()

	if err := h.repo.DeactivateUser(ctx, userID); err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("Failed to deactivate user")
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete account",
			Code:    500,
		})
	}

	return c.JSON(types.SuccessResponse{
		Success: true,
		Message: "Account deactivated successfully",
	})
}

// Helper function to get KYC limits based on tier
func getKYCLimits(tier string) types.KYCLimits {
	switch tier {
	case "tier1":
		return types.KYCLimits{
			DailyDeposit:    50000,   // KES 50,000
			DailyWithdrawal: 25000,   // KES 25,000
			DailyTrade:      100000,  // KES 100,000
		}
	case "tier2":
		return types.KYCLimits{
			DailyDeposit:    500000,  // KES 500,000
			DailyWithdrawal: 250000,  // KES 250,000
			DailyTrade:      1000000, // KES 1,000,000
		}
	case "tier3":
		return types.KYCLimits{
			DailyDeposit:    5000000,  // KES 5,000,000
			DailyWithdrawal: 2500000,  // KES 2,500,000
			DailyTrade:      10000000, // KES 10,000,000
		}
	default:
		return types.KYCLimits{
			DailyDeposit:    10000,
			DailyWithdrawal: 5000,
			DailyTrade:      20000,
		}
	}
}
