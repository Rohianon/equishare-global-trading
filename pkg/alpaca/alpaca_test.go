package alpaca

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		paper   bool
		wantURL string
	}{
		{
			name:    "production client",
			paper:   false,
			wantURL: "https://api.alpaca.markets",
		},
		{
			name:    "paper trading client",
			paper:   true,
			wantURL: "https://paper-api.alpaca.markets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(&Config{
				APIKey:    "test-key",
				SecretKey: "test-secret",
				Paper:     tt.paper,
			})

			if client.baseURL != tt.wantURL {
				t.Errorf("baseURL = %v, want %v", client.baseURL, tt.wantURL)
			}
		})
	}
}

func TestMockClient_CreateOrder(t *testing.T) {
	ctx := context.Background()
	client := NewMockClient()

	order, err := client.CreateOrder(ctx, &CreateOrderRequest{
		Symbol:      "AAPL",
		Qty:         "10",
		Side:        Buy,
		Type:        Market,
		TimeInForce: Day,
	})

	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	if order.Symbol != "AAPL" {
		t.Errorf("order.Symbol = %v, want AAPL", order.Symbol)
	}

	if order.Status != OrderStatusFilled {
		t.Errorf("order.Status = %v, want %v", order.Status, OrderStatusFilled)
	}

	if order.Side != Buy {
		t.Errorf("order.Side = %v, want %v", order.Side, Buy)
	}
}

func TestMockClient_GetOrder(t *testing.T) {
	ctx := context.Background()
	client := NewMockClient()

	created, _ := client.CreateOrder(ctx, &CreateOrderRequest{
		Symbol:      "GOOGL",
		Qty:         "5",
		Side:        Buy,
		Type:        Market,
		TimeInForce: Day,
	})

	retrieved, err := client.GetOrder(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetOrder failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("retrieved.ID = %v, want %v", retrieved.ID, created.ID)
	}

	// Test order not found
	_, err = client.GetOrder(ctx, "non-existent")
	if err == nil {
		t.Error("expected error for non-existent order")
	}
}

func TestMockClient_ListPositions(t *testing.T) {
	ctx := context.Background()
	client := NewMockClient()

	// Initially no positions
	positions, err := client.ListPositions(ctx)
	if err != nil {
		t.Fatalf("ListPositions failed: %v", err)
	}
	if len(positions) != 0 {
		t.Errorf("expected 0 positions, got %d", len(positions))
	}

	// Create an order to generate a position
	_, _ = client.CreateOrder(ctx, &CreateOrderRequest{
		Symbol:      "MSFT",
		Qty:         "10",
		Side:        Buy,
		Type:        Market,
		TimeInForce: Day,
	})

	positions, err = client.ListPositions(ctx)
	if err != nil {
		t.Fatalf("ListPositions failed: %v", err)
	}
	if len(positions) != 1 {
		t.Errorf("expected 1 position, got %d", len(positions))
	}
}

func TestMockClient_CancelOrder(t *testing.T) {
	ctx := context.Background()
	client := NewMockClient()

	// Create a limit order (won't be filled immediately)
	order, _ := client.CreateOrder(ctx, &CreateOrderRequest{
		Symbol:      "AMZN",
		Qty:         "1",
		Side:        Buy,
		Type:        Limit,
		LimitPrice:  "100.00",
		TimeInForce: GTC,
	})

	err := client.CancelOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("CancelOrder failed: %v", err)
	}

	canceled, _ := client.GetOrder(ctx, order.ID)
	if canceled.Status != OrderStatusCanceled {
		t.Errorf("order.Status = %v, want %v", canceled.Status, OrderStatusCanceled)
	}
}

func TestMockClient_GetAccount(t *testing.T) {
	ctx := context.Background()
	client := NewMockClient()

	account, err := client.GetAccount(ctx)
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}

	if account.Status != "ACTIVE" {
		t.Errorf("account.Status = %v, want ACTIVE", account.Status)
	}

	if account.Currency != "USD" {
		t.Errorf("account.Currency = %v, want USD", account.Currency)
	}
}

func TestMockClient_GetAsset(t *testing.T) {
	ctx := context.Background()
	client := NewMockClient()

	asset, err := client.GetAsset(ctx, "TSLA")
	if err != nil {
		t.Fatalf("GetAsset failed: %v", err)
	}

	if asset.Symbol != "TSLA" {
		t.Errorf("asset.Symbol = %v, want TSLA", asset.Symbol)
	}

	if !asset.Tradable {
		t.Error("expected asset to be tradable")
	}

	if !asset.Fractionable {
		t.Error("expected asset to be fractionable")
	}
}

func TestMockClient_GetQuote(t *testing.T) {
	ctx := context.Background()
	client := NewMockClient()

	quote, err := client.GetQuote(ctx, "AAPL")
	if err != nil {
		t.Fatalf("GetQuote failed: %v", err)
	}

	if quote.Symbol != "AAPL" {
		t.Errorf("quote.Symbol = %v, want AAPL", quote.Symbol)
	}

	if quote.BidPrice <= 0 {
		t.Error("expected positive bid price")
	}

	if quote.AskPrice <= 0 {
		t.Error("expected positive ask price")
	}
}

func TestMockClient_ClosePosition(t *testing.T) {
	ctx := context.Background()
	client := NewMockClient()

	// Add a position
	client.AddMockPosition("NVDA", "50", "450.00")

	// Close the position
	order, err := client.ClosePosition(ctx, "NVDA", "")
	if err != nil {
		t.Fatalf("ClosePosition failed: %v", err)
	}

	if order.Side != Sell {
		t.Errorf("order.Side = %v, want %v", order.Side, Sell)
	}

	// Position should be gone
	_, err = client.GetPosition(ctx, "NVDA")
	if err == nil {
		t.Error("expected position to be closed")
	}
}

func TestMockClient_ListAssets(t *testing.T) {
	ctx := context.Background()
	client := NewMockClient()

	assets, err := client.ListAssets(ctx, nil)
	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}

	if len(assets) == 0 {
		t.Error("expected some assets")
	}

	// Check that we have expected assets
	symbols := make(map[string]bool)
	for _, a := range assets {
		symbols[a.Symbol] = true
	}

	expected := []string{"AAPL", "GOOGL", "MSFT", "AMZN", "TSLA"}
	for _, s := range expected {
		if !symbols[s] {
			t.Errorf("expected asset %s not found", s)
		}
	}
}

func TestOrderStatus_Values(t *testing.T) {
	statuses := []OrderStatus{
		OrderStatusNew,
		OrderStatusFilled,
		OrderStatusCanceled,
		OrderStatusPartiallyFilled,
		OrderStatusRejected,
	}

	for _, s := range statuses {
		if s == "" {
			t.Error("order status should not be empty")
		}
	}
}

func TestOrderSide_Values(t *testing.T) {
	if Buy != "buy" {
		t.Errorf("Buy = %v, want buy", Buy)
	}
	if Sell != "sell" {
		t.Errorf("Sell = %v, want sell", Sell)
	}
}

func TestOrderType_Values(t *testing.T) {
	if Market != "market" {
		t.Errorf("Market = %v, want market", Market)
	}
	if Limit != "limit" {
		t.Errorf("Limit = %v, want limit", Limit)
	}
}

func TestTimeInForce_Values(t *testing.T) {
	if Day != "day" {
		t.Errorf("Day = %v, want day", Day)
	}
	if GTC != "gtc" {
		t.Errorf("GTC = %v, want gtc", GTC)
	}
}
