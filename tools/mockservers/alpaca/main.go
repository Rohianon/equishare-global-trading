package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/google/uuid"
)

// =============================================================================
// Alpaca Mock Server
// =============================================================================
// This server simulates the Alpaca Trading API for integration testing.
// It supports:
// - Account information
// - Order management (create, list, cancel)
// - Position management
// - Asset information
// - Market data (quotes)
// =============================================================================

type Server struct {
	mu        sync.RWMutex
	account   Account
	orders    map[string]*Order
	positions map[string]*Position
	assets    map[string]*Asset
}

type Account struct {
	ID               string `json:"id"`
	AccountNumber    string `json:"account_number"`
	Status           string `json:"status"`
	Currency         string `json:"currency"`
	Cash             string `json:"cash"`
	PortfolioValue   string `json:"portfolio_value"`
	BuyingPower      string `json:"buying_power"`
	Equity           string `json:"equity"`
	LastEquity       string `json:"last_equity"`
	PatternDayTrader bool   `json:"pattern_day_trader"`
	TradingBlocked   bool   `json:"trading_blocked"`
	TransfersBlocked bool   `json:"transfers_blocked"`
	AccountBlocked   bool   `json:"account_blocked"`
}

type Order struct {
	ID             string     `json:"id"`
	ClientOrderID  string     `json:"client_order_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	SubmittedAt    time.Time  `json:"submitted_at"`
	FilledAt       *time.Time `json:"filled_at"`
	ExpiredAt      *time.Time `json:"expired_at"`
	CanceledAt     *time.Time `json:"canceled_at"`
	FailedAt       *time.Time `json:"failed_at"`
	ReplacedAt     *time.Time `json:"replaced_at"`
	ReplacedBy     *string    `json:"replaced_by"`
	Replaces       *string    `json:"replaces"`
	Symbol         string     `json:"symbol"`
	AssetID        string     `json:"asset_id"`
	AssetClass     string     `json:"asset_class"`
	Notional       string     `json:"notional,omitempty"`
	Qty            string     `json:"qty,omitempty"`
	FilledQty      string     `json:"filled_qty"`
	FilledAvgPrice string     `json:"filled_avg_price,omitempty"`
	OrderType      string     `json:"type"`
	Side           string     `json:"side"`
	TimeInForce    string     `json:"time_in_force"`
	LimitPrice     string     `json:"limit_price,omitempty"`
	StopPrice      string     `json:"stop_price,omitempty"`
	Status         string     `json:"status"`
	ExtendedHours  bool       `json:"extended_hours"`
}

type Position struct {
	AssetID        string `json:"asset_id"`
	Symbol         string `json:"symbol"`
	Exchange       string `json:"exchange"`
	AssetClass     string `json:"asset_class"`
	AvgEntryPrice  string `json:"avg_entry_price"`
	Qty            string `json:"qty"`
	Side           string `json:"side"`
	MarketValue    string `json:"market_value"`
	CostBasis      string `json:"cost_basis"`
	UnrealizedPL   string `json:"unrealized_pl"`
	UnrealizedPLPC string `json:"unrealized_plpc"`
	CurrentPrice   string `json:"current_price"`
	LastdayPrice   string `json:"lastday_price"`
	ChangeToday    string `json:"change_today"`
}

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

func NewServer() *Server {
	s := &Server{
		account: Account{
			ID:             "mock-account-id",
			AccountNumber:  "MOCK123456",
			Status:         "ACTIVE",
			Currency:       "USD",
			Cash:           "100000.00",
			PortfolioValue: "100000.00",
			BuyingPower:    "200000.00",
			Equity:         "100000.00",
			LastEquity:     "99500.00",
		},
		orders:    make(map[string]*Order),
		positions: make(map[string]*Position),
		assets:    make(map[string]*Asset),
	}

	// Initialize default assets
	s.initAssets()
	return s
}

func (s *Server) initAssets() {
	defaultAssets := []Asset{
		{ID: "1", Symbol: "AAPL", Name: "Apple Inc.", Class: "us_equity", Exchange: "NASDAQ", Status: "active", Tradable: true, Marginable: true, Shortable: true, EasyToBorrow: true, Fractionable: true},
		{ID: "2", Symbol: "GOOGL", Name: "Alphabet Inc.", Class: "us_equity", Exchange: "NASDAQ", Status: "active", Tradable: true, Marginable: true, Shortable: true, EasyToBorrow: true, Fractionable: true},
		{ID: "3", Symbol: "MSFT", Name: "Microsoft Corporation", Class: "us_equity", Exchange: "NASDAQ", Status: "active", Tradable: true, Marginable: true, Shortable: true, EasyToBorrow: true, Fractionable: true},
		{ID: "4", Symbol: "AMZN", Name: "Amazon.com Inc.", Class: "us_equity", Exchange: "NASDAQ", Status: "active", Tradable: true, Marginable: true, Shortable: true, EasyToBorrow: true, Fractionable: true},
		{ID: "5", Symbol: "TSLA", Name: "Tesla Inc.", Class: "us_equity", Exchange: "NASDAQ", Status: "active", Tradable: true, Marginable: true, Shortable: true, EasyToBorrow: true, Fractionable: true},
		{ID: "6", Symbol: "META", Name: "Meta Platforms Inc.", Class: "us_equity", Exchange: "NASDAQ", Status: "active", Tradable: true, Marginable: true, Shortable: true, EasyToBorrow: true, Fractionable: true},
		{ID: "7", Symbol: "NVDA", Name: "NVIDIA Corporation", Class: "us_equity", Exchange: "NASDAQ", Status: "active", Tradable: true, Marginable: true, Shortable: true, EasyToBorrow: true, Fractionable: true},
	}

	for _, asset := range defaultAssets {
		s.assets[asset.Symbol] = &asset
	}
}

func main() {
	server := NewServer()

	app := fiber.New(fiber.Config{
		AppName: "Alpaca Mock Server",
	})

	app.Use(logger.New())

	// Authentication middleware
	app.Use(func(c *fiber.Ctx) error {
		if c.Path() == "/health" || c.Path() == "/admin/reset" {
			return c.Next()
		}
		apiKey := c.Get("APCA-API-KEY-ID")
		apiSecret := c.Get("APCA-API-SECRET-KEY")
		if apiKey == "" || apiSecret == "" {
			return c.Status(401).JSON(fiber.Map{"message": "Authentication required"})
		}
		return c.Next()
	})

	// Account
	app.Get("/v2/account", server.getAccount)

	// Orders
	app.Post("/v2/orders", server.createOrder)
	app.Get("/v2/orders", server.listOrders)
	app.Get("/v2/orders/:id", server.getOrder)
	app.Delete("/v2/orders/:id", server.cancelOrder)
	app.Delete("/v2/orders", server.cancelAllOrders)
	app.Patch("/v2/orders/:id", server.replaceOrder)

	// Positions
	app.Get("/v2/positions", server.listPositions)
	app.Get("/v2/positions/:symbol", server.getPosition)
	app.Delete("/v2/positions/:symbol", server.closePosition)
	app.Delete("/v2/positions", server.closeAllPositions)

	// Assets
	app.Get("/v2/assets", server.listAssets)
	app.Get("/v2/assets/:symbol", server.getAsset)

	// Market Data
	app.Get("/v2/stocks/:symbol/quotes/latest", server.getQuote)

	// Admin endpoints
	app.Post("/admin/reset", server.reset)
	app.Post("/admin/set-cash", server.setCash)
	app.Get("/admin/state", server.getState)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy", "service": "alpaca-mock"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8092"
	}

	log.Printf("Alpaca Mock Server starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}

// =============================================================================
// Account
// =============================================================================

func (s *Server) getAccount(c *fiber.Ctx) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return c.JSON(s.account)
}

// =============================================================================
// Orders
// =============================================================================

type CreateOrderRequest struct {
	Symbol        string `json:"symbol"`
	Qty           string `json:"qty,omitempty"`
	Notional      string `json:"notional,omitempty"`
	Side          string `json:"side"`
	Type          string `json:"type"`
	TimeInForce   string `json:"time_in_force"`
	LimitPrice    string `json:"limit_price,omitempty"`
	StopPrice     string `json:"stop_price,omitempty"`
	ClientOrderID string `json:"client_order_id,omitempty"`
	ExtendedHours bool   `json:"extended_hours,omitempty"`
}

func (s *Server) createOrder(c *fiber.Ctx) error {
	var req CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid request body"})
	}

	if req.Symbol == "" || req.Side == "" || req.Type == "" {
		return c.Status(422).JSON(fiber.Map{"message": "Missing required fields"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if asset exists
	asset, exists := s.assets[req.Symbol]
	if !exists {
		return c.Status(422).JSON(fiber.Map{"message": "Asset not found"})
	}

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
		AssetID:       asset.ID,
		AssetClass:    "us_equity",
		Qty:           req.Qty,
		Notional:      req.Notional,
		FilledQty:     "0",
		OrderType:     req.Type,
		Side:          req.Side,
		TimeInForce:   req.TimeInForce,
		LimitPrice:    req.LimitPrice,
		StopPrice:     req.StopPrice,
		Status:        "new",
		ExtendedHours: req.ExtendedHours,
	}

	// Simulate immediate fill for market orders
	if req.Type == "market" {
		order.Status = "filled"
		order.FilledQty = order.Qty
		if order.Qty == "" && order.Notional != "" {
			order.FilledQty = "1"
		}
		order.FilledAvgPrice = s.getMockPrice(req.Symbol)
		filledAt := now
		order.FilledAt = &filledAt

		// Update position
		s.updatePosition(order)
	}

	s.orders[orderID] = order
	return c.Status(201).JSON(order)
}

func (s *Server) getMockPrice(symbol string) string {
	prices := map[string]string{
		"AAPL":  "175.50",
		"GOOGL": "140.25",
		"MSFT":  "380.00",
		"AMZN":  "155.75",
		"TSLA":  "245.00",
		"META":  "350.00",
		"NVDA":  "480.00",
	}
	if price, ok := prices[symbol]; ok {
		return price
	}
	return "100.00"
}

func (s *Server) updatePosition(order *Order) {
	pos, exists := s.positions[order.Symbol]
	if !exists {
		pos = &Position{
			AssetID:    order.AssetID,
			Symbol:     order.Symbol,
			Exchange:   "NASDAQ",
			AssetClass: "us_equity",
			Qty:        "0",
			Side:       "long",
		}
		s.positions[order.Symbol] = pos
	}

	if order.Side == "buy" {
		qty, _ := strconv.ParseFloat(pos.Qty, 64)
		orderQty, _ := strconv.ParseFloat(order.FilledQty, 64)
		pos.Qty = fmt.Sprintf("%.4f", qty+orderQty)
		pos.AvgEntryPrice = order.FilledAvgPrice
	} else {
		qty, _ := strconv.ParseFloat(pos.Qty, 64)
		orderQty, _ := strconv.ParseFloat(order.FilledQty, 64)
		newQty := qty - orderQty
		if newQty <= 0 {
			delete(s.positions, order.Symbol)
			return
		}
		pos.Qty = fmt.Sprintf("%.4f", newQty)
	}

	price, _ := strconv.ParseFloat(order.FilledAvgPrice, 64)
	qty, _ := strconv.ParseFloat(pos.Qty, 64)
	pos.MarketValue = fmt.Sprintf("%.2f", price*qty)
	pos.CostBasis = pos.MarketValue
	pos.CurrentPrice = order.FilledAvgPrice
	pos.UnrealizedPL = "0.00"
	pos.UnrealizedPLPC = "0.0000"
}

func (s *Server) listOrders(c *fiber.Ctx) error {
	status := c.Query("status", "all")

	s.mu.RLock()
	defer s.mu.RUnlock()

	orders := make([]Order, 0)
	for _, order := range s.orders {
		if status == "all" ||
			(status == "open" && (order.Status == "new" || order.Status == "accepted" || order.Status == "pending_new")) ||
			(status == "closed" && (order.Status == "filled" || order.Status == "canceled" || order.Status == "expired")) {
			orders = append(orders, *order)
		}
	}

	return c.JSON(orders)
}

func (s *Server) getOrder(c *fiber.Ctx) error {
	id := c.Params("id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	order, exists := s.orders[id]
	if !exists {
		// Check by client order ID
		for _, o := range s.orders {
			if o.ClientOrderID == id {
				return c.JSON(o)
			}
		}
		return c.Status(404).JSON(fiber.Map{"message": "Order not found"})
	}

	return c.JSON(order)
}

func (s *Server) cancelOrder(c *fiber.Ctx) error {
	id := c.Params("id")

	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.orders[id]
	if !exists {
		return c.Status(404).JSON(fiber.Map{"message": "Order not found"})
	}

	if order.Status == "filled" {
		return c.Status(422).JSON(fiber.Map{"message": "Cannot cancel filled order"})
	}

	now := time.Now()
	order.Status = "canceled"
	order.CanceledAt = &now

	return c.SendStatus(204)
}

func (s *Server) cancelAllOrders(c *fiber.Ctx) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	canceled := make([]fiber.Map, 0)

	for _, order := range s.orders {
		if order.Status != "filled" && order.Status != "canceled" {
			order.Status = "canceled"
			order.CanceledAt = &now
			canceled = append(canceled, fiber.Map{
				"id":     order.ID,
				"status": 200,
			})
		}
	}

	return c.JSON(canceled)
}

func (s *Server) replaceOrder(c *fiber.Ctx) error {
	id := c.Params("id")

	var req struct {
		Qty           string `json:"qty,omitempty"`
		TimeInForce   string `json:"time_in_force,omitempty"`
		LimitPrice    string `json:"limit_price,omitempty"`
		StopPrice     string `json:"stop_price,omitempty"`
		ClientOrderID string `json:"client_order_id,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid request body"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.orders[id]
	if !exists {
		return c.Status(404).JSON(fiber.Map{"message": "Order not found"})
	}

	if order.Status == "filled" {
		return c.Status(422).JSON(fiber.Map{"message": "Cannot replace filled order"})
	}

	// Create replacement order
	newOrderID := uuid.New().String()
	now := time.Now()

	newOrder := &Order{
		ID:            newOrderID,
		ClientOrderID: req.ClientOrderID,
		CreatedAt:     now,
		UpdatedAt:     now,
		SubmittedAt:   now,
		Symbol:        order.Symbol,
		AssetID:       order.AssetID,
		AssetClass:    order.AssetClass,
		Qty:           req.Qty,
		FilledQty:     "0",
		OrderType:     order.OrderType,
		Side:          order.Side,
		TimeInForce:   req.TimeInForce,
		LimitPrice:    req.LimitPrice,
		StopPrice:     req.StopPrice,
		Status:        "new",
		Replaces:      &id,
	}

	// Mark old order as replaced
	order.Status = "replaced"
	order.ReplacedBy = &newOrderID
	order.ReplacedAt = &now

	s.orders[newOrderID] = newOrder

	return c.JSON(newOrder)
}

