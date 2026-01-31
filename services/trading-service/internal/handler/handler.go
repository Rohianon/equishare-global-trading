package handler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/Rohianon/equishare-global-trading/pkg/alpaca"
	apperrors "github.com/Rohianon/equishare-global-trading/pkg/errors"
	"github.com/Rohianon/equishare-global-trading/pkg/events"
	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/services/trading-service/internal/repository"
	"github.com/Rohianon/equishare-global-trading/services/trading-service/internal/types"
)

// Handler handles trading HTTP requests
type Handler struct {
	userRepo    *repository.UserRepository
	walletRepo  *repository.WalletRepository
	orderRepo   *repository.OrderRepository
	holdingRepo *repository.HoldingRepository
	alpaca      alpaca.TradingClient
	publisher   events.Publisher
}

// New creates a new trading handler
func New(
	userRepo *repository.UserRepository,
	walletRepo *repository.WalletRepository,
	orderRepo *repository.OrderRepository,
	holdingRepo *repository.HoldingRepository,
	alpacaClient alpaca.TradingClient,
	publisher events.Publisher,
) *Handler {
	return &Handler{
		userRepo:    userRepo,
		walletRepo:  walletRepo,
		orderRepo:   orderRepo,
		holdingRepo: holdingRepo,
		alpaca:      alpacaClient,
		publisher:   publisher,
	}
}

// PlaceOrder handles order placement
func (h *Handler) PlaceOrder(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var req types.PlaceOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.ErrValidation.WithDetails("Invalid request body")
	}

	if req.Symbol == "" {
		return apperrors.ErrValidation.WithDetails("Symbol is required")
	}
	if req.Side != "buy" && req.Side != "sell" {
		return apperrors.ErrValidation.WithDetails("Side must be 'buy' or 'sell'")
	}
	if req.Amount <= 0 && req.Qty <= 0 {
		return apperrors.ErrValidation.WithDetails("Amount or qty is required")
	}

	ctx := c.Context()

	// Verify user exists and is active
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user")
		return apperrors.ErrInternal
	}
	if !user.IsActive {
		return apperrors.ErrForbidden.WithDetails("Account is deactivated")
	}

	// Get USD wallet for balance checks
	wallet, err := h.walletRepo.GetByUserAndCurrency(ctx, userID, "USD")
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get wallet")
		return apperrors.ErrInternal
	}

	// For buy orders, check and lock funds
	if req.Side == "buy" {
		amount := req.Amount
		if amount <= 0 {
			// If qty specified, estimate the amount (we'll use actual at fill)
			quote, err := h.alpaca.GetQuote(ctx, req.Symbol)
			if err != nil {
				return apperrors.ErrServiceUnavailable.WithDetails("Failed to get quote")
			}
			amount = req.Qty * quote.AskPrice * 1.01 // Add 1% buffer
		}

		if wallet.AvailableBalance() < amount {
			return apperrors.ErrValidation.WithDetails("Insufficient balance")
		}

		if err := h.walletRepo.Lock(ctx, wallet.ID, amount); err != nil {
			return apperrors.ErrInternal.WithDetails("Failed to lock funds")
		}
	}

	// For sell orders, check holdings
	if req.Side == "sell" {
		if req.Qty <= 0 {
			return apperrors.ErrValidation.WithDetails("Qty is required for sell orders")
		}

		hasSufficient, err := h.holdingRepo.HasSufficientQty(ctx, userID, req.Symbol, req.Qty)
		if err != nil || !hasSufficient {
			return apperrors.ErrValidation.WithDetails("Insufficient shares to sell")
		}
	}

	// Submit order to Alpaca
	clientOrderID := uuid.New().String()
	alpacaReq := &alpaca.CreateOrderRequest{
		Symbol:        req.Symbol,
		Side:          alpaca.OrderSide(req.Side),
		Type:          alpaca.Market,
		TimeInForce:   alpaca.Day,
		ClientOrderID: clientOrderID,
	}

	if req.Amount > 0 {
		alpacaReq.Notional = fmt.Sprintf("%.2f", req.Amount)
	} else {
		alpacaReq.Qty = fmt.Sprintf("%.6f", req.Qty)
	}

	alpacaOrder, err := h.alpaca.CreateOrder(ctx, alpacaReq)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Str("symbol", req.Symbol).Msg("Failed to create Alpaca order")
		// Unlock funds on failure
		if req.Side == "buy" {
			h.walletRepo.Unlock(ctx, wallet.ID, req.Amount)
		}
		return apperrors.ErrServiceUnavailable.WithDetails("Failed to place order")
	}

	// Save order to database
	order := &types.Order{
		UserID:        userID,
		AlpacaOrderID: alpacaOrder.ID,
		Symbol:        req.Symbol,
		Side:          req.Side,
		Type:          "market",
		Amount:        req.Amount,
		Qty:           req.Qty,
		Status:        "pending",
		Source:        req.Source,
	}
	if order.Source == "" {
		order.Source = "api"
	}

	if err := h.orderRepo.Create(ctx, order); err != nil {
		logger.Error().Err(err).Msg("Failed to save order to database")
		// Order was placed with Alpaca, log but continue
	}

	// Publish order created event
	if h.publisher != nil {
		h.publisher.Publish(ctx, events.TopicOrderCreated, &events.Event{
			Type:   "order.created",
			Source: "trading-service",
			Data: map[string]any{
				"order_id":       order.ID,
				"user_id":        userID,
				"symbol":         req.Symbol,
				"side":           req.Side,
				"amount":         req.Amount,
				"alpaca_order_id": alpacaOrder.ID,
			},
		})
	}

	logger.Info().
		Str("user_id", userID).
		Str("order_id", order.ID).
		Str("symbol", req.Symbol).
		Str("side", req.Side).
		Msg("Order placed successfully")

	return c.Status(fiber.StatusCreated).JSON(types.PlaceOrderResponse{
		OrderID:       order.ID,
		AlpacaOrderID: alpacaOrder.ID,
		Symbol:        req.Symbol,
		Side:          req.Side,
		Amount:        req.Amount,
		Status:        string(alpacaOrder.Status),
		Message:       "Order placed successfully",
	})
}

