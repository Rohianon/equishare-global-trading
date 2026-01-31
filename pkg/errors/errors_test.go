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

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        *AppError
		httpStatus int
	}{
		{"ErrUnauthorized", ErrUnauthorized, http.StatusUnauthorized},
		{"ErrForbidden", ErrForbidden, http.StatusForbidden},
		{"ErrInvalidCredentials", ErrInvalidCredentials, http.StatusUnauthorized},
		{"ErrValidation", ErrValidation, http.StatusBadRequest},
		{"ErrInvalidPhone", ErrInvalidPhone, http.StatusBadRequest},
		{"ErrNotFound", ErrNotFound, http.StatusNotFound},
		{"ErrConflict", ErrConflict, http.StatusConflict},
		{"ErrInsufficientFunds", ErrInsufficientFunds, http.StatusBadRequest},
		{"ErrKYCRequired", ErrKYCRequired, http.StatusForbidden},
		{"ErrTradingHoursClosed", ErrTradingHoursClosed, http.StatusBadRequest},
		{"ErrOrderLimitExceeded", ErrOrderLimitExceeded, http.StatusBadRequest},
		{"ErrRateLimited", ErrRateLimited, http.StatusTooManyRequests},
		{"ErrInternal", ErrInternal, http.StatusInternalServerError},
		{"ErrServiceUnavailable", ErrServiceUnavailable, http.StatusServiceUnavailable},
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
