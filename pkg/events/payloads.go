package events

import "time"

// =============================================================================
// Event Payload Definitions
// =============================================================================
// Each event type has a corresponding payload struct defined here.
// When adding new events, define the payload structure here first.
//
// Guidelines:
// - Use primitive types where possible (string, int, float64, bool)
// - Use time.Time for timestamps
// - Use pointers for optional fields
// - Include user_id in all user-related payloads
// - Include correlation fields to link related entities
// =============================================================================

// OrderCreatedPayload is the payload for order.created.v1 events
type OrderCreatedPayload struct {
	OrderID       string  `json:"order_id"`
	UserID        string  `json:"user_id"`
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"` // buy, sell
	Type          string  `json:"type"` // market, limit
	Amount        float64 `json:"amount,omitempty"`
	Qty           float64 `json:"qty,omitempty"`
	LimitPrice    float64 `json:"limit_price,omitempty"`
	Source        string  `json:"source"` // web, mobile, ussd, api
	AlpacaOrderID string  `json:"alpaca_order_id,omitempty"`
}

// OrderFilledPayload is the payload for order.filled.v1 events
type OrderFilledPayload struct {
	OrderID        string    `json:"order_id"`
	UserID         string    `json:"user_id"`
	Symbol         string    `json:"symbol"`
	Side           string    `json:"side"`
	FilledQty      float64   `json:"filled_qty"`
	FilledAvgPrice float64   `json:"filled_avg_price"`
	TotalValue     float64   `json:"total_value"`
	FilledAt       time.Time `json:"filled_at"`
}

// OrderPartialFillPayload is the payload for order.partial_fill.v1 events
type OrderPartialFillPayload struct {
	OrderID        string  `json:"order_id"`
	UserID         string  `json:"user_id"`
	Symbol         string  `json:"symbol"`
	FilledQty      float64 `json:"filled_qty"`
	RemainingQty   float64 `json:"remaining_qty"`
	FilledAvgPrice float64 `json:"filled_avg_price"`
}

// OrderCancelledPayload is the payload for order.cancelled.v1 events
type OrderCancelledPayload struct {
	OrderID      string    `json:"order_id"`
	UserID       string    `json:"user_id"`
	Symbol       string    `json:"symbol"`
	CancelledAt  time.Time `json:"cancelled_at"`
	CancelReason string    `json:"cancel_reason,omitempty"`
}

// OrderRejectedPayload is the payload for order.rejected.v1 events
type OrderRejectedPayload struct {
	OrderID      string `json:"order_id"`
	UserID       string `json:"user_id"`
	Symbol       string `json:"symbol"`
	RejectReason string `json:"reject_reason"`
}

// PaymentInitiatedPayload is the payload for payment.initiated.v1 events
type PaymentInitiatedPayload struct {
	UserID            string  `json:"user_id"`
	WalletID          string  `json:"wallet_id"`
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency"`
	Provider          string  `json:"provider"` // mpesa, card, bank
	CheckoutRequestID string  `json:"checkout_request_id,omitempty"`
}

// PaymentCompletedPayload is the payload for payment.completed.v1 events
type PaymentCompletedPayload struct {
	UserID        string    `json:"user_id"`
	WalletID      string    `json:"wallet_id"`
	TransactionID string    `json:"transaction_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Provider      string    `json:"provider"`
	ProviderRef   string    `json:"provider_ref"` // e.g., M-Pesa receipt number
	CompletedAt   time.Time `json:"completed_at"`
	NewBalance    float64   `json:"new_balance"`
}

// PaymentFailedPayload is the payload for payment.failed.v1 events
type PaymentFailedPayload struct {
	UserID       string `json:"user_id"`
	WalletID     string `json:"wallet_id"`
	Amount       float64 `json:"amount"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	FailureCode  string `json:"failure_code"`
	FailureReason string `json:"failure_reason"`
}