// CancelOrder cancels an open order
func (h *Handler) CancelOrder(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	orderID := c.Params("id")

	ctx := c.Context()

	order, err := h.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return apperrors.ErrNotFound.WithDetails("Order not found")
	}

	if order.UserID != userID {
		return apperrors.ErrForbidden.WithDetails("Not your order")
	}

	if order.Status != "pending" && order.Status != "new" {
		return apperrors.ErrValidation.WithDetails("Order cannot be canceled")
	}

	if err := h.alpaca.CancelOrder(ctx, order.AlpacaOrderID); err != nil {
		logger.Error().Err(err).Str("order_id", orderID).Msg("Failed to cancel Alpaca order")
		return apperrors.ErrServiceUnavailable.WithDetails("Failed to cancel order")
	}

	if err := h.orderRepo.UpdateStatus(ctx, orderID, "canceled"); err != nil {
		logger.Error().Err(err).Msg("Failed to update order status")
	}

	// Unlock funds for buy orders
	if order.Side == "buy" && order.Amount > 0 {
		wallet, _ := h.walletRepo.GetByUserAndCurrency(ctx, userID, "USD")
		if wallet != nil {
			h.walletRepo.Unlock(ctx, wallet.ID, order.Amount)
		}
	}

	logger.Info().Str("order_id", orderID).Msg("Order canceled")

	return c.JSON(types.CancelOrderResponse{
		OrderID: orderID,
		Status:  "canceled",
		Message: "Order canceled successfully",
	})
}

// GetOrder retrieves an order by ID
func (h *Handler) GetOrder(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	orderID := c.Params("id")

	order, err := h.orderRepo.GetByID(c.Context(), orderID)
	if err != nil {
		return apperrors.ErrNotFound.WithDetails("Order not found")
	}

	if order.UserID != userID {
		return apperrors.ErrForbidden.WithDetails("Not your order")
	}

	return c.JSON(order)
}

// ListOrders retrieves orders for the user
func (h *Handler) ListOrders(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	status := c.Query("status")
	limit := c.QueryInt("limit", 50)

	orders, err := h.orderRepo.ListByUser(c.Context(), userID, status, limit)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list orders")
		return apperrors.ErrInternal
	}

	return c.JSON(fiber.Map{
		"orders": orders,
		"count":  len(orders),
	})
}

