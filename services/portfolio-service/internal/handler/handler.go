package handler

import (
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/Rohianon/equishare-global-trading/pkg/alpaca"
	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/services/portfolio-service/internal/repository"
	"github.com/Rohianon/equishare-global-trading/services/portfolio-service/internal/types"
)

// Handler handles portfolio HTTP requests
type Handler struct {
	repo   *repository.Repository
	alpaca alpaca.TradingClient
}

// NewHandler creates a new portfolio handler
func NewHandler(repo *repository.Repository, alpacaClient alpaca.TradingClient) *Handler {
	return &Handler{
		repo:   repo,
		alpaca: alpacaClient,
	}
}

// GetPortfolio retrieves the full portfolio with summary and holdings
// GET /portfolio
func (h *Handler) GetPortfolio(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(types.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID required",
			Code:    401,
		})
	}

	ctx := c.Context()

	// Get holdings from database
	holdings, err := h.repo.ListHoldingsByUser(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("Failed to list holdings")
		// Return empty portfolio, not an error
		holdings = []types.Holding{}
	}

	// Get cash balance
	cashBalance, err := h.repo.GetTotalCashBalance(ctx, userID)
	if err != nil {
		logger.Warn().Err(err).Str("user_id", userID).Msg("Failed to get cash balance")
		cashBalance = 0
	}

	// If no holdings, return empty portfolio
	if len(holdings) == 0 {
		return c.JSON(types.PortfolioResponse{
			Summary: types.PortfolioSummary{
				TotalValue:        cashBalance,
				TotalCostBasis:    0,
				TotalUnrealizedPL: 0,
				TotalUnrealizedPLPct: 0,
				DayChange:         0,
				DayChangePct:      0,
				CashBalance:       cashBalance,
				HoldingsCount:     0,
			},
			Holdings: []types.HoldingWithPrice{},
		})
	}

	// Get current prices for all holdings
	symbols := make([]string, len(holdings))
	for i, h := range holdings {
		symbols[i] = h.Symbol
	}

	quotes, err := h.alpaca.GetMultiQuotes(ctx, symbols)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch quotes")
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch current prices",
			Code:    500,
		})
	}

	// Calculate portfolio metrics
	var totalValue float64
	var totalCostBasis float64
	var totalDayChange float64

	holdingsWithPrice := make([]types.HoldingWithPrice, len(holdings))

	for i, holding := range holdings {
		quote, ok := quotes[holding.Symbol]
		currentPrice := 0.0
		if ok {
			currentPrice = (quote.BidPrice + quote.AskPrice) / 2 // Mid price
		}

		marketValue := holding.Quantity * currentPrice
		unrealizedPL := marketValue - holding.TotalCostBasis
		unrealizedPLPct := 0.0
		if holding.TotalCostBasis > 0 {
			unrealizedPLPct = (unrealizedPL / holding.TotalCostBasis) * 100
		}

		// For day change, we'd need previous close price
		// Using a simple estimate: assume 1% daily movement for mock
		dayChange := marketValue * 0.01
		dayChangePct := 1.0

		totalValue += marketValue
		totalCostBasis += holding.TotalCostBasis
		totalDayChange += dayChange

		holdingsWithPrice[i] = types.HoldingWithPrice{
			Symbol:          holding.Symbol,
			Quantity:        holding.Quantity,
			AvgCostBasis:    holding.AvgCostBasis,
			TotalCostBasis:  holding.TotalCostBasis,
			CurrentPrice:    currentPrice,
			MarketValue:     marketValue,
			UnrealizedPL:    unrealizedPL,
			UnrealizedPLPct: unrealizedPLPct,
			DayChange:       dayChange,
			DayChangePct:    dayChangePct,
		}
	}

	// Calculate allocation percentages
	totalPortfolioValue := totalValue + cashBalance
	for i := range holdingsWithPrice {
		if totalPortfolioValue > 0 {
			holdingsWithPrice[i].AllocationPct = (holdingsWithPrice[i].MarketValue / totalPortfolioValue) * 100
		}
	}

	// Calculate summary
	totalUnrealizedPL := totalValue - totalCostBasis
	totalUnrealizedPLPct := 0.0
	if totalCostBasis > 0 {
		totalUnrealizedPLPct = (totalUnrealizedPL / totalCostBasis) * 100
	}

	dayChangePct := 0.0
	if totalValue > 0 {
		dayChangePct = (totalDayChange / totalValue) * 100
	}

	return c.JSON(types.PortfolioResponse{
		Summary: types.PortfolioSummary{
			TotalValue:           totalPortfolioValue,
			TotalCostBasis:       totalCostBasis,
			TotalUnrealizedPL:    totalUnrealizedPL,
			TotalUnrealizedPLPct: totalUnrealizedPLPct,
			DayChange:            totalDayChange,
			DayChangePct:         dayChangePct,
			CashBalance:          cashBalance,
			HoldingsCount:        len(holdings),
		},
		Holdings: holdingsWithPrice,
	})
}

