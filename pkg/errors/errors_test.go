package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected string
	}{
		{
			name:     "without wrapped error",
			err:      ErrUnauthorized,
			expected: "UNAUTHORIZED: Authentication required",
		},
		{
			name:     "with wrapped error",
			err:      ErrInternal.WithError(errors.New("db connection failed")),
			expected: "INTERNAL_ERROR: An unexpected error occurred (db connection failed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("AppError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	appErr := ErrInternal.WithError(innerErr)

	if appErr.Unwrap() != innerErr {
		t.Errorf("AppError.Unwrap() did not return the wrapped error")
	}

	if ErrUnauthorized.Unwrap() != nil {
		t.Errorf("AppError.Unwrap() should return nil when no error is wrapped")
	}
}

func TestAppError_WithDetails(t *testing.T) {
	details := map[string]string{"field": "email", "reason": "invalid format"}
	appErr := ErrValidation.WithDetails(details)

	if appErr.Details == nil {
		t.Errorf("WithDetails should set Details")
	}

	if appErr.Code != ErrValidation.Code {
		t.Errorf("WithDetails should preserve Code")
	}

	if appErr.HTTPStatus != ErrValidation.HTTPStatus {
		t.Errorf("WithDetails should preserve HTTPStatus")
	}
}

func TestAppError_WithError(t *testing.T) {
	innerErr := errors.New("database error")
	appErr := ErrInternal.WithError(innerErr)

	if appErr.Err != innerErr {
		t.Errorf("WithError should set Err")
	}

	if appErr.Code != ErrInternal.Code {
		t.Errorf("WithError should preserve Code")
	}
}

func TestAppError_WithMessage(t *testing.T) {
	appErr := ErrNotFound.WithMessage("User not found")

	if appErr.Message != "User not found" {
		t.Errorf("WithMessage should set Message")
	}

	if appErr.Code != ErrNotFound.Code {
		t.Errorf("WithMessage should preserve Code")
	}

	if ErrNotFound.Message == "User not found" {
		t.Error("WithMessage should not modify original")
	}
}

func TestNew(t *testing.T) {
	err := New("CUSTOM_ERROR", "Custom message", http.StatusTeapot)

	if err.Code != "CUSTOM_ERROR" {
		t.Errorf("Code = %s, want CUSTOM_ERROR", err.Code)
	}
	if err.Message != "Custom message" {
		t.Errorf("Message = %s, want Custom message", err.Message)
	}
	if err.HTTPStatus != http.StatusTeapot {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusTeapot)
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        *AppError
		httpStatus int
	}{
		// Common errors
		{"ErrBadRequest", ErrBadRequest, http.StatusBadRequest},
		{"ErrUnauthorized", ErrUnauthorized, http.StatusUnauthorized},
		{"ErrForbidden", ErrForbidden, http.StatusForbidden},
		{"ErrNotFound", ErrNotFound, http.StatusNotFound},
		{"ErrConflict", ErrConflict, http.StatusConflict},
		{"ErrValidation", ErrValidation, http.StatusBadRequest},
		{"ErrRateLimited", ErrRateLimited, http.StatusTooManyRequests},
		{"ErrInternal", ErrInternal, http.StatusInternalServerError},
		{"ErrServiceUnavailable", ErrServiceUnavailable, http.StatusServiceUnavailable},

		// Auth errors
		{"ErrInvalidCredentials", ErrInvalidCredentials, http.StatusUnauthorized},
		{"ErrInvalidToken", ErrInvalidToken, http.StatusUnauthorized},
		{"ErrTokenExpired", ErrTokenExpired, http.StatusUnauthorized},
		{"ErrSessionExpired", ErrSessionExpired, http.StatusUnauthorized},
		{"ErrInvalidOTP", ErrInvalidOTP, http.StatusBadRequest},
		{"ErrOTPExpired", ErrOTPExpired, http.StatusBadRequest},
		{"Err2FARequired", Err2FARequired, http.StatusForbidden},
		{"ErrInvalid2FACode", ErrInvalid2FACode, http.StatusBadRequest},

		// User errors
		{"ErrInvalidPhone", ErrInvalidPhone, http.StatusBadRequest},
		{"ErrPhoneAlreadyExists", ErrPhoneAlreadyExists, http.StatusConflict},
		{"ErrEmailAlreadyExists", ErrEmailAlreadyExists, http.StatusConflict},
		{"ErrUserNotFound", ErrUserNotFound, http.StatusNotFound},
		{"ErrUserDeactivated", ErrUserDeactivated, http.StatusForbidden},
		{"ErrKYCRequired", ErrKYCRequired, http.StatusForbidden},
		{"ErrKYCPending", ErrKYCPending, http.StatusForbidden},
		{"ErrKYCRejected", ErrKYCRejected, http.StatusForbidden},

		// Payment errors
		{"ErrInsufficientFunds", ErrInsufficientFunds, http.StatusBadRequest},
		{"ErrWalletNotFound", ErrWalletNotFound, http.StatusNotFound},
		{"ErrPaymentFailed", ErrPaymentFailed, http.StatusBadRequest},
		{"ErrPaymentPending", ErrPaymentPending, http.StatusAccepted},
		{"ErrWithdrawalFailed", ErrWithdrawalFailed, http.StatusBadRequest},
		{"ErrMinimumAmount", ErrMinimumAmount, http.StatusBadRequest},
		{"ErrMaximumAmount", ErrMaximumAmount, http.StatusBadRequest},
		{"ErrDailyLimitExceeded", ErrDailyLimitExceeded, http.StatusBadRequest},

		// Trading errors
		{"ErrTradingHoursClosed", ErrTradingHoursClosed, http.StatusBadRequest},
		{"ErrOrderNotFound", ErrOrderNotFound, http.StatusNotFound},
		{"ErrOrderAlreadyFilled", ErrOrderAlreadyFilled, http.StatusBadRequest},
		{"ErrOrderAlreadyCancelled", ErrOrderAlreadyCancelled, http.StatusBadRequest},
		{"ErrOrderLimitExceeded", ErrOrderLimitExceeded, http.StatusBadRequest},
		{"ErrInvalidSymbol", ErrInvalidSymbol, http.StatusBadRequest},
		{"ErrInvalidOrderType", ErrInvalidOrderType, http.StatusBadRequest},
		{"ErrInvalidOrderSide", ErrInvalidOrderSide, http.StatusBadRequest},
		{"ErrInvalidQuantity", ErrInvalidQuantity, http.StatusBadRequest},
		{"ErrSymbolNotTradeable", ErrSymbolNotTradeable, http.StatusBadRequest},
		{"ErrPositionNotFound", ErrPositionNotFound, http.StatusNotFound},

		// Provider errors
		{"ErrMpesaUnavailable", ErrMpesaUnavailable, http.StatusServiceUnavailable},
		{"ErrAlpacaUnavailable", ErrAlpacaUnavailable, http.StatusServiceUnavailable},
		{"ErrSMSUnavailable", ErrSMSUnavailable, http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.HTTPStatus != tt.httpStatus {
				t.Errorf("%s.HTTPStatus = %v, want %v", tt.name, tt.err.HTTPStatus, tt.httpStatus)
			}
			if tt.err.Code == "" {
				t.Errorf("%s.Code should not be empty", tt.name)
			}
			if tt.err.Message == "" {
				t.Errorf("%s.Message should not be empty", tt.name)
			}
		})
	}
}

func TestAppError_Chaining(t *testing.T) {
	err := ErrInsufficientFunds.
		WithDetails("Available: 100, Required: 500").
		WithError(errors.New("validation failed"))

	if err.Details != "Available: 100, Required: 500" {
		t.Error("Chaining should preserve details")
	}
	if err.Err == nil {
		t.Error("Chaining should set wrapped error")
	}
	if err.Code != "PAYMENT_INSUFFICIENT_FUNDS" {
		t.Error("Chaining should preserve code")
	}
}

func TestAppError_ImmutabilityOnChaining(t *testing.T) {
	original := ErrValidation

	_ = original.WithDetails("detail1")
	_ = original.WithMessage("new message")
	_ = original.WithError(errors.New("error"))

	if original.Details != nil {
		t.Error("Original should not be modified by WithDetails")
	}
	if original.Message != "Invalid input" {
		t.Error("Original should not be modified by WithMessage")
	}
	if original.Err != nil {
		t.Error("Original should not be modified by WithError")
	}
}
