package alpaca

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Config holds Alpaca API configuration
type Config struct {
	APIKey    string
	SecretKey string
	Paper     bool // Use paper trading environment
}

// Client is the Alpaca API client
type Client struct {
	config     *Config
	httpClient *http.Client
	baseURL    string
	dataURL    string
}

// NewClient creates a new Alpaca API client
func NewClient(cfg *Config) *Client {
	baseURL := "https://api.alpaca.markets"
	dataURL := "https://data.alpaca.markets"
	if cfg.Paper {
		baseURL = "https://paper-api.alpaca.markets"
	}

	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		dataURL: dataURL,
	}
}

// doRequest performs an authenticated HTTP request to Alpaca
func (c *Client) doRequest(ctx context.Context, method, url string, body any) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("APCA-API-KEY-ID", c.config.APIKey)
	req.Header.Set("APCA-API-SECRET-KEY", c.config.SecretKey)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// decodeResponse decodes a JSON response into the target
func decodeResponse(resp *http.Response, target any) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// GetAccount retrieves the trading account information
func (c *Client) GetAccount(ctx context.Context) (*Account, error) {
	url := fmt.Sprintf("%s/v2/account", c.baseURL)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var account Account
	if err := decodeResponse(resp, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

// Account represents an Alpaca trading account
type Account struct {
	ID                    string `json:"id"`
	AccountNumber         string `json:"account_number"`
	Status                string `json:"status"`
	Currency              string `json:"currency"`
	Cash                  string `json:"cash"`
	PortfolioValue        string `json:"portfolio_value"`
	PatternDayTrader      bool   `json:"pattern_day_trader"`
	TradingBlocked        bool   `json:"trading_blocked"`
	TransfersBlocked      bool   `json:"transfers_blocked"`
	AccountBlocked        bool   `json:"account_blocked"`
	CreatedAt             string `json:"created_at"`
	ShortingEnabled       bool   `json:"shorting_enabled"`
	Multiplier            string `json:"multiplier"`
	BuyingPower           string `json:"buying_power"`
	DaytradingBuyingPower string `json:"daytrading_buying_power"`
	Equity                string `json:"equity"`
	LastEquity            string `json:"last_equity"`
	InitialMargin         string `json:"initial_margin"`
	MaintenanceMargin     string `json:"maintenance_margin"`
}
