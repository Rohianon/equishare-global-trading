package alpaca

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Bar represents OHLCV data for a time period
type Bar struct {
	Timestamp  time.Time `json:"t"`
	Open       float64   `json:"o"`
	High       float64   `json:"h"`
	Low        float64   `json:"l"`
	Close      float64   `json:"c"`
	Volume     uint64    `json:"v"`
	TradeCount uint64    `json:"n"`
	VWAP       float64   `json:"vw"`
}

// BarsResponse represents the response from the bars endpoint
type BarsResponse struct {
	Bars          map[string][]Bar `json:"bars"`
	NextPageToken string           `json:"next_page_token,omitempty"`
}

// GetBarsParams contains parameters for fetching historical bars
type GetBarsParams struct {
	Timeframe string    // 1Min, 5Min, 15Min, 1Hour, 1Day, 1Week, 1Month
	Start     time.Time // Start of time range
	End       time.Time // End of time range (optional, defaults to now)
	Limit     int       // Max number of bars (default 1000, max 10000)
	Feed      string    // iex, sip (default: iex for free tier)
}

// GetBars retrieves historical bars for a symbol
func (c *Client) GetBars(ctx context.Context, symbol string, params *GetBarsParams) ([]Bar, error) {
	if params == nil {
		params = &GetBarsParams{
			Timeframe: "1Day",
			Limit:     100,
		}
	}

	if params.Timeframe == "" {
		params.Timeframe = "1Day"
	}
	if params.Limit == 0 {
		params.Limit = 100
	}
	if params.Feed == "" {
		params.Feed = "iex"
	}

	query := url.Values{}
	query.Set("timeframe", params.Timeframe)
	query.Set("limit", fmt.Sprintf("%d", params.Limit))
	query.Set("feed", params.Feed)

	if !params.Start.IsZero() {
		query.Set("start", params.Start.Format(time.RFC3339))
	}
	if !params.End.IsZero() {
		query.Set("end", params.End.Format(time.RFC3339))
	}

	apiURL := fmt.Sprintf("%s/v2/stocks/%s/bars?%s", c.dataURL, symbol, query.Encode())
	resp, err := c.doRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Bars          []Bar  `json:"bars"`
		NextPageToken string `json:"next_page_token,omitempty"`
	}
	if err := decodeResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Bars, nil
}

// MultiQuotesResponse represents the response from multi-quotes endpoint
type MultiQuotesResponse struct {
	Quotes map[string]Quote `json:"quotes"`
}

// GetMultiQuotes retrieves quotes for multiple symbols
func (c *Client) GetMultiQuotes(ctx context.Context, symbols []string) (map[string]Quote, error) {
	if len(symbols) == 0 {
		return make(map[string]Quote), nil
	}

	symbolsParam := strings.Join(symbols, ",")
	apiURL := fmt.Sprintf("%s/v2/stocks/quotes/latest?symbols=%s&feed=iex", c.dataURL, url.QueryEscape(symbolsParam))

	resp, err := c.doRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Quotes map[string]struct {
			AskPrice  float64   `json:"ap"`
			AskSize   int       `json:"as"`
			BidPrice  float64   `json:"bp"`
			BidSize   int       `json:"bs"`
			Timestamp time.Time `json:"t"`
		} `json:"quotes"`
	}
	if err := decodeResponse(resp, &result); err != nil {
		return nil, err
	}

	// Convert to our Quote type
	quotes := make(map[string]Quote)
	for symbol, q := range result.Quotes {
		quotes[symbol] = Quote{
			Symbol:    symbol,
			AskPrice:  q.AskPrice,
			AskSize:   q.AskSize,
			BidPrice:  q.BidPrice,
			BidSize:   q.BidSize,
			Timestamp: q.Timestamp.Format(time.RFC3339),
		}
	}

	return quotes, nil
}

// Clock represents market clock information
type Clock struct {
	Timestamp time.Time `json:"timestamp"`
	IsOpen    bool      `json:"is_open"`
	NextOpen  time.Time `json:"next_open"`
	NextClose time.Time `json:"next_close"`
}

// GetClock retrieves the current market clock
func (c *Client) GetClock(ctx context.Context) (*Clock, error) {
	apiURL := fmt.Sprintf("%s/v2/clock", c.baseURL)
	resp, err := c.doRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	var clock Clock
	if err := decodeResponse(resp, &clock); err != nil {
		return nil, err
	}

	return &clock, nil
}

// CalendarDay represents a single trading day
type CalendarDay struct {
	Date  string `json:"date"`
	Open  string `json:"open"`
	Close string `json:"close"`
}

// GetCalendarParams contains parameters for fetching calendar
type GetCalendarParams struct {
	Start string // YYYY-MM-DD format
	End   string // YYYY-MM-DD format
}

// GetCalendar retrieves the market calendar
func (c *Client) GetCalendar(ctx context.Context, params *GetCalendarParams) ([]CalendarDay, error) {
	apiURL := fmt.Sprintf("%s/v2/calendar", c.baseURL)

	if params != nil {
		query := url.Values{}
		if params.Start != "" {
			query.Set("start", params.Start)
		}
		if params.End != "" {
			query.Set("end", params.End)
		}
		if len(query) > 0 {
			apiURL += "?" + query.Encode()
		}
	}

	resp, err := c.doRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	var calendar []CalendarDay
	if err := decodeResponse(resp, &calendar); err != nil {
		return nil, err
	}

	return calendar, nil
}

// Snapshot represents a complete market snapshot for a symbol
type Snapshot struct {
	LatestTrade  *Trade `json:"latestTrade,omitempty"`
	LatestQuote  *Quote `json:"latestQuote,omitempty"`
	MinuteBar    *Bar   `json:"minuteBar,omitempty"`
	DailyBar     *Bar   `json:"dailyBar,omitempty"`
	PrevDailyBar *Bar   `json:"prevDailyBar,omitempty"`
}

// Trade represents a single trade
type Trade struct {
	Timestamp time.Time `json:"t"`
	Price     float64   `json:"p"`
	Size      uint64    `json:"s"`
	Exchange  string    `json:"x"`
	ID        uint64    `json:"i"`
	Tape      string    `json:"z"`
}

// GetSnapshot retrieves a complete snapshot for a symbol
func (c *Client) GetSnapshot(ctx context.Context, symbol string) (*Snapshot, error) {
	apiURL := fmt.Sprintf("%s/v2/stocks/%s/snapshot?feed=iex", c.dataURL, symbol)
	resp, err := c.doRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	var snapshot Snapshot
	if err := decodeResponse(resp, &snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}
