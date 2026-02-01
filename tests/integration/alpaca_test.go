package integration

import (
	"testing"
	"time"
)

func TestAlpacaGetAccount(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	h := NewHarness(t)

	if err := h.WaitForAlpaca(30 * time.Second); err != nil {
		t.Skipf("Alpaca mock not available: %v", err)
	}

	if err := h.ResetAlpaca(); err != nil {
		t.Fatalf("Failed to reset: %v", err)
	}

	resp, err := h.Do(Request{
		Method: "GET",
		URL:    h.Config().AlpacaURL + "/v2/account",
		Headers: map[string]string{
			"APCA-API-KEY-ID":     "test-key",
			"APCA-API-SECRET-KEY": "test-secret",
		},
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	h.AssertStatus(resp, 200)

	var account map[string]any
	if err := resp.JSON(&account); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if account["status"] != "ACTIVE" {
		t.Errorf("Expected status ACTIVE, got %v", account["status"])
	}

	if account["currency"] != "USD" {
		t.Errorf("Expected currency USD, got %v", account["currency"])
	}
}

func TestAlpacaCreateMarketOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	h := NewHarness(t)

	if err := h.WaitForAlpaca(30 * time.Second); err != nil {
		t.Skipf("Alpaca mock not available: %v", err)
	}

	if err := h.ResetAlpaca(); err != nil {
		t.Fatalf("Failed to reset: %v", err)
	}

	headers := map[string]string{
		"APCA-API-KEY-ID":     "test-key",
		"APCA-API-SECRET-KEY": "test-secret",
	}

	// Create market order
	resp, err := h.Do(Request{
		Method:  "POST",
		URL:     h.Config().AlpacaURL + "/v2/orders",
		Headers: headers,
		Body: map[string]any{
			"symbol":        "AAPL",
			"qty":           "10",
			"side":          "buy",
			"type":          "market",
			"time_in_force": "day",
		},
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	h.AssertStatus(resp, 201)

	var order map[string]any
	if err := resp.JSON(&order); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Market orders should be filled immediately
	if order["status"] != "filled" {
		t.Errorf("Expected status filled, got %v", order["status"])
	}

	if order["symbol"] != "AAPL" {
		t.Errorf("Expected symbol AAPL, got %v", order["symbol"])
	}

	if order["filled_qty"] != "10" {
		t.Errorf("Expected filled_qty 10, got %v", order["filled_qty"])
	}

	// Check position was created
	posResp, err := h.Do(Request{
		Method:  "GET",
		URL:     h.Config().AlpacaURL + "/v2/positions/AAPL",
		Headers: headers,
	})
	if err != nil {
		t.Fatalf("Position request failed: %v", err)
	}
	h.AssertStatus(posResp, 200)

	var pos map[string]any
	if err := posResp.JSON(&pos); err != nil {
		t.Fatalf("Failed to parse position: %v", err)
	}

	if pos["symbol"] != "AAPL" {
		t.Errorf("Expected position symbol AAPL, got %v", pos["symbol"])
	}
}

func TestAlpacaCreateLimitOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	h := NewHarness(t)

	if err := h.WaitForAlpaca(30 * time.Second); err != nil {
		t.Skipf("Alpaca mock not available: %v", err)
	}

	if err := h.ResetAlpaca(); err != nil {
		t.Fatalf("Failed to reset: %v", err)
	}

	headers := map[string]string{
		"APCA-API-KEY-ID":     "test-key",
		"APCA-API-SECRET-KEY": "test-secret",
	}

	// Create limit order (won't fill immediately)
	resp, err := h.Do(Request{
		Method:  "POST",
		URL:     h.Config().AlpacaURL + "/v2/orders",
		Headers: headers,
		Body: map[string]any{
			"symbol":        "GOOGL",
			"qty":           "5",
			"side":          "buy",
			"type":          "limit",
			"time_in_force": "gtc",
			"limit_price":   "100.00",
		},
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	h.AssertStatus(resp, 201)

	var order map[string]any
	if err := resp.JSON(&order); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Limit orders should be in "new" status
	if order["status"] != "new" {
		t.Errorf("Expected status new, got %v", order["status"])
	}

	orderID := order["id"].(string)

	// Cancel the order
	cancelResp, err := h.Do(Request{
		Method:  "DELETE",
		URL:     h.Config().AlpacaURL + "/v2/orders/" + orderID,
		Headers: headers,
	})
	if err != nil {
		t.Fatalf("Cancel request failed: %v", err)
	}
	h.AssertStatus(cancelResp, 204)

	// Verify order was canceled
	getResp, err := h.Do(Request{
		Method:  "GET",
		URL:     h.Config().AlpacaURL + "/v2/orders/" + orderID,
		Headers: headers,
	})
	if err != nil {
		t.Fatalf("Get order failed: %v", err)
	}
	h.AssertStatus(getResp, 200)

	var canceledOrder map[string]any
	getResp.JSON(&canceledOrder)
	if canceledOrder["status"] != "canceled" {
		t.Errorf("Expected status canceled, got %v", canceledOrder["status"])
	}
}

func TestAlpacaClosePosition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	h := NewHarness(t)

	if err := h.WaitForAlpaca(30 * time.Second); err != nil {
		t.Skipf("Alpaca mock not available: %v", err)
	}

	if err := h.ResetAlpaca(); err != nil {
		t.Fatalf("Failed to reset: %v", err)
	}

	headers := map[string]string{
		"APCA-API-KEY-ID":     "test-key",
		"APCA-API-SECRET-KEY": "test-secret",
	}

	// Create a position
	_, err := h.Do(Request{
		Method:  "POST",
		URL:     h.Config().AlpacaURL + "/v2/orders",
		Headers: headers,
		Body: map[string]any{
			"symbol":        "MSFT",
			"qty":           "20",
			"side":          "buy",
			"type":          "market",
			"time_in_force": "day",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create position: %v", err)
	}

	// Close the position
	closeResp, err := h.Do(Request{
		Method:  "DELETE",
		URL:     h.Config().AlpacaURL + "/v2/positions/MSFT",
		Headers: headers,
	})
	if err != nil {
		t.Fatalf("Close position failed: %v", err)
	}
	h.AssertStatus(closeResp, 200)

	// Verify position is gone
	posResp, err := h.Do(Request{
		Method:  "GET",
		URL:     h.Config().AlpacaURL + "/v2/positions/MSFT",
		Headers: headers,
	})
	if err != nil {
		t.Fatalf("Get position failed: %v", err)
	}
	h.AssertStatus(posResp, 404)
}

func TestAlpacaListAssets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	h := NewHarness(t)

	if err := h.WaitForAlpaca(30 * time.Second); err != nil {
		t.Skipf("Alpaca mock not available: %v", err)
	}

	headers := map[string]string{
		"APCA-API-KEY-ID":     "test-key",
		"APCA-API-SECRET-KEY": "test-secret",
	}

	resp, err := h.Do(Request{
		Method:  "GET",
		URL:     h.Config().AlpacaURL + "/v2/assets",
		Headers: headers,
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	h.AssertStatus(resp, 200)

	var assets []map[string]any
	if err := resp.JSON(&assets); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(assets) == 0 {
		t.Error("Expected at least one asset")
	}

	// Check for AAPL
	found := false
	for _, asset := range assets {
		if asset["symbol"] == "AAPL" {
			found = true
			if asset["tradable"] != true {
				t.Error("Expected AAPL to be tradable")
			}
			break
		}
	}
	if !found {
		t.Error("AAPL not found in assets")
	}
}

func TestAlpacaUnauthorized(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	h := NewHarness(t)

	if err := h.WaitForAlpaca(30 * time.Second); err != nil {
		t.Skipf("Alpaca mock not available: %v", err)
	}

	// Request without auth headers
	resp, err := h.Do(Request{
		Method: "GET",
		URL:    h.Config().AlpacaURL + "/v2/account",
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	h.AssertStatus(resp, 401)
}

func TestAlpacaGetQuote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	h := NewHarness(t)

	if err := h.WaitForAlpaca(30 * time.Second); err != nil {
		t.Skipf("Alpaca mock not available: %v", err)
	}

	headers := map[string]string{
		"APCA-API-KEY-ID":     "test-key",
		"APCA-API-SECRET-KEY": "test-secret",
	}

	resp, err := h.Do(Request{
		Method:  "GET",
		URL:     h.Config().AlpacaURL + "/v2/stocks/AAPL/quotes/latest",
		Headers: headers,
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	h.AssertStatus(resp, 200)

	var data map[string]any
	if err := resp.JSON(&data); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	quote, ok := data["quote"].(map[string]any)
	if !ok {
		t.Fatal("No quote in response")
	}

	if quote["ap"] == nil || quote["bp"] == nil {
		t.Error("Quote missing ask/bid prices")
	}
}