// GetHoldings retrieves all holdings with current prices
// GET /portfolio/holdings
func (h *Handler) GetHoldings(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(types.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID required",
			Code:    401,
		})
	}

	ctx := c.Context()

	holdings, err := h.repo.ListHoldingsByUser(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("Failed to list holdings")
		holdings = []types.Holding{}
	}

	if len(holdings) == 0 {
		return c.JSON(types.HoldingsResponse{
			Holdings: []types.HoldingWithPrice{},
			Total:    0,
		})
	}

	// Get current prices
	symbols := make([]string, len(holdings))
	for i, h := range holdings {
		symbols[i] = h.Symbol
	}

	quotes, err := h.alpaca.GetMultiQuotes(ctx, symbols)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch quotes")
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch current prices",
			Code:    500,
		})
	}

	// Calculate total value for allocation
	var totalValue float64
	holdingsWithPrice := make([]types.HoldingWithPrice, len(holdings))

	for i, holding := range holdings {
		quote, ok := quotes[holding.Symbol]
		currentPrice := 0.0
		if ok {
			currentPrice = (quote.BidPrice + quote.AskPrice) / 2
		}

		marketValue := holding.Quantity * currentPrice
		unrealizedPL := marketValue - holding.TotalCostBasis
		unrealizedPLPct := 0.0
		if holding.TotalCostBasis > 0 {
			unrealizedPLPct = (unrealizedPL / holding.TotalCostBasis) * 100
		}

		totalValue += marketValue

		holdingsWithPrice[i] = types.HoldingWithPrice{
			Symbol:          holding.Symbol,
			Quantity:        holding.Quantity,
			AvgCostBasis:    holding.AvgCostBasis,
			TotalCostBasis:  holding.TotalCostBasis,
			CurrentPrice:    currentPrice,
			MarketValue:     marketValue,
			UnrealizedPL:    unrealizedPL,
			UnrealizedPLPct: unrealizedPLPct,
			DayChange:       marketValue * 0.01,
			DayChangePct:    1.0,
		}
	}

	// Calculate allocation
	for i := range holdingsWithPrice {
		if totalValue > 0 {
			holdingsWithPrice[i].AllocationPct = (holdingsWithPrice[i].MarketValue / totalValue) * 100
		}
	}

	return c.JSON(types.HoldingsResponse{
		Holdings: holdingsWithPrice,
		Total:    len(holdingsWithPrice),
	})
}

// GetHolding retrieves a specific holding
// GET /portfolio/holdings/:symbol
func (h *Handler) GetHolding(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(types.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID required",
			Code:    401,
		})
	}

	symbol := strings.ToUpper(c.Params("symbol"))
	if symbol == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Error:   "bad_request",
			Message: "Symbol is required",
			Code:    400,
		})
	}

	ctx := c.Context()

	holding, err := h.repo.GetHoldingBySymbol(ctx, userID, symbol)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(types.ErrorResponse{
			Error:   "not_found",
			Message: "Holding not found",
			Code:    404,
		})
	}

	// Get current price
	quote, err := h.alpaca.GetQuote(ctx, symbol)
	if err != nil {
		logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to fetch quote")
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch current price",
			Code:    500,
		})
	}

	currentPrice := (quote.BidPrice + quote.AskPrice) / 2
	marketValue := holding.Quantity * currentPrice
	unrealizedPL := marketValue - holding.TotalCostBasis
	unrealizedPLPct := 0.0
	if holding.TotalCostBasis > 0 {
		unrealizedPLPct = (unrealizedPL / holding.TotalCostBasis) * 100
	}

	return c.JSON(types.HoldingDetailResponse{
		Holding: types.HoldingWithPrice{
			Symbol:          holding.Symbol,
			Quantity:        holding.Quantity,
			AvgCostBasis:    holding.AvgCostBasis,
			TotalCostBasis:  holding.TotalCostBasis,
			CurrentPrice:    currentPrice,
			MarketValue:     marketValue,
			UnrealizedPL:    unrealizedPL,
			UnrealizedPLPct: unrealizedPLPct,
			DayChange:       marketValue * 0.01,
			DayChangePct:    1.0,
		},
	})
}