// =============================================================================
// Positions
// =============================================================================

func (s *Server) listPositions(c *fiber.Ctx) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	positions := make([]Position, 0, len(s.positions))
	for _, pos := range s.positions {
		positions = append(positions, *pos)
	}

	return c.JSON(positions)
}

func (s *Server) getPosition(c *fiber.Ctx) error {
	symbol := c.Params("symbol")

	s.mu.RLock()
	defer s.mu.RUnlock()

	pos, exists := s.positions[symbol]
	if !exists {
		return c.Status(404).JSON(fiber.Map{"message": "Position not found"})
	}

	return c.JSON(pos)
}

func (s *Server) closePosition(c *fiber.Ctx) error {
	symbol := c.Params("symbol")

	s.mu.Lock()
	defer s.mu.Unlock()

	pos, exists := s.positions[symbol]
	if !exists {
		return c.Status(404).JSON(fiber.Map{"message": "Position not found"})
	}

	// Create closing order
	now := time.Now()
	order := &Order{
		ID:             uuid.New().String(),
		ClientOrderID:  uuid.New().String(),
		CreatedAt:      now,
		UpdatedAt:      now,
		SubmittedAt:    now,
		FilledAt:       &now,
		Symbol:         symbol,
		AssetClass:     "us_equity",
		Qty:            pos.Qty,
		FilledQty:      pos.Qty,
		FilledAvgPrice: s.getMockPrice(symbol),
		OrderType:      "market",
		Side:           "sell",
		TimeInForce:    "day",
		Status:         "filled",
	}

	s.orders[order.ID] = order
	delete(s.positions, symbol)

	return c.JSON(order)
}

