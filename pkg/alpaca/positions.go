package alpaca

import (
	"context"
	"fmt"
)

// Position represents a current holding
type Position struct {
	AssetID           string `json:"asset_id"`
	Symbol            string `json:"symbol"`
	Exchange          string `json:"exchange"`
	AssetClass        string `json:"asset_class"`
	AssetMarginable   bool   `json:"asset_marginable"`
	Qty               string `json:"qty"`
	AvgEntryPrice     string `json:"avg_entry_price"`
	Side              string `json:"side"`
	MarketValue       string `json:"market_value"`
	CostBasis         string `json:"cost_basis"`
	UnrealizedPL      string `json:"unrealized_pl"`
	UnrealizedPLPC    string `json:"unrealized_plpc"`
	UnrealizedIntradayPL   string `json:"unrealized_intraday_pl"`
	UnrealizedIntradayPLPC string `json:"unrealized_intraday_plpc"`
	CurrentPrice      string `json:"current_price"`
	LastdayPrice      string `json:"lastday_price"`
	ChangeToday       string `json:"change_today"`
	QtyAvailable      string `json:"qty_available"`
}

// ListPositions retrieves all open positions
func (c *Client) ListPositions(ctx context.Context) ([]Position, error) {
	url := fmt.Sprintf("%s/v2/positions", c.baseURL)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var positions []Position
	if err := decodeResponse(resp, &positions); err != nil {
		return nil, err
	}

	return positions, nil
}

// GetPosition retrieves a specific position by symbol
func (c *Client) GetPosition(ctx context.Context, symbol string) (*Position, error) {
	url := fmt.Sprintf("%s/v2/positions/%s", c.baseURL, symbol)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var position Position
	if err := decodeResponse(resp, &position); err != nil {
		return nil, err
	}

	return &position, nil
}

// ClosePosition closes (liquidates) a position by symbol
func (c *Client) ClosePosition(ctx context.Context, symbol string, qty string) (*Order, error) {
	url := fmt.Sprintf("%s/v2/positions/%s", c.baseURL, symbol)
	if qty != "" {
		url += fmt.Sprintf("?qty=%s", qty)
	}

	resp, err := c.doRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	var order Order
	if err := decodeResponse(resp, &order); err != nil {
		return nil, err
	}

	return &order, nil
}

// CloseAllPositions closes all open positions
func (c *Client) CloseAllPositions(ctx context.Context, cancelOrders bool) ([]Order, error) {
	url := fmt.Sprintf("%s/v2/positions?cancel_orders=%t", c.baseURL, cancelOrders)
	resp, err := c.doRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	var orders []Order
	if err := decodeResponse(resp, &orders); err != nil {
		return nil, err
	}

	return orders, nil
}

// Asset represents a tradeable asset
type Asset struct {
	ID           string `json:"id"`
	Class        string `json:"class"`
	Exchange     string `json:"exchange"`
	Symbol       string `json:"symbol"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	Tradable     bool   `json:"tradable"`
	Marginable   bool   `json:"marginable"`
	Shortable    bool   `json:"shortable"`
	EasyToBorrow bool   `json:"easy_to_borrow"`
	Fractionable bool   `json:"fractionable"`
}

// GetAsset retrieves asset information by symbol
func (c *Client) GetAsset(ctx context.Context, symbol string) (*Asset, error) {
	url := fmt.Sprintf("%s/v2/assets/%s", c.baseURL, symbol)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var asset Asset
	if err := decodeResponse(resp, &asset); err != nil {
		return nil, err
	}

	return &asset, nil
}

// ListAssetsParams contains parameters for listing assets
type ListAssetsParams struct {
	Status     string // active, inactive
	AssetClass string // us_equity, crypto
}

// ListAssets retrieves a list of tradeable assets
func (c *Client) ListAssets(ctx context.Context, params *ListAssetsParams) ([]Asset, error) {
	url := fmt.Sprintf("%s/v2/assets", c.baseURL)
	if params != nil {
		query := "?"
		if params.Status != "" {
			query += fmt.Sprintf("status=%s&", params.Status)
		}
		if params.AssetClass != "" {
			query += fmt.Sprintf("asset_class=%s&", params.AssetClass)
		}
		url += query
	}

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var assets []Asset
	if err := decodeResponse(resp, &assets); err != nil {
		return nil, err
	}

	return assets, nil
}

// Quote represents current market data for an asset
type Quote struct {
	Symbol    string  `json:"symbol"`
	BidPrice  float64 `json:"bid_price"`
	BidSize   int     `json:"bid_size"`
	AskPrice  float64 `json:"ask_price"`
	AskSize   int     `json:"ask_size"`
	Timestamp string  `json:"timestamp"`
}

// GetQuote retrieves the latest quote for a symbol
func (c *Client) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	url := fmt.Sprintf("%s/v2/stocks/%s/quotes/latest", c.dataURL, symbol)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Quote Quote `json:"quote"`
	}
	if err := decodeResponse(resp, &result); err != nil {
		return nil, err
	}

	result.Quote.Symbol = symbol
	return &result.Quote, nil
}
