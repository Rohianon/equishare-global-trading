package errors

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Details    any    `json:"details,omitempty"`
	HTTPStatus int    `json:"-"`
	Err        error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) WithDetails(details any) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    e.Message,
		Details:    details,
		HTTPStatus: e.HTTPStatus,
		Err:        e.Err,
	}
}

func (e *AppError) WithError(err error) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    e.Message,
		Details:    e.Details,
		HTTPStatus: e.HTTPStatus,
		Err:        err,
	}
}

var (
	ErrUnauthorized = &AppError{
		Code:       "UNAUTHORIZED",
		Message:    "Authentication required",
		HTTPStatus: http.StatusUnauthorized,
	}

	ErrForbidden = &AppError{
		Code:       "FORBIDDEN",
		Message:    "Access denied",
		HTTPStatus: http.StatusForbidden,
	}

	ErrInvalidCredentials = &AppError{
		Code:       "INVALID_CREDENTIALS",
		Message:    "Invalid phone number or PIN",
		HTTPStatus: http.StatusUnauthorized,
	}

	ErrValidation = &AppError{
		Code:       "VALIDATION_ERROR",
		Message:    "Invalid input",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrInvalidPhone = &AppError{
		Code:       "INVALID_PHONE",
		Message:    "Invalid phone number format",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrNotFound = &AppError{
		Code:       "NOT_FOUND",
		Message:    "Resource not found",
		HTTPStatus: http.StatusNotFound,
	}

	ErrConflict = &AppError{
		Code:       "CONFLICT",
		Message:    "Resource already exists",
		HTTPStatus: http.StatusConflict,
	}

	ErrInsufficientFunds = &AppError{
		Code:       "INSUFFICIENT_FUNDS",
		Message:    "Insufficient balance for this transaction",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrKYCRequired = &AppError{
		Code:       "KYC_REQUIRED",
		Message:    "KYC verification required for this action",
		HTTPStatus: http.StatusForbidden,
	}

	ErrTradingHoursClosed = &AppError{
		Code:       "TRADING_HOURS_CLOSED",
		Message:    "Trading is currently closed",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrOrderLimitExceeded = &AppError{
		Code:       "ORDER_LIMIT_EXCEEDED",
		Message:    "Order exceeds your daily limit",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrRateLimited = &AppError{
		Code:       "RATE_LIMITED",
		Message:    "Too many requests, please try again later",
		HTTPStatus: http.StatusTooManyRequests,
	}

	ErrInternal = &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "An unexpected error occurred",
		HTTPStatus: http.StatusInternalServerError,
	}

	ErrServiceUnavailable = &AppError{
		Code:       "SERVICE_UNAVAILABLE",
		Message:    "Service temporarily unavailable",
		HTTPStatus: http.StatusServiceUnavailable,
	}
)
