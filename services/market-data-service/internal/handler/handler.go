package handler

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Rohianon/equishare-global-trading/pkg/alpaca"
	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/services/market-data-service/internal/types"
)

// Handler handles market data HTTP requests
type Handler struct {
	alpaca alpaca.TradingClient
}

// NewHandler creates a new market data handler
func NewHandler(alpacaClient alpaca.TradingClient) *Handler {
	return &Handler{
		alpaca: alpacaClient,
	}
}

// GetQuote retrieves the latest quote for a symbol
// GET /quotes/:symbol
func (h *Handler) GetQuote(c *fiber.Ctx) error {
	symbol := strings.ToUpper(c.Params("symbol"))
	if symbol == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Error:   "bad_request",
			Message: "Symbol is required",
			Code:    400,
		})
	}

	ctx := c.Context()

	quote, err := h.alpaca.GetQuote(ctx, symbol)
	if err != nil {
		logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to fetch quote")
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch quote",
			Code:    500,
		})
	}

	midPrice := (quote.BidPrice + quote.AskPrice) / 2
	spread := quote.AskPrice - quote.BidPrice

	return c.JSON(types.QuoteResponse{
		Symbol:    symbol,
		BidPrice:  quote.BidPrice,
		BidSize:   quote.BidSize,
		AskPrice:  quote.AskPrice,
		AskSize:   quote.AskSize,
		MidPrice:  midPrice,
		Spread:    spread,
		Timestamp: quote.Timestamp,
	})
}

// GetMultiQuotes retrieves quotes for multiple symbols
// GET /quotes?symbols=AAPL,GOOGL,MSFT
func (h *Handler) GetMultiQuotes(c *fiber.Ctx) error {
	symbolsParam := c.Query("symbols")
	if symbolsParam == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Error:   "bad_request",
			Message: "Symbols query parameter is required",
			Code:    400,
		})
	}

	symbols := strings.Split(strings.ToUpper(symbolsParam), ",")
	if len(symbols) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Error:   "bad_request",
			Message: "Maximum 100 symbols allowed",
			Code:    400,
		})
	}

	// Clean up symbols
	cleanSymbols := make([]string, 0, len(symbols))
	for _, s := range symbols {
		s = strings.TrimSpace(s)
		if s != "" {
			cleanSymbols = append(cleanSymbols, s)
		}
	}

	ctx := c.Context()

	quotes, err := h.alpaca.GetMultiQuotes(ctx, cleanSymbols)
	if err != nil {
		logger.Error().Err(err).Strs("symbols", cleanSymbols).Msg("Failed to fetch multi quotes")
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch quotes",
			Code:    500,
		})
	}

	response := types.MultiQuoteResponse{
		Quotes: make(map[string]types.QuoteResponse),
	}

	for symbol, q := range quotes {
		midPrice := (q.BidPrice + q.AskPrice) / 2
		spread := q.AskPrice - q.BidPrice
		response.Quotes[symbol] = types.QuoteResponse{
			Symbol:    symbol,
			BidPrice:  q.BidPrice,
			BidSize:   q.BidSize,
			AskPrice:  q.AskPrice,
			AskSize:   q.AskSize,
			MidPrice:  midPrice,
			Spread:    spread,
			Timestamp: q.Timestamp,
		}
	}

	return c.JSON(response)
}

