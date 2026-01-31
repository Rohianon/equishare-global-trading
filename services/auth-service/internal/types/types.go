package types

import "time"

type RegisterRequest struct {
	Phone string `json:"phone" validate:"required"`
}

type RegisterResponse struct {
	Message   string `json:"message"`
	ExpiresIn int    `json:"expires_in"`
}

type VerifyRequest struct {
	Phone    string `json:"phone" validate:"required"`
	OTP      string `json:"otp" validate:"required,len=6"`
	PIN      string `json:"pin" validate:"required,len=4"`
	Password string `json:"password,omitempty"`
}

type VerifyResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int          `json:"expires_in"`
}

type UserResponse struct {
	ID        string `json:"id"`
	Phone     string `json:"phone"`
	KYCStatus string `json:"kyc_status"`
	KYCTier   string `json:"kyc_tier"`
}

type User struct {
	ID              string
	Phone           string
	Email           *string
	PasswordHash    *string
	PINHash         *string
	FirstName       *string
	LastName        *string
	KYCStatus       string
	KYCTier         string
	AlpacaAccountID *string
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Wallet struct {
	ID            string
	UserID        string
	Currency      string
	Balance       float64
	LockedBalance float64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type LoginRequest struct {
	Phone    string `json:"phone" validate:"required"`
	PIN      string `json:"pin,omitempty"`
	Password string `json:"password,omitempty"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}
