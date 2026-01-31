package alpaca

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MockClient is a mock implementation of the Alpaca client for testing
type MockClient struct {
	mu        sync.RWMutex
	orders    map[string]*Order
	positions map[string]*Position
	account   *Account
}

// NewMockClient creates a new mock Alpaca client
func NewMockClient() *MockClient {
	return &MockClient{
		orders:    make(map[string]*Order),
		positions: make(map[string]*Position),
		account: &Account{
			ID:             "mock-account-id",
			AccountNumber:  "MOCK123456",
			Status:         "ACTIVE",
			Currency:       "USD",
			Cash:           "100000.00",
			PortfolioValue: "100000.00",
			BuyingPower:    "100000.00",
			Equity:         "100000.00",
		},
	}
}

// GetAccount returns the mock account
func (c *MockClient) GetAccount(ctx context.Context) (*Account, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.account, nil
}

// CreateOrder creates a mock order
func (c *MockClient) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	orderID := uuid.New().String()
	clientOrderID := req.ClientOrderID
	if clientOrderID == "" {
		clientOrderID = uuid.New().String()
	}

	now := time.Now()
	order := &Order{
		ID:            orderID,
		ClientOrderID: clientOrderID,
		CreatedAt:     now,
		UpdatedAt:     now,
		SubmittedAt:   now,
		Symbol:        req.Symbol,
		AssetClass:    "us_equity",
		Qty:           req.Qty,
		Notional:      req.Notional,
		OrderType:     req.Type,
		Side:          req.Side,
		TimeInForce:   req.TimeInForce,
		LimitPrice:    req.LimitPrice,
		StopPrice:     req.StopPrice,
		Status:        OrderStatusNew,
		ExtendedHours: req.ExtendedHours,
	}

	// Simulate immediate fill for market orders
	if req.Type == Market {
		order.Status = OrderStatusFilled
		order.FilledQty = order.Qty
		if order.Qty == "" && order.Notional != "" {
			order.FilledQty = "1" // Simulate fractional fill
		}
		order.FilledAvgPrice = "150.00" // Mock price
		filledAt := now
		order.FilledAt = &filledAt

		// Update positions for filled orders
		c.updatePosition(order)
	}

	c.orders[orderID] = order
	return order, nil
}

// updatePosition updates the mock position after an order fill
func (c *MockClient) updatePosition(order *Order) {
	pos, exists := c.positions[order.Symbol]
	if !exists {
		pos = &Position{
			AssetID:       order.AssetID,
			Symbol:        order.Symbol,
			Exchange:      "NASDAQ",
			AssetClass:    "us_equity",
			Qty:           "0",
			AvgEntryPrice: "0",
			Side:          "long",
			MarketValue:   "0",
			CostBasis:     "0",
			CurrentPrice:  "150.00",
		}
		c.positions[order.Symbol] = pos
	}

	// Simple position update (not calculating exact values)
	if order.Side == Buy {
		pos.Qty = order.FilledQty
		pos.AvgEntryPrice = order.FilledAvgPrice
		pos.MarketValue = fmt.Sprintf("%.2f", 150.00) // Mock value
	}
}

// GetOrder retrieves a mock order by ID
func (c *MockClient) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	order, exists := c.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}
	return order, nil
}

// GetOrderByClientID retrieves a mock order by client order ID
func (c *MockClient) GetOrderByClientID(ctx context.Context, clientOrderID string) (*Order, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, order := range c.orders {
		if order.ClientOrderID == clientOrderID {
			return order, nil
		}
	}
	return nil, fmt.Errorf("order not found with client ID: %s", clientOrderID)
}

// ListOrders returns all mock orders
func (c *MockClient) ListOrders(ctx context.Context, params *ListOrdersParams) ([]Order, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	orders := make([]Order, 0, len(c.orders))
	for _, order := range c.orders {
		if params != nil && params.Status != "" {
			if params.Status == "open" && order.Status == OrderStatusFilled {
				continue
			}
			if params.Status == "closed" && order.Status != OrderStatusFilled {
				continue
			}
		}
		orders = append(orders, *order)
	}
	return orders, nil
}

// CancelOrder cancels a mock order
func (c *MockClient) CancelOrder(ctx context.Context, orderID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	order, exists := c.orders[orderID]
	if !exists {
		return fmt.Errorf("order not found: %s", orderID)
	}

	if order.Status == OrderStatusFilled {
		return fmt.Errorf("cannot cancel filled order")
	}

	order.Status = OrderStatusCanceled
	now := time.Now()
	order.CanceledAt = &now
	return nil
}

// CancelAllOrders cancels all open mock orders
func (c *MockClient) CancelAllOrders(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for _, order := range c.orders {
		if order.Status != OrderStatusFilled {
			order.Status = OrderStatusCanceled
			order.CanceledAt = &now
		}
	}
	return nil
}

// ReplaceOrder modifies a mock order
func (c *MockClient) ReplaceOrder(ctx context.Context, orderID string, req *ReplaceOrderRequest) (*Order, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	order, exists := c.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	if order.Status == OrderStatusFilled {
		return nil, fmt.Errorf("cannot replace filled order")
	}

	// Create new order with replaced values
	newOrderID := uuid.New().String()
	newOrder := &Order{
		ID:            newOrderID,
		ClientOrderID: req.ClientOrderID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		SubmittedAt:   time.Now(),
		Symbol:        order.Symbol,
		AssetClass:    order.AssetClass,
		Qty:           req.Qty,
		OrderType:     order.OrderType,
		Side:          order.Side,
		TimeInForce:   req.TimeInForce,
		LimitPrice:    req.LimitPrice,
		StopPrice:     req.StopPrice,
		Status:        OrderStatusNew,
		Replaces:      &orderID,
	}

	// Mark old order as replaced
	order.Status = OrderStatusReplaced
	order.ReplacedBy = &newOrderID
	now := time.Now()
	order.ReplacedAt = &now

	c.orders[newOrderID] = newOrder
	return newOrder, nil
}