// GetBars retrieves historical OHLCV bars for a symbol
// GET /bars/:symbol?timeframe=1Day&start=2024-01-01&end=2024-12-31&limit=100
func (h *Handler) GetBars(c *fiber.Ctx) error {
	symbol := strings.ToUpper(c.Params("symbol"))
	if symbol == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Error:   "bad_request",
			Message: "Symbol is required",
			Code:    400,
		})
	}

	timeframe := c.Query("timeframe", "1Day")
	startStr := c.Query("start")
	endStr := c.Query("end")
	limit := c.QueryInt("limit", 100)

	if limit > 10000 {
		limit = 10000
	}

	params := &alpaca.GetBarsParams{
		Timeframe: timeframe,
		Limit:     limit,
	}

	if startStr != "" {
		start, err := parseTime(startStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
				Error:   "bad_request",
				Message: "Invalid start date format. Use RFC3339 or YYYY-MM-DD",
				Code:    400,
			})
		}
		params.Start = start
	}

	if endStr != "" {
		end, err := parseTime(endStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
				Error:   "bad_request",
				Message: "Invalid end date format. Use RFC3339 or YYYY-MM-DD",
				Code:    400,
			})
		}
		params.End = end
	}

	ctx := c.Context()

	bars, err := h.alpaca.GetBars(ctx, symbol, params)
	if err != nil {
		logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to fetch bars")
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch historical data",
			Code:    500,
		})
	}

	response := types.BarsResponse{
		Symbol: symbol,
		Bars:   make([]types.BarResponse, len(bars)),
	}

	for i, bar := range bars {
		response.Bars[i] = types.BarResponse{
			Timestamp:  bar.Timestamp,
			Open:       bar.Open,
			High:       bar.High,
			Low:        bar.Low,
			Close:      bar.Close,
			Volume:     bar.Volume,
			TradeCount: bar.TradeCount,
			VWAP:       bar.VWAP,
		}
	}

	return c.JSON(response)
}

// GetAsset retrieves asset information by symbol
// GET /assets/:symbol
func (h *Handler) GetAsset(c *fiber.Ctx) error {
	symbol := strings.ToUpper(c.Params("symbol"))
	if symbol == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Error:   "bad_request",
			Message: "Symbol is required",
			Code:    400,
		})
	}

	ctx := c.Context()

	asset, err := h.alpaca.GetAsset(ctx, symbol)
	if err != nil {
		logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to fetch asset")
		return c.Status(fiber.StatusNotFound).JSON(types.ErrorResponse{
			Error:   "not_found",
			Message: "Asset not found",
			Code:    404,
		})
	}

	return c.JSON(types.AssetResponse{
		ID:           asset.ID,
		Symbol:       asset.Symbol,
		Name:         asset.Name,
		Exchange:     asset.Exchange,
		Class:        asset.Class,
		Status:       asset.Status,
		Tradable:     asset.Tradable,
		Fractionable: asset.Fractionable,
		Marginable:   asset.Marginable,
		Shortable:    asset.Shortable,
	})
}

// SearchAssets searches for tradeable assets
// GET /assets/search?q=apple&class=us_equity&status=active&limit=20
func (h *Handler) SearchAssets(c *fiber.Ctx) error {
	query := strings.ToUpper(c.Query("q"))
	class := c.Query("class", "us_equity")
	status := c.Query("status", "active")
	limit := c.QueryInt("limit", 20)

	if limit > 100 {
		limit = 100
	}

	ctx := c.Context()

	params := &alpaca.ListAssetsParams{
		Status:     status,
		AssetClass: class,
	}

	assets, err := h.alpaca.ListAssets(ctx, params)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list assets")
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to search assets",
			Code:    500,
		})
	}

	// Filter by query if provided
	filtered := make([]types.AssetResponse, 0)
	for _, asset := range assets {
		if query != "" {
			symbolMatch := strings.Contains(strings.ToUpper(asset.Symbol), query)
			nameMatch := strings.Contains(strings.ToUpper(asset.Name), query)
			if !symbolMatch && !nameMatch {
				continue
			}
		}

		filtered = append(filtered, types.AssetResponse{
			ID:           asset.ID,
			Symbol:       asset.Symbol,
			Name:         asset.Name,
			Exchange:     asset.Exchange,
			Class:        asset.Class,
			Status:       asset.Status,
			Tradable:     asset.Tradable,
			Fractionable: asset.Fractionable,
			Marginable:   asset.Marginable,
			Shortable:    asset.Shortable,
		})

		if len(filtered) >= limit {
			break
		}
	}

	return c.JSON(types.AssetSearchResponse{
		Assets: filtered,
		Total:  len(filtered),
	})
}

