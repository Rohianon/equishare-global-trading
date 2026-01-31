package errors

import (
	"fmt"
	"net/http"
)

// =============================================================================
// Application Error
// =============================================================================
// AppError is the standard error type used across all services.
// It includes an error code, human-readable message, optional details,
// and the HTTP status code to return.
//
// Error codes follow the pattern: DOMAIN_ACTION or DOMAIN_CONDITION
// Examples: AUTH_INVALID_CREDENTIALS, ORDER_LIMIT_EXCEEDED, PAYMENT_FAILED
// =============================================================================

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

func (e *AppError) WithMessage(message string) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    message,
		Details:    e.Details,
		HTTPStatus: e.HTTPStatus,
		Err:        e.Err,
	}
}

// New creates a new AppError with the given code, message, and HTTP status
func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// =============================================================================
// Common HTTP Errors
// =============================================================================

var (
	ErrBadRequest = &AppError{
		Code:       "BAD_REQUEST",
		Message:    "Invalid request",
		HTTPStatus: http.StatusBadRequest,
	}

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

	ErrValidation = &AppError{
		Code:       "VALIDATION_ERROR",
		Message:    "Invalid input",
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

// =============================================================================
// Authentication & Authorization Errors
// =============================================================================

var (
	ErrInvalidCredentials = &AppError{
		Code:       "AUTH_INVALID_CREDENTIALS",
		Message:    "Invalid phone number or PIN",
		HTTPStatus: http.StatusUnauthorized,
	}

	ErrInvalidToken = &AppError{
		Code:       "AUTH_INVALID_TOKEN",
		Message:    "Invalid or expired token",
		HTTPStatus: http.StatusUnauthorized,
	}

	ErrTokenExpired = &AppError{
		Code:       "AUTH_TOKEN_EXPIRED",
		Message:    "Token has expired",
		HTTPStatus: http.StatusUnauthorized,
	}

	ErrSessionExpired = &AppError{
		Code:       "AUTH_SESSION_EXPIRED",
		Message:    "Session has expired, please login again",
		HTTPStatus: http.StatusUnauthorized,
	}

	ErrInvalidOTP = &AppError{
		Code:       "AUTH_INVALID_OTP",
		Message:    "Invalid or expired OTP",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrOTPExpired = &AppError{
		Code:       "AUTH_OTP_EXPIRED",
		Message:    "OTP has expired",
		HTTPStatus: http.StatusBadRequest,
	}

	Err2FARequired = &AppError{
		Code:       "AUTH_2FA_REQUIRED",
		Message:    "Two-factor authentication required",
		HTTPStatus: http.StatusForbidden,
	}

	ErrInvalid2FACode = &AppError{
		Code:       "AUTH_INVALID_2FA_CODE",
		Message:    "Invalid 2FA code",
		HTTPStatus: http.StatusBadRequest,
	}
)

// =============================================================================
// User Errors
// =============================================================================

var (
	ErrInvalidPhone = &AppError{
		Code:       "USER_INVALID_PHONE",
		Message:    "Invalid phone number format",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrPhoneAlreadyExists = &AppError{
		Code:       "USER_PHONE_EXISTS",
		Message:    "Phone number already registered",
		HTTPStatus: http.StatusConflict,
	}

	ErrEmailAlreadyExists = &AppError{
		Code:       "USER_EMAIL_EXISTS",
		Message:    "Email already registered",
		HTTPStatus: http.StatusConflict,
	}

	ErrUserNotFound = &AppError{
		Code:       "USER_NOT_FOUND",
		Message:    "User not found",
		HTTPStatus: http.StatusNotFound,
	}

	ErrUserDeactivated = &AppError{
		Code:       "USER_DEACTIVATED",
		Message:    "User account is deactivated",
		HTTPStatus: http.StatusForbidden,
	}

	ErrKYCRequired = &AppError{
		Code:       "USER_KYC_REQUIRED",
		Message:    "KYC verification required for this action",
		HTTPStatus: http.StatusForbidden,
	}

	ErrKYCPending = &AppError{
		Code:       "USER_KYC_PENDING",
		Message:    "KYC verification is pending",
		HTTPStatus: http.StatusForbidden,
	}

	ErrKYCRejected = &AppError{
		Code:       "USER_KYC_REJECTED",
		Message:    "KYC verification was rejected",
		HTTPStatus: http.StatusForbidden,
	}
)

// =============================================================================
// Payment & Wallet Errors
// =============================================================================

var (
	ErrInsufficientFunds = &AppError{
		Code:       "PAYMENT_INSUFFICIENT_FUNDS",
		Message:    "Insufficient balance for this transaction",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrWalletNotFound = &AppError{
		Code:       "PAYMENT_WALLET_NOT_FOUND",
		Message:    "Wallet not found",
		HTTPStatus: http.StatusNotFound,
	}

	ErrPaymentFailed = &AppError{
		Code:       "PAYMENT_FAILED",
		Message:    "Payment processing failed",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrPaymentPending = &AppError{
		Code:       "PAYMENT_PENDING",
		Message:    "Payment is still being processed",
		HTTPStatus: http.StatusAccepted,
	}

	ErrWithdrawalFailed = &AppError{
		Code:       "PAYMENT_WITHDRAWAL_FAILED",
		Message:    "Withdrawal processing failed",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrMinimumAmount = &AppError{
		Code:       "PAYMENT_MINIMUM_AMOUNT",
		Message:    "Amount is below minimum allowed",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrMaximumAmount = &AppError{
		Code:       "PAYMENT_MAXIMUM_AMOUNT",
		Message:    "Amount exceeds maximum allowed",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrDailyLimitExceeded = &AppError{
		Code:       "PAYMENT_DAILY_LIMIT",
		Message:    "Daily transaction limit exceeded",
		HTTPStatus: http.StatusBadRequest,
	}
)

// =============================================================================
// Trading Errors
// =============================================================================

var (
	ErrTradingHoursClosed = &AppError{
		Code:       "TRADING_MARKET_CLOSED",
		Message:    "Trading is currently closed",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrOrderNotFound = &AppError{
		Code:       "TRADING_ORDER_NOT_FOUND",
		Message:    "Order not found",
		HTTPStatus: http.StatusNotFound,
	}

	ErrOrderAlreadyFilled = &AppError{
		Code:       "TRADING_ORDER_FILLED",
		Message:    "Order has already been filled",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrOrderAlreadyCancelled = &AppError{
		Code:       "TRADING_ORDER_CANCELLED",
		Message:    "Order has already been cancelled",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrOrderLimitExceeded = &AppError{
		Code:       "TRADING_ORDER_LIMIT",
		Message:    "Order exceeds your daily limit",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrInvalidSymbol = &AppError{
		Code:       "TRADING_INVALID_SYMBOL",
		Message:    "Invalid or unsupported symbol",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrInvalidOrderType = &AppError{
		Code:       "TRADING_INVALID_ORDER_TYPE",
		Message:    "Invalid order type",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrInvalidOrderSide = &AppError{
		Code:       "TRADING_INVALID_ORDER_SIDE",
		Message:    "Invalid order side (must be buy or sell)",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrInvalidQuantity = &AppError{
		Code:       "TRADING_INVALID_QUANTITY",
		Message:    "Invalid quantity",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrSymbolNotTradeable = &AppError{
		Code:       "TRADING_SYMBOL_NOT_TRADEABLE",
		Message:    "Symbol is not currently tradeable",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrPositionNotFound = &AppError{
		Code:       "TRADING_POSITION_NOT_FOUND",
		Message:    "Position not found",
		HTTPStatus: http.StatusNotFound,
	}
)

// =============================================================================
// External Provider Errors
// =============================================================================

var (
	ErrMpesaUnavailable = &AppError{
		Code:       "PROVIDER_MPESA_UNAVAILABLE",
		Message:    "M-Pesa service is currently unavailable",
		HTTPStatus: http.StatusServiceUnavailable,
	}

	ErrAlpacaUnavailable = &AppError{
		Code:       "PROVIDER_ALPACA_UNAVAILABLE",
		Message:    "Trading service is currently unavailable",
		HTTPStatus: http.StatusServiceUnavailable,
	}

	ErrSMSUnavailable = &AppError{
		Code:       "PROVIDER_SMS_UNAVAILABLE",
		Message:    "SMS service is currently unavailable",
		HTTPStatus: http.StatusServiceUnavailable,
	}
)
