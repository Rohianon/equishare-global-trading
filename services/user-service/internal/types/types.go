package types

import "time"

// User represents a user from the database
type User struct {
	ID              string     `json:"id"`
	Phone           string     `json:"phone"`
	Email           *string    `json:"email,omitempty"`
	FirstName       *string    `json:"first_name,omitempty"`
	LastName        *string    `json:"last_name,omitempty"`
	KYCStatus       string     `json:"kyc_status"`
	KYCTier         string     `json:"kyc_tier"`
	KYCSubmittedAt  *time.Time `json:"kyc_submitted_at,omitempty"`
	KYCVerifiedAt   *time.Time `json:"kyc_verified_at,omitempty"`
	AlpacaAccountID *string    `json:"alpaca_account_id,omitempty"`
	IsActive        bool       `json:"is_active"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// UserProfile represents the public profile response
type UserProfile struct {
	ID        string     `json:"id"`
	Phone     string     `json:"phone"`
	Email     *string    `json:"email,omitempty"`
	FirstName *string    `json:"first_name,omitempty"`
	LastName  *string    `json:"last_name,omitempty"`
	FullName  string     `json:"full_name"`
	KYCStatus string     `json:"kyc_status"`
	KYCTier   string     `json:"kyc_tier"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
}

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Email     *string `json:"email,omitempty"`
}

// UserSettings represents user preferences/settings
type UserSettings struct {
	UserID              string `json:"user_id"`
	NotifySMS           bool   `json:"notify_sms"`
	NotifyEmail         bool   `json:"notify_email"`
	NotifyPush          bool   `json:"notify_push"`
	DefaultCurrency     string `json:"default_currency"`
	Language            string `json:"language"`
	TwoFactorEnabled    bool   `json:"two_factor_enabled"`
}

// UpdateSettingsRequest represents a settings update request
type UpdateSettingsRequest struct {
	NotifySMS       *bool   `json:"notify_sms,omitempty"`
	NotifyEmail     *bool   `json:"notify_email,omitempty"`
	NotifyPush      *bool   `json:"notify_push,omitempty"`
	DefaultCurrency *string `json:"default_currency,omitempty"`
	Language        *string `json:"language,omitempty"`
}

// KYCStatusResponse represents KYC status information
type KYCStatusResponse struct {
	Status      string     `json:"status"`
	Tier        string     `json:"tier"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	Limits      KYCLimits  `json:"limits"`
}

// KYCLimits represents limits based on KYC tier
type KYCLimits struct {
	DailyDeposit    float64 `json:"daily_deposit"`
	DailyWithdrawal float64 `json:"daily_withdrawal"`
	DailyTrade      float64 `json:"daily_trade"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
