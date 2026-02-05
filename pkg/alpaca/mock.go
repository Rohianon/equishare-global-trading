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

// GetMultiQuotes returns mock quotes for multiple symbols
func (c *MockClient) GetMultiQuotes(ctx context.Context, symbols []string) (map[string]Quote, error) {
	quotes := make(map[string]Quote)
	for _, symbol := range symbols {
		quotes[symbol] = Quote{
			Symbol:    symbol,
			BidPrice:  149.50,
			BidSize:   100,
			AskPrice:  150.50,
			AskSize:   100,
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}
	return quotes, nil
}

// GetBars returns mock historical bars
func (c *MockClient) GetBars(ctx context.Context, symbol string, params *GetBarsParams) ([]Bar, error) {
	limit := 10
	if params != nil && params.Limit > 0 {
		limit = params.Limit
		if limit > 100 {
			limit = 100
		}
	}

	bars := make([]Bar, limit)
	basePrice := 150.0
	now := time.Now()

	for i := 0; i < limit; i++ {
		dayOffset := limit - i - 1
		bars[i] = Bar{
			Timestamp:  now.AddDate(0, 0, -dayOffset),
			Open:       basePrice + float64(i)*0.5,
			High:       basePrice + float64(i)*0.5 + 2.0,
			Low:        basePrice + float64(i)*0.5 - 1.0,
			Close:      basePrice + float64(i)*0.5 + 1.0,
			Volume:     1000000 + uint64(i*10000),
			TradeCount: 5000 + uint64(i*100),
			VWAP:       basePrice + float64(i)*0.5 + 0.5,
		}
	}
	return bars, nil
}

// GetClock returns mock market clock
func (c *MockClient) GetClock(ctx context.Context) (*Clock, error) {
	now := time.Now()
	hour := now.Hour()
	weekday := now.Weekday()

	// Market is open 9:30 AM - 4:00 PM ET, Monday-Friday
	isOpen := weekday >= time.Monday && weekday <= time.Friday && hour >= 9 && hour < 16

	nextOpen := now
	nextClose := now

	if isOpen {
		nextClose = time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, now.Location())
		nextOpen = nextClose.Add(17*time.Hour + 30*time.Minute) // Next day 9:30 AM
	} else {
		// Find next open (simplified)
		nextOpen = time.Date(now.Year(), now.Month(), now.Day()+1, 9, 30, 0, 0, now.Location())
		nextClose = time.Date(now.Year(), now.Month(), now.Day()+1, 16, 0, 0, 0, now.Location())
	}

	return &Clock{
		Timestamp: now,
		IsOpen:    isOpen,
		NextOpen:  nextOpen,
		NextClose: nextClose,
	}, nil
}

// GetCalendar returns mock market calendar
func (c *MockClient) GetCalendar(ctx context.Context, params *GetCalendarParams) ([]CalendarDay, error) {
	calendar := make([]CalendarDay, 30)
	now := time.Now()

	for i := 0; i < 30; i++ {
		day := now.AddDate(0, 0, i)
		// Skip weekends
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}
		calendar[i] = CalendarDay{
			Date:  day.Format("2006-01-02"),
			Open:  "09:30",
			Close: "16:00",
		}
	}

	// Filter out empty entries
	filtered := make([]CalendarDay, 0)
	for _, d := range calendar {
		if d.Date != "" {
			filtered = append(filtered, d)
		}
	}

	return filtered, nil
}

// GetSnapshot returns a mock snapshot for a symbol
func (c *MockClient) GetSnapshot(ctx context.Context, symbol string) (*Snapshot, error) {
	now := time.Now()
	basePrice := 150.0

	return &Snapshot{
		LatestTrade: &Trade{
			Timestamp: now,
			Price:     basePrice,
			Size:      100,
			Exchange:  "NASDAQ",
			ID:        12345,
			Tape:      "C",
		},
		LatestQuote: &Quote{
			Symbol:    symbol,
			BidPrice:  basePrice - 0.50,
			BidSize:   100,
			AskPrice:  basePrice + 0.50,
			AskSize:   100,
			Timestamp: now.Format(time.RFC3339),
		},
		MinuteBar: &Bar{
			Timestamp:  now.Truncate(time.Minute),
			Open:       basePrice - 0.25,
			High:       basePrice + 0.50,
			Low:        basePrice - 0.50,
			Close:      basePrice,
			Volume:     10000,
			TradeCount: 50,
			VWAP:       basePrice,
		},
		DailyBar: &Bar{
			Timestamp:  now.Truncate(24 * time.Hour),
			Open:       basePrice - 2.0,
			High:       basePrice + 3.0,
			Low:        basePrice - 3.0,
			Close:      basePrice,
			Volume:     5000000,
			TradeCount: 25000,
			VWAP:       basePrice - 0.5,
		},
		PrevDailyBar: &Bar{
			Timestamp:  now.AddDate(0, 0, -1).Truncate(24 * time.Hour),
			Open:       basePrice - 3.0,
			High:       basePrice + 1.0,
			Low:        basePrice - 4.0,
			Close:      basePrice - 2.0,
			Volume:     4500000,
			TradeCount: 22000,
			VWAP:       basePrice - 1.5,
		},
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
