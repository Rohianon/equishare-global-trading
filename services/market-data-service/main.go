package main

import (
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/websocket/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"

	"github.com/Rohianon/equishare-global-trading/pkg/alpaca"
	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/pkg/middleware"
	"github.com/Rohianon/equishare-global-trading/services/market-data-service/internal/handler"
	ws "github.com/Rohianon/equishare-global-trading/services/market-data-service/internal/websocket"
)

func main() {
	logger.Init("market-data-service", "info", true)
	logger.Info().Msg("Starting Market Data Service")

	// Alpaca client
	var alpacaClient alpaca.TradingClient
	alpacaAPIKey := os.Getenv("ALPACA_API_KEY")
	alpacaSecretKey := os.Getenv("ALPACA_SECRET_KEY")
	alpacaPaper := os.Getenv("ALPACA_PAPER") != "false" // Default to paper trading

	if alpacaAPIKey == "" {
		logger.Warn().Msg("Using mock Alpaca client (no API key configured)")
		alpacaClient = alpaca.NewMockClient()
	} else {
		alpacaClient = alpaca.NewClient(&alpaca.Config{
			APIKey:    alpacaAPIKey,
			SecretKey: alpacaSecretKey,
			Paper:     alpacaPaper,
		})
		logger.Info().Bool("paper", alpacaPaper).Msg("Connected to Alpaca")
	}

	// Initialize handler
	h := handler.NewHandler(alpacaClient)

	// Initialize WebSocket hub
	hub := ws.NewHub(alpacaClient)
	go hub.Run()

	// Setup Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "EquiShare Market Data Service",
		ErrorHandler: customErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(middleware.RequestID())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, OPTIONS",
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "market-data-service",
		})
	})

	// API routes
	api := app.Group("/api/v1")

	// Quote endpoints
	api.Get("/quotes", h.GetMultiQuotes)       // GET /api/v1/quotes?symbols=AAPL,GOOGL
	api.Get("/quotes/:symbol", h.GetQuote)     // GET /api/v1/quotes/AAPL

	// Historical data
	api.Get("/bars/:symbol", h.GetBars)        // GET /api/v1/bars/AAPL?timeframe=1Day&limit=100

	// Asset endpoints
	api.Get("/assets/search", h.SearchAssets)  // GET /api/v1/assets/search?q=apple
	api.Get("/assets/:symbol", h.GetAsset)     // GET /api/v1/assets/AAPL

	// Market status
	api.Get("/market/clock", h.GetClock)       // GET /api/v1/market/clock
	api.Get("/market/calendar", h.GetCalendar) // GET /api/v1/market/calendar

	// Snapshot (combined data)
	api.Get("/snapshot/:symbol", h.GetSnapshot) // GET /api/v1/snapshot/AAPL

	// WebSocket endpoint for real-time streaming
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		clientID := uuid.New().String()
		client := ws.NewClient(clientID, c, hub)
		hub.Register(client)

		// Start read and write pumps
		go client.WritePump()
		client.ReadPump()
	}))

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8007"
	}

	// Start server
	go func() {
		addr := ":" + port
		logger.Info().Str("addr", addr).Msg("Server listening")
		if err := app.Listen(addr); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down Market Data Service")
	if err := app.Shutdown(); err != nil {
		logger.Error().Err(err).Msg("Error during shutdown")
	}
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
		message = e.Message
	}

	logger.Error().Err(err).Int("status", code).Msg("Request error")

	return c.Status(code).JSON(fiber.Map{
		"error":   message,
		"code":    code,
	})
}
