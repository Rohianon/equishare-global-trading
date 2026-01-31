package response

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/Rohianon/equishare-global-trading/pkg/errors"
)

// ErrorHandler is a Fiber error handler that converts errors to standard response format
func ErrorHandler(c *fiber.Ctx, err error) error {
	// Check for AppError
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		details := make([]string, 0)
		if appErr.Details != nil {
			switch d := appErr.Details.(type) {
			case string:
				details = append(details, d)
			case []string:
				details = d
			}
		}

		return c.Status(appErr.HTTPStatus).JSON(Response{
			Error: &ErrorBody{
				Code:    appErr.Code,
				Message: appErr.Message,
				Details: details,
			},
			Meta: buildMeta(c),
		})
	}

	// Check for Fiber error
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		code := httpStatusToErrorCode(fiberErr.Code)
		return c.Status(fiberErr.Code).JSON(Response{
			Error: &ErrorBody{
				Code:    code,
				Message: fiberErr.Message,
			},
			Meta: buildMeta(c),
		})
	}

	// Default to internal server error
	return c.Status(fiber.StatusInternalServerError).JSON(Response{
		Error: &ErrorBody{
			Code:    "INTERNAL_ERROR",
			Message: "An unexpected error occurred",
		},
		Meta: buildMeta(c),
	})
}

func httpStatusToErrorCode(status int) string {
	switch status {
	case fiber.StatusBadRequest:
		return "BAD_REQUEST"
	case fiber.StatusUnauthorized:
		return "UNAUTHORIZED"
	case fiber.StatusForbidden:
		return "FORBIDDEN"
	case fiber.StatusNotFound:
		return "NOT_FOUND"
	case fiber.StatusMethodNotAllowed:
		return "METHOD_NOT_ALLOWED"
	case fiber.StatusConflict:
		return "CONFLICT"
	case fiber.StatusTooManyRequests:
		return "RATE_LIMITED"
	case fiber.StatusInternalServerError:
		return "INTERNAL_ERROR"
	case fiber.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	default:
		return "UNKNOWN_ERROR"
	}
}