// GetPortfolio retrieves the user's portfolio
func (h *Handler) GetPortfolio(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := c.Context()

	holdings, err := h.holdingRepo.ListByUser(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list holdings")
		return apperrors.ErrInternal
	}

	// Get current prices and calculate P&L
	var totalValue, totalPL float64
	for i := range holdings {
		quote, err := h.alpaca.GetQuote(ctx, holdings[i].Symbol)
		if err == nil {
			midPrice := (quote.BidPrice + quote.AskPrice) / 2
			holdings[i].CurrentPrice = midPrice
			holdings[i].MarketValue = holdings[i].Qty * midPrice
			holdings[i].UnrealizedPL = (midPrice - holdings[i].AvgEntryPrice) * holdings[i].Qty
			if holdings[i].AvgEntryPrice > 0 {
				holdings[i].UnrealizedPLPct = (midPrice/holdings[i].AvgEntryPrice - 1) * 100
			}
		}
		totalValue += holdings[i].MarketValue
		totalPL += holdings[i].UnrealizedPL
	}

	// Get cash balance
	wallet, _ := h.walletRepo.GetByUserAndCurrency(ctx, userID, "USD")
	cashUSD := 0.0
	if wallet != nil {
		cashUSD = wallet.Balance
	}

	costBasis := totalValue - totalPL
	totalPLPct := 0.0
	if costBasis > 0 {
		totalPLPct = (totalPL / costBasis) * 100
	}

	return c.JSON(types.Portfolio{
		Holdings:   holdings,
		TotalValue: totalValue,
		TotalPL:    totalPL,
		TotalPLPct: totalPLPct,
		CashUSD:    cashUSD,
	})
}

// GetQuote retrieves a stock quote
func (h *Handler) GetQuote(c *fiber.Ctx) error {
	symbol := c.Params("symbol")
	if symbol == "" {
		return apperrors.ErrValidation.WithDetails("Symbol is required")
	}

	quote, err := h.alpaca.GetQuote(c.Context(), symbol)
	if err != nil {
		return apperrors.ErrNotFound.WithDetails("Quote not found")
	}

	lastPrice := (quote.BidPrice + quote.AskPrice) / 2

	return c.JSON(types.QuoteResponse{
		Symbol:    symbol,
		BidPrice:  quote.BidPrice,
		AskPrice:  quote.AskPrice,
		LastPrice: lastPrice,
	})
}

// SearchAssets searches for tradeable assets
func (h *Handler) SearchAssets(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return apperrors.ErrValidation.WithDetails("Search query is required")
	}

	assets, err := h.alpaca.ListAssets(c.Context(), &alpaca.ListAssetsParams{
		Status:     "active",
		AssetClass: "us_equity",
	})
	if err != nil {
		return apperrors.ErrServiceUnavailable.WithDetails("Failed to search assets")
	}

	// Filter by query (simple substring match)
	var results []types.AssetInfo
	for _, a := range assets {
		if containsIgnoreCase(a.Symbol, query) || containsIgnoreCase(a.Name, query) {
			results = append(results, types.AssetInfo{
				Symbol:       a.Symbol,
				Name:         a.Name,
				Exchange:     a.Exchange,
				Tradable:     a.Tradable,
				Fractionable: a.Fractionable,
			})
			if len(results) >= 20 {
				break
			}
		}
	}

	return c.JSON(types.SearchResponse{Assets: results})
}

