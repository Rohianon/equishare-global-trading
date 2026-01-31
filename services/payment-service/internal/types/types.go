package types

import "time"

type DepositRequest struct {
	Amount int `json:"amount" validate:"required,min=10,max=150000"`
}

type DepositResponse struct {
	CheckoutRequestID string `json:"checkout_request_id"`
	Message           string `json:"message"`
	Amount            int    `json:"amount"`
	Currency          string `json:"currency"`
}

type MpesaTransaction struct {
	ID                string
	UserID            string
	TransactionID     *string
	CheckoutRequestID string
	MerchantRequestID string
	Amount            float64
	Phone             string
	Status            string
	MpesaReceipt      *string
	ResultCode        *int
	ResultDesc        *string
	CallbackPayload   []byte
	CreatedAt         time.Time
	UpdatedAt         time.Time
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

type Transaction struct {
	ID          string
	UserID      string
	WalletID    string
	Type        string
	Status      string
	Amount      float64
	Fee         float64
	Currency    string
	Provider    string
	ProviderRef *string
	Description *string
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type WebhookResponse struct {
	ResultCode int    `json:"ResultCode"`
	ResultDesc string `json:"ResultDesc"`
}
