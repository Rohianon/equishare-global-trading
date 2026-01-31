package types

import "time"

// PlaceOrderRequest is the request to place a new order
type PlaceOrderRequest struct {
	Symbol string  `json:"symbol"`
	Side   string  `json:"side"`   // buy, sell
	Amount float64 `json:"amount"` // Dollar amount for fractional shares
	Qty    float64 `json:"qty"`    // Number of shares (alternative to amount)
	Source string  `json:"source"` // web, mobile, ussd
}

// PlaceOrderResponse is the response after placing an order
type PlaceOrderResponse struct {
	OrderID       string  `json:"order_id"`
	AlpacaOrderID string  `json:"alpaca_order_id"`
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"`
	Message       string  `json:"message"`
}

// CancelOrderResponse is the response after canceling an order
type CancelOrderResponse struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Order represents a trading order
type Order struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	AlpacaOrderID  string     `json:"alpaca_order_id"`
	Symbol         string     `json:"symbol"`
	Side           string     `json:"side"`
	Type           string     `json:"type"`
	Amount         float64    `json:"amount"`
	Qty            float64    `json:"qty"`
	FilledQty      float64    `json:"filled_qty"`
	FilledAvgPrice float64    `json:"filled_avg_price"`
	Status         string     `json:"status"`
	Source         string     `json:"source"`
	FailedReason   *string    `json:"failed_reason,omitempty"`
	FilledAt       *time.Time `json:"filled_at,omitempty"`
	CanceledAt     *time.Time `json:"canceled_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Holding represents a user's stock holding
type Holding struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Symbol          string    `json:"symbol"`
	Qty             float64   `json:"qty"`
	AvgEntryPrice   float64   `json:"avg_entry_price"`
	CurrentPrice    float64   `json:"current_price"`
	MarketValue     float64   `json:"market_value"`
	UnrealizedPL    float64   `json:"unrealized_pl"`
	UnrealizedPLPct float64   `json:"unrealized_pl_pct"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Portfolio represents a user's complete portfolio
type Portfolio struct {
	Holdings   []Holding `json:"holdings"`
	TotalValue float64   `json:"total_value"`
	TotalPL    float64   `json:"total_pl"`
	TotalPLPct float64   `json:"total_pl_pct"`
	CashUSD    float64   `json:"cash_usd"`
}

// User represents user info needed for trading
type User struct {
	ID            string  `json:"id"`
	Phone         string  `json:"phone"`
	IsActive      bool    `json:"is_active"`
	IsKYCVerified bool    `json:"is_kyc_verified"`
}

// Wallet represents user wallet info
type Wallet struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	Currency      string    `json:"currency"`
	Balance       float64   `json:"balance"`
	LockedBalance float64   `json:"locked_balance"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AvailableBalance returns the balance available for trading
func (w *Wallet) AvailableBalance() float64 {
	return w.Balance - w.LockedBalance
}

// AlpacaWebhookEvent represents an Alpaca trade update webhook
type AlpacaWebhookEvent struct {
	Event string            `json:"event"`
	Order AlpacaOrderUpdate `json:"order"`
}

// AlpacaOrderUpdate represents order data from Alpaca webhook
type AlpacaOrderUpdate struct {
	ID             string     `json:"id"`
	ClientOrderID  string     `json:"client_order_id"`
	Symbol         string     `json:"symbol"`
	Side           string     `json:"side"`
	Type           string     `json:"type"`
	Qty            string     `json:"qty"`
	FilledQty      string     `json:"filled_qty"`
	FilledAvgPrice string     `json:"filled_avg_price"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	FilledAt       *time.Time `json:"filled_at"`
	CanceledAt     *time.Time `json:"canceled_at"`
	FailedAt       *time.Time `json:"failed_at"`
}

// QuoteResponse represents stock quote data
type QuoteResponse struct {
	Symbol    string  `json:"symbol"`
	BidPrice  float64 `json:"bid_price"`
	AskPrice  float64 `json:"ask_price"`
	LastPrice float64 `json:"last_price"`
}

// SearchResponse represents stock search results
type SearchResponse struct {
	Assets []AssetInfo `json:"assets"`
}

// AssetInfo represents basic asset information
type AssetInfo struct {
	Symbol       string `json:"symbol"`
	Name         string `json:"name"`
	Exchange     string `json:"exchange"`
	Tradable     bool   `json:"tradable"`
	Fractionable bool   `json:"fractionable"`
}