func (s *Server) closeAllPositions(c *fiber.Ctx) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	orders := make([]Order, 0)
	now := time.Now()

	for symbol, pos := range s.positions {
		order := Order{
			ID:             uuid.New().String(),
			ClientOrderID:  uuid.New().String(),
			CreatedAt:      now,
			UpdatedAt:      now,
			SubmittedAt:    now,
			FilledAt:       &now,
			Symbol:         symbol,
			AssetClass:     "us_equity",
			Qty:            pos.Qty,
			FilledQty:      pos.Qty,
			FilledAvgPrice: s.getMockPrice(symbol),
			OrderType:      "market",
			Side:           "sell",
			TimeInForce:    "day",
			Status:         "filled",
		}
		orders = append(orders, order)
		s.orders[order.ID] = &order
	}

	s.positions = make(map[string]*Position)

	return c.JSON(orders)
}

// =============================================================================
// Assets
// =============================================================================

func (s *Server) listAssets(c *fiber.Ctx) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	assets := make([]Asset, 0, len(s.assets))
	for _, asset := range s.assets {
		assets = append(assets, *asset)
	}

	return c.JSON(assets)
}

func (s *Server) getAsset(c *fiber.Ctx) error {
	symbol := c.Params("symbol")

	s.mu.RLock()
	defer s.mu.RUnlock()

	asset, exists := s.assets[symbol]
	if !exists {
		return c.Status(404).JSON(fiber.Map{"message": "Asset not found"})
	}

	return c.JSON(asset)
}