// ListPositions returns all mock positions
func (c *MockClient) ListPositions(ctx context.Context) ([]Position, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	positions := make([]Position, 0, len(c.positions))
	for _, pos := range c.positions {
		positions = append(positions, *pos)
	}
	return positions, nil
}

// GetPosition retrieves a specific mock position
func (c *MockClient) GetPosition(ctx context.Context, symbol string) (*Position, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	pos, exists := c.positions[symbol]
	if !exists {
		return nil, fmt.Errorf("position not found: %s", symbol)
	}
	return pos, nil
}

// ClosePosition closes a mock position
func (c *MockClient) ClosePosition(ctx context.Context, symbol string, qty string) (*Order, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	pos, exists := c.positions[symbol]
	if !exists {
		return nil, fmt.Errorf("position not found: %s", symbol)
	}

	// Create sell order
	orderID := uuid.New().String()
	now := time.Now()
	order := &Order{
		ID:             orderID,
		ClientOrderID:  uuid.New().String(),
		CreatedAt:      now,
		UpdatedAt:      now,
		SubmittedAt:    now,
		FilledAt:       &now,
		Symbol:         symbol,
		AssetClass:     "us_equity",
		Qty:            pos.Qty,
		FilledQty:      pos.Qty,
		FilledAvgPrice: pos.CurrentPrice,
		OrderType:      Market,
		Side:           Sell,
		TimeInForce:    Day,
		Status:         OrderStatusFilled,
	}

	c.orders[orderID] = order
	delete(c.positions, symbol)

	return order, nil
}

// CloseAllPositions closes all mock positions
func (c *MockClient) CloseAllPositions(ctx context.Context, cancelOrders bool) ([]Order, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	orders := make([]Order, 0)
	now := time.Now()

	for symbol, pos := range c.positions {
		orderID := uuid.New().String()
		order := Order{
			ID:             orderID,
			ClientOrderID:  uuid.New().String(),
			CreatedAt:      now,
			UpdatedAt:      now,
			SubmittedAt:    now,
			FilledAt:       &now,
			Symbol:         symbol,
			AssetClass:     "us_equity",
			Qty:            pos.Qty,
			FilledQty:      pos.Qty,
			FilledAvgPrice: pos.CurrentPrice,
			OrderType:      Market,
			Side:           Sell,
			TimeInForce:    Day,
			Status:         OrderStatusFilled,
		}
		orders = append(orders, order)
		c.orders[orderID] = &order
	}

	c.positions = make(map[string]*Position)
	return orders, nil
}

// GetAsset retrieves mock asset information
func (c *MockClient) GetAsset(ctx context.Context, symbol string) (*Asset, error) {
	return &Asset{
		ID:           "mock-asset-" + symbol,
		Class:        "us_equity",
		Exchange:     "NASDAQ",
		Symbol:       symbol,
		Name:         symbol + " Inc.",
		Status:       "active",
		Tradable:     true,
		Marginable:   true,
		Shortable:    true,
		EasyToBorrow: true,
		Fractionable: true,
	}, nil
}

// ListAssets returns a list of mock assets
func (c *MockClient) ListAssets(ctx context.Context, params *ListAssetsParams) ([]Asset, error) {
	assets := []Asset{
		{ID: "1", Symbol: "AAPL", Name: "Apple Inc.", Class: "us_equity", Exchange: "NASDAQ", Status: "active", Tradable: true, Fractionable: true},
		{ID: "2", Symbol: "GOOGL", Name: "Alphabet Inc.", Class: "us_equity", Exchange: "NASDAQ", Status: "active", Tradable: true, Fractionable: true},
		{ID: "3", Symbol: "MSFT", Name: "Microsoft Corporation", Class: "us_equity", Exchange: "NASDAQ", Status: "active", Tradable: true, Fractionable: true},
		{ID: "4", Symbol: "AMZN", Name: "Amazon.com Inc.", Class: "us_equity", Exchange: "NASDAQ", Status: "active", Tradable: true, Fractionable: true},
		{ID: "5", Symbol: "TSLA", Name: "Tesla Inc.", Class: "us_equity", Exchange: "NASDAQ", Status: "active", Tradable: true, Fractionable: true},
	}
	return assets, nil
}

// GetQuote returns a mock quote
func (c *MockClient) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	return &Quote{
		Symbol:    symbol,
		BidPrice:  149.50,
		BidSize:   100,
		AskPrice:  150.50,
		AskSize:   100,
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// SetAccountCash sets the mock account cash balance (for testing)
func (c *MockClient) SetAccountCash(cash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.account.Cash = cash
	c.account.BuyingPower = cash
}

// AddMockPosition adds a mock position (for testing)
func (c *MockClient) AddMockPosition(symbol, qty, avgPrice string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.positions[symbol] = &Position{
		Symbol:        symbol,
		Qty:           qty,
		AvgEntryPrice: avgPrice,
		CurrentPrice:  avgPrice,
		Side:          "long",
		AssetClass:    "us_equity",
	}
}