// GetAllocation retrieves portfolio allocation breakdown
// GET /portfolio/allocation
func (h *Handler) GetAllocation(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(types.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID required",
			Code:    401,
		})
	}

	ctx := c.Context()

	holdings, err := h.repo.ListHoldingsByUser(ctx, userID)
	if err != nil {
		holdings = []types.Holding{}
	}

	cashBalance, err := h.repo.GetTotalCashBalance(ctx, userID)
	if err != nil {
		cashBalance = 0
	}

	if len(holdings) == 0 {
		cashPct := 100.0
		if cashBalance == 0 {
			cashPct = 0
		}
		return c.JSON(types.AllocationResponse{
			Allocations: []types.AllocationItem{},
			CashPct:     cashPct,
		})
	}

	// Get current prices
	symbols := make([]string, len(holdings))
	for i, h := range holdings {
		symbols[i] = h.Symbol
	}

	quotes, err := h.alpaca.GetMultiQuotes(ctx, symbols)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch current prices",
			Code:    500,
		})
	}

	// Calculate allocations
	var totalValue float64
	allocations := make([]types.AllocationItem, len(holdings))

	for i, holding := range holdings {
		quote, ok := quotes[holding.Symbol]
		currentPrice := 0.0
		if ok {
			currentPrice = (quote.BidPrice + quote.AskPrice) / 2
		}

		marketValue := holding.Quantity * currentPrice
		totalValue += marketValue

		allocations[i] = types.AllocationItem{
			Symbol:      holding.Symbol,
			MarketValue: marketValue,
		}
	}

	totalPortfolioValue := totalValue + cashBalance

	// Calculate percentages
	for i := range allocations {
		if totalPortfolioValue > 0 {
			allocations[i].AllocationPct = (allocations[i].MarketValue / totalPortfolioValue) * 100
		}
	}

	// Sort by allocation descending
	sort.Slice(allocations, func(i, j int) bool {
		return allocations[i].AllocationPct > allocations[j].AllocationPct
	})

	cashPct := 0.0
	if totalPortfolioValue > 0 {
		cashPct = (cashBalance / totalPortfolioValue) * 100
	}

	return c.JSON(types.AllocationResponse{
		Allocations: allocations,
		CashPct:     cashPct,
	})
}

// GetPerformance retrieves portfolio performance metrics
// GET /portfolio/performance
func (h *Handler) GetPerformance(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(types.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID required",
			Code:    401,
		})
	}

	ctx := c.Context()

	holdings, err := h.repo.ListHoldingsByUser(ctx, userID)
	if err != nil || len(holdings) == 0 {
		return c.JSON(types.PerformanceResponse{
			TotalReturn:    0,
			TotalReturnPct: 0,
			DayReturn:      0,
			DayReturnPct:   0,
		})
	}

	// Get current prices
	symbols := make([]string, len(holdings))
	for i, h := range holdings {
		symbols[i] = h.Symbol
	}

	quotes, err := h.alpaca.GetMultiQuotes(ctx, symbols)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch current prices",
			Code:    500,
		})
	}

	var totalValue float64
	var totalCostBasis float64
	var bestReturn float64
	var worstReturn float64
	var bestSymbol string
	var worstSymbol string

	for _, holding := range holdings {
		quote, ok := quotes[holding.Symbol]
		currentPrice := 0.0
		if ok {
			currentPrice = (quote.BidPrice + quote.AskPrice) / 2
		}

		marketValue := holding.Quantity * currentPrice
		totalValue += marketValue
		totalCostBasis += holding.TotalCostBasis

		// Track best/worst performers
		returnPct := 0.0
		if holding.TotalCostBasis > 0 {
			returnPct = ((marketValue - holding.TotalCostBasis) / holding.TotalCostBasis) * 100
		}

		if bestSymbol == "" || returnPct > bestReturn {
			bestReturn = returnPct
			bestSymbol = holding.Symbol
		}
		if worstSymbol == "" || returnPct < worstReturn {
			worstReturn = returnPct
			worstSymbol = holding.Symbol
		}
	}

	totalReturn := totalValue - totalCostBasis
	totalReturnPct := 0.0
	if totalCostBasis > 0 {
		totalReturnPct = (totalReturn / totalCostBasis) * 100
	}

	// Day return (simplified - would need previous day values for accuracy)
	dayReturn := totalValue * 0.01
	dayReturnPct := 1.0

	return c.JSON(types.PerformanceResponse{
		TotalReturn:    totalReturn,
		TotalReturnPct: totalReturnPct,
		DayReturn:      dayReturn,
		DayReturnPct:   dayReturnPct,
		BestPerformer:  bestSymbol,
		WorstPerformer: worstSymbol,
	})
}
