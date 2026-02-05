package types

import "time"

// QuoteResponse represents a single stock quote
type QuoteResponse struct {
	Symbol    string  `json:"symbol"`
	BidPrice  float64 `json:"bid_price"`
	BidSize   int     `json:"bid_size"`
	AskPrice  float64 `json:"ask_price"`
	AskSize   int     `json:"ask_size"`
	MidPrice  float64 `json:"mid_price"`
	Spread    float64 `json:"spread"`
	Timestamp string  `json:"timestamp"`
}

// MultiQuoteRequest represents a request for multiple quotes
type MultiQuoteRequest struct {
	Symbols []string `json:"symbols" query:"symbols"`
}

// MultiQuoteResponse represents quotes for multiple symbols
type MultiQuoteResponse struct {
	Quotes map[string]QuoteResponse `json:"quotes"`
}

// BarResponse represents a single OHLCV bar
type BarResponse struct {
	Timestamp  time.Time `json:"timestamp"`
	Open       float64   `json:"open"`
	High       float64   `json:"high"`
	Low        float64   `json:"low"`
	Close      float64   `json:"close"`
	Volume     uint64    `json:"volume"`
	TradeCount uint64    `json:"trade_count"`
	VWAP       float64   `json:"vwap"`
}

// BarsRequest represents parameters for fetching bars
type BarsRequest struct {
	Timeframe string `json:"timeframe" query:"timeframe"` // 1Min, 5Min, 15Min, 1Hour, 1Day
	Start     string `json:"start" query:"start"`         // RFC3339 or YYYY-MM-DD
	End       string `json:"end" query:"end"`             // RFC3339 or YYYY-MM-DD
	Limit     int    `json:"limit" query:"limit"`         // Max 10000
}

// BarsResponse represents historical bars
type BarsResponse struct {
	Symbol string        `json:"symbol"`
	Bars   []BarResponse `json:"bars"`
}

// AssetResponse represents asset information
type AssetResponse struct {
	ID           string `json:"id"`
	Symbol       string `json:"symbol"`
	Name         string `json:"name"`
	Exchange     string `json:"exchange"`
	Class        string `json:"class"`
	Status       string `json:"status"`
	Tradable     bool   `json:"tradable"`
	Fractionable bool   `json:"fractionable"`
	Marginable   bool   `json:"marginable"`
	Shortable    bool   `json:"shortable"`
}

// AssetSearchRequest represents search parameters
type AssetSearchRequest struct {
	Query  string `json:"query" query:"q"`
	Class  string `json:"class" query:"class"`   // us_equity, crypto
	Status string `json:"status" query:"status"` // active, inactive
	Limit  int    `json:"limit" query:"limit"`
}

// AssetSearchResponse represents search results
type AssetSearchResponse struct {
	Assets []AssetResponse `json:"assets"`
	Total  int             `json:"total"`
}

// ClockResponse represents market clock status
type ClockResponse struct {
	Timestamp string `json:"timestamp"`
	IsOpen    bool   `json:"is_open"`
	NextOpen  string `json:"next_open"`
	NextClose string `json:"next_close"`
}

// CalendarDayResponse represents a trading day
type CalendarDayResponse struct {
	Date  string `json:"date"`
	Open  string `json:"open"`
	Close string `json:"close"`
}

// CalendarRequest represents calendar query parameters
type CalendarRequest struct {
	Start string `json:"start" query:"start"` // YYYY-MM-DD
	End   string `json:"end" query:"end"`     // YYYY-MM-DD
}

// CalendarResponse represents market calendar
type CalendarResponse struct {
	Days []CalendarDayResponse `json:"days"`
}

// SnapshotResponse represents a complete market snapshot
type SnapshotResponse struct {
	Symbol      string        `json:"symbol"`
	LatestTrade *TradeInfo    `json:"latest_trade,omitempty"`
	LatestQuote *QuoteResponse `json:"latest_quote,omitempty"`
	MinuteBar   *BarResponse  `json:"minute_bar,omitempty"`
	DailyBar    *BarResponse  `json:"daily_bar,omitempty"`
	PrevDailyBar *BarResponse `json:"prev_daily_bar,omitempty"`
}

// TradeInfo represents a single trade
type TradeInfo struct {
	Timestamp string  `json:"timestamp"`
	Price     float64 `json:"price"`
	Size      uint64  `json:"size"`
	Exchange  string  `json:"exchange"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