// =============================================================================
// Market Data
// =============================================================================

func (s *Server) getQuote(c *fiber.Ctx) error {
	symbol := c.Params("symbol")

	price := s.getMockPrice(symbol)
	priceFloat, _ := strconv.ParseFloat(price, 64)

	return c.JSON(fiber.Map{
		"quote": fiber.Map{
			"ap": priceFloat + 0.01,
			"as": 100,
			"ax": "Q",
			"bp": priceFloat - 0.01,
			"bs": 100,
			"bx": "Q",
			"c":  []string{"R"},
			"t":  time.Now().Format(time.RFC3339Nano),
			"z":  "C",
		},
	})
}

// =============================================================================
// Admin
// =============================================================================

func (s *Server) reset(c *fiber.Ctx) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.orders = make(map[string]*Order)
	s.positions = make(map[string]*Position)
	s.account.Cash = "100000.00"
	s.account.BuyingPower = "200000.00"

	return c.JSON(fiber.Map{"status": "reset complete"})
}

func (s *Server) setCash(c *fiber.Ctx) error {
	var req struct {
		Cash string `json:"cash"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid request"})
	}

	s.mu.Lock()
	s.account.Cash = req.Cash
	s.account.BuyingPower = req.Cash
	s.mu.Unlock()

	return c.JSON(fiber.Map{"status": "cash updated", "cash": req.Cash})
}

func (s *Server) getState(c *fiber.Ctx) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return c.JSON(fiber.Map{
		"account":   s.account,
		"orders":    s.orders,
		"positions": s.positions,
	})
}