// AlpacaWebhook handles order updates from Alpaca
func (h *Handler) AlpacaWebhook(c *fiber.Ctx) error {
	var event types.AlpacaWebhookEvent
	if err := c.BodyParser(&event); err != nil {
		logger.Error().Err(err).Msg("Failed to parse Alpaca webhook")
		return c.SendStatus(fiber.StatusOK)
	}

	ctx := c.Context()

	logger.Info().
		Str("event", event.Event).
		Str("order_id", event.Order.ID).
		Str("status", event.Order.Status).
		Msg("Received Alpaca webhook")

	order, err := h.orderRepo.GetByAlpacaOrderID(ctx, event.Order.ID)
	if err != nil {
		logger.Warn().Str("alpaca_order_id", event.Order.ID).Msg("Order not found in database")
		return c.SendStatus(fiber.StatusOK)
	}

	switch event.Event {
	case "fill":
		h.handleOrderFill(ctx, order, &event.Order)
	case "partial_fill":
		h.handlePartialFill(ctx, order, &event.Order)
	case "canceled":
		h.handleOrderCanceled(ctx, order)
	case "rejected":
		h.handleOrderRejected(ctx, order, "Order rejected by exchange")
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) handleOrderFill(ctx context.Context, order *types.Order, alpacaOrder *types.AlpacaOrderUpdate) {
	filledQty, _ := strconv.ParseFloat(alpacaOrder.FilledQty, 64)
	filledAvgPrice, _ := strconv.ParseFloat(alpacaOrder.FilledAvgPrice, 64)

	if err := h.orderRepo.UpdateFill(ctx, order.AlpacaOrderID, filledQty, filledAvgPrice, "filled"); err != nil {
		logger.Error().Err(err).Msg("Failed to update order fill")
	}

	// Update holdings
	if order.Side == "buy" {
		h.holdingRepo.Upsert(ctx, order.UserID, order.Symbol, filledQty, filledAvgPrice)

		// Debit wallet
		wallet, _ := h.walletRepo.GetByUserAndCurrency(ctx, order.UserID, "USD")
		if wallet != nil {
			totalCost := filledQty * filledAvgPrice
			h.walletRepo.DebitLocked(ctx, wallet.ID, totalCost)
			// Unlock any excess that was locked
			if order.Amount > totalCost {
				h.walletRepo.Unlock(ctx, wallet.ID, order.Amount-totalCost)
			}
		}
	} else {
		// Sell order - reduce holdings and credit wallet
		h.holdingRepo.ReduceQty(ctx, order.UserID, order.Symbol, filledQty)

		wallet, _ := h.walletRepo.GetByUserAndCurrency(ctx, order.UserID, "USD")
		if wallet != nil {
			proceeds := filledQty * filledAvgPrice
			h.walletRepo.Credit(ctx, wallet.ID, proceeds)
		}
	}

	// Publish event
	if h.publisher != nil {
		h.publisher.Publish(ctx, events.TopicOrderFilled, &events.Event{
			Type:   "order.filled",
			Source: "trading-service",
			Data: map[string]any{
				"order_id":         order.ID,
				"user_id":          order.UserID,
				"symbol":           order.Symbol,
				"side":             order.Side,
				"filled_qty":       filledQty,
				"filled_avg_price": filledAvgPrice,
			},
		})
	}

	logger.Info().
		Str("order_id", order.ID).
		Float64("filled_qty", filledQty).
		Float64("filled_avg_price", filledAvgPrice).
		Msg("Order filled")
}

func (h *Handler) handlePartialFill(ctx context.Context, order *types.Order, alpacaOrder *types.AlpacaOrderUpdate) {
	filledQty, _ := strconv.ParseFloat(alpacaOrder.FilledQty, 64)
	filledAvgPrice, _ := strconv.ParseFloat(alpacaOrder.FilledAvgPrice, 64)

	if err := h.orderRepo.UpdateFill(ctx, order.AlpacaOrderID, filledQty, filledAvgPrice, "partial_fill"); err != nil {
		logger.Error().Err(err).Msg("Failed to update partial fill")
	}

	logger.Info().
		Str("order_id", order.ID).
		Float64("filled_qty", filledQty).
		Msg("Order partially filled")
}

func (h *Handler) handleOrderCanceled(ctx context.Context, order *types.Order) {
	if err := h.orderRepo.UpdateCanceled(ctx, order.AlpacaOrderID); err != nil {
		logger.Error().Err(err).Msg("Failed to update order canceled")
	}

	// Unlock funds for buy orders
	if order.Side == "buy" && order.Amount > 0 {
		wallet, _ := h.walletRepo.GetByUserAndCurrency(ctx, order.UserID, "USD")
		if wallet != nil {
			h.walletRepo.Unlock(ctx, wallet.ID, order.Amount)
		}
	}

	logger.Info().Str("order_id", order.ID).Msg("Order canceled via webhook")
}

func (h *Handler) handleOrderRejected(ctx context.Context, order *types.Order, reason string) {
	if err := h.orderRepo.UpdateFailed(ctx, order.AlpacaOrderID, reason); err != nil {
		logger.Error().Err(err).Msg("Failed to update order rejected")
	}

	// Unlock funds for buy orders
	if order.Side == "buy" && order.Amount > 0 {
		wallet, _ := h.walletRepo.GetByUserAndCurrency(ctx, order.UserID, "USD")
		if wallet != nil {
			h.walletRepo.Unlock(ctx, wallet.ID, order.Amount)
		}
	}

	logger.Info().Str("order_id", order.ID).Str("reason", reason).Msg("Order rejected")
}

func containsIgnoreCase(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if equalIgnoreCase(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
