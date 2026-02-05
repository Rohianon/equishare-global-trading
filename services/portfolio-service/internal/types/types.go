package types

import "time"

// Holding represents a stock holding from the database
type Holding struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	Symbol        string    `json:"symbol"`
	Quantity      float64   `json:"quantity"`
	AvgCostBasis  float64   `json:"avg_cost_basis"`
	TotalCostBasis float64  `json:"total_cost_basis"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// HoldingWithPrice represents a holding with current market data
type HoldingWithPrice struct {
	Symbol           string  `json:"symbol"`
	Quantity         float64 `json:"quantity"`
	AvgCostBasis     float64 `json:"avg_cost_basis"`
	TotalCostBasis   float64 `json:"total_cost_basis"`
	CurrentPrice     float64 `json:"current_price"`
	MarketValue      float64 `json:"market_value"`
	UnrealizedPL     float64 `json:"unrealized_pl"`
	UnrealizedPLPct  float64 `json:"unrealized_pl_pct"`
	DayChange        float64 `json:"day_change"`
	DayChangePct     float64 `json:"day_change_pct"`
	AllocationPct    float64 `json:"allocation_pct"`
}

// PortfolioSummary represents the overall portfolio summary
type PortfolioSummary struct {
	TotalValue        float64 `json:"total_value"`
	TotalCostBasis    float64 `json:"total_cost_basis"`
	TotalUnrealizedPL float64 `json:"total_unrealized_pl"`
	TotalUnrealizedPLPct float64 `json:"total_unrealized_pl_pct"`
	DayChange         float64 `json:"day_change"`
	DayChangePct      float64 `json:"day_change_pct"`
	CashBalance       float64 `json:"cash_balance"`
	HoldingsCount     int     `json:"holdings_count"`
}

// PortfolioResponse represents the full portfolio response
type PortfolioResponse struct {
	Summary  PortfolioSummary   `json:"summary"`
	Holdings []HoldingWithPrice `json:"holdings"`
}

// HoldingsResponse represents list of holdings
type HoldingsResponse struct {
	Holdings []HoldingWithPrice `json:"holdings"`
	Total    int                `json:"total"`
}

// HoldingDetailResponse represents detailed holding info
type HoldingDetailResponse struct {
	Holding HoldingWithPrice `json:"holding"`
}

// AllocationItem represents portfolio allocation by symbol
type AllocationItem struct {
	Symbol        string  `json:"symbol"`
	MarketValue   float64 `json:"market_value"`
	AllocationPct float64 `json:"allocation_pct"`
}

// AllocationResponse represents portfolio allocation breakdown
type AllocationResponse struct {
	Allocations []AllocationItem `json:"allocations"`
	CashPct     float64          `json:"cash_pct"`
}

// PerformanceResponse represents portfolio performance metrics
type PerformanceResponse struct {
	TotalReturn      float64 `json:"total_return"`
	TotalReturnPct   float64 `json:"total_return_pct"`
	DayReturn        float64 `json:"day_return"`
	DayReturnPct     float64 `json:"day_return_pct"`
	BestPerformer    string  `json:"best_performer,omitempty"`
	WorstPerformer   string  `json:"worst_performer,omitempty"`
}

// Wallet represents a user wallet
type Wallet struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Currency  string    `json:"currency"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