// GetClock retrieves the current market clock
// GET /market/clock
func (h *Handler) GetClock(c *fiber.Ctx) error {
	ctx := c.Context()

	clock, err := h.alpaca.GetClock(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch market clock")
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch market clock",
			Code:    500,
		})
	}

	return c.JSON(types.ClockResponse{
		Timestamp: clock.Timestamp.Format(time.RFC3339),
		IsOpen:    clock.IsOpen,
		NextOpen:  clock.NextOpen.Format(time.RFC3339),
		NextClose: clock.NextClose.Format(time.RFC3339),
	})
}

// GetCalendar retrieves the market calendar
// GET /market/calendar?start=2024-01-01&end=2024-12-31
func (h *Handler) GetCalendar(c *fiber.Ctx) error {
	start := c.Query("start")
	end := c.Query("end")

	var params *alpaca.GetCalendarParams
	if start != "" || end != "" {
		params = &alpaca.GetCalendarParams{
			Start: start,
			End:   end,
		}
	}

	ctx := c.Context()

	calendar, err := h.alpaca.GetCalendar(ctx, params)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch market calendar")
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch market calendar",
			Code:    500,
		})
	}

	days := make([]types.CalendarDayResponse, len(calendar))
	for i, day := range calendar {
		days[i] = types.CalendarDayResponse{
			Date:  day.Date,
			Open:  day.Open,
			Close: day.Close,
		}
	}

	return c.JSON(types.CalendarResponse{
		Days: days,
	})
}

// GetSnapshot retrieves a complete snapshot for a symbol
// GET /snapshot/:symbol
func (h *Handler) GetSnapshot(c *fiber.Ctx) error {
	symbol := strings.ToUpper(c.Params("symbol"))
	if symbol == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Error:   "bad_request",
			Message: "Symbol is required",
			Code:    400,
		})
	}

	ctx := c.Context()

	snapshot, err := h.alpaca.GetSnapshot(ctx, symbol)
	if err != nil {
		logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to fetch snapshot")
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch snapshot",
			Code:    500,
		})
	}

	response := types.SnapshotResponse{
		Symbol: symbol,
	}

	if snapshot.LatestTrade != nil {
		response.LatestTrade = &types.TradeInfo{
			Timestamp: snapshot.LatestTrade.Timestamp.Format(time.RFC3339),
			Price:     snapshot.LatestTrade.Price,
			Size:      snapshot.LatestTrade.Size,
			Exchange:  snapshot.LatestTrade.Exchange,
		}
	}

	if snapshot.LatestQuote != nil {
		midPrice := (snapshot.LatestQuote.BidPrice + snapshot.LatestQuote.AskPrice) / 2
		spread := snapshot.LatestQuote.AskPrice - snapshot.LatestQuote.BidPrice
		response.LatestQuote = &types.QuoteResponse{
			Symbol:    symbol,
			BidPrice:  snapshot.LatestQuote.BidPrice,
			BidSize:   snapshot.LatestQuote.BidSize,
			AskPrice:  snapshot.LatestQuote.AskPrice,
			AskSize:   snapshot.LatestQuote.AskSize,
			MidPrice:  midPrice,
			Spread:    spread,
			Timestamp: snapshot.LatestQuote.Timestamp,
		}
	}

	if snapshot.MinuteBar != nil {
		response.MinuteBar = barToResponse(snapshot.MinuteBar)
	}
	if snapshot.DailyBar != nil {
		response.DailyBar = barToResponse(snapshot.DailyBar)
	}
	if snapshot.PrevDailyBar != nil {
		response.PrevDailyBar = barToResponse(snapshot.PrevDailyBar)
	}

	return c.JSON(response)
}

// Helper functions

func parseTime(s string) (time.Time, error) {
	// Try RFC3339 first
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}

	// Try date only format
	t, err = time.Parse("2006-01-02", s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, err
}

func barToResponse(bar *alpaca.Bar) *types.BarResponse {
	return &types.BarResponse{
		Timestamp:  bar.Timestamp,
		Open:       bar.Open,
		High:       bar.High,
		Low:        bar.Low,
		Close:      bar.Close,
		Volume:     bar.Volume,
		TradeCount: bar.TradeCount,
		VWAP:       bar.VWAP,
	}
}