// WithdrawalInitiatedPayload is the payload for withdrawal.initiated.v1 events
type WithdrawalInitiatedPayload struct {
	WithdrawalID string  `json:"withdrawal_id"`
	UserID       string  `json:"user_id"`
	WalletID     string  `json:"wallet_id"`
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	Destination  string  `json:"destination"` // phone number or bank account
	Provider     string  `json:"provider"`    // mpesa, bank
}

// WithdrawalCompletedPayload is the payload for withdrawal.completed.v1 events
type WithdrawalCompletedPayload struct {
	WithdrawalID string    `json:"withdrawal_id"`
	UserID       string    `json:"user_id"`
	Amount       float64   `json:"amount"`
	Currency     string    `json:"currency"`
	ProviderRef  string    `json:"provider_ref"`
	CompletedAt  time.Time `json:"completed_at"`
}

// WithdrawalFailedPayload is the payload for withdrawal.failed.v1 events
type WithdrawalFailedPayload struct {
	WithdrawalID  string `json:"withdrawal_id"`
	UserID        string `json:"user_id"`
	Amount        float64 `json:"amount"`
	Currency      string `json:"currency"`
	FailureCode   string `json:"failure_code"`
	FailureReason string `json:"failure_reason"`
}

// KYCSubmittedPayload is the payload for kyc.submitted.v1 events
type KYCSubmittedPayload struct {
	UserID       string   `json:"user_id"`
	DocumentType string   `json:"document_type"` // national_id, passport, drivers_license
	Documents    []string `json:"documents"`     // document IDs/URLs
	SubmittedAt  time.Time `json:"submitted_at"`
}

// KYCVerifiedPayload is the payload for kyc.verified.v1 events
type KYCVerifiedPayload struct {
	UserID     string    `json:"user_id"`
	VerifiedAt time.Time `json:"verified_at"`
	VerifiedBy string    `json:"verified_by,omitempty"` // manual or provider name
}

// KYCRejectedPayload is the payload for kyc.rejected.v1 events
type KYCRejectedPayload struct {
	UserID       string `json:"user_id"`
	RejectReason string `json:"reject_reason"`
	CanRetry     bool   `json:"can_retry"`
}

// UserRegisteredPayload is the payload for user.registered.v1 events
type UserRegisteredPayload struct {
	UserID       string    `json:"user_id"`
	Phone        string    `json:"phone"`
	Email        string    `json:"email,omitempty"`
	RegisteredAt time.Time `json:"registered_at"`
	Source       string    `json:"source"` // app, ussd, web
}

// UserVerifiedPayload is the payload for user.verified.v1 events
type UserVerifiedPayload struct {
	UserID       string    `json:"user_id"`
	VerifiedType string    `json:"verified_type"` // phone, email
	VerifiedAt   time.Time `json:"verified_at"`
}

// PriceUpdatePayload is the payload for price.update.v1 events
type PriceUpdatePayload struct {
	Symbol    string    `json:"symbol"`
	BidPrice  float64   `json:"bid_price"`
	AskPrice  float64   `json:"ask_price"`
	LastPrice float64   `json:"last_price"`
	Volume    int64     `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

// MarketStatusPayload is the payload for market.open.v1 and market.close.v1 events
type MarketStatusPayload struct {
	Exchange  string    `json:"exchange"`
	Status    string    `json:"status"` // open, close, halt
	Timestamp time.Time `json:"timestamp"`
}

// NotificationPayload is the payload for notification.send.v1 events
type NotificationPayload struct {
	UserID       string            `json:"user_id"`
	Channel      string            `json:"channel"`  // sms, push, email
	Template     string            `json:"template"` // template name
	TemplateData map[string]string `json:"template_data"`
	Priority     string            `json:"priority,omitempty"` // high, normal, low
}

// AlertTriggeredPayload is the payload for alert.triggered.v1 events
type AlertTriggeredPayload struct {
	AlertID      string    `json:"alert_id"`
	UserID       string    `json:"user_id"`
	Symbol       string    `json:"symbol"`
	AlertType    string    `json:"alert_type"`    // price_above, price_below, percent_change
	TargetValue  float64   `json:"target_value"`
	CurrentValue float64   `json:"current_value"`
	TriggeredAt  time.Time `json:"triggered_at"`
}
