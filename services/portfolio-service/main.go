package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/Rohianon/equishare-global-trading/pkg/alpaca"
	"github.com/Rohianon/equishare-global-trading/pkg/config"
	"github.com/Rohianon/equishare-global-trading/pkg/database"
	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/pkg/middleware"
	"github.com/Rohianon/equishare-global-trading/services/portfolio-service/internal/handler"
	"github.com/Rohianon/equishare-global-trading/services/portfolio-service/internal/repository"
)

func main() {
	logger.Init("portfolio-service", "info", true)
	logger.Info().Msg("Starting Portfolio Service")

	cfg, err := config.Load("config")
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to load config, using defaults")
	}

	ctx := context.Background()

	// Database connection
	dbCfg := &database.Config{
		Host:     getEnvOrDefault("DB_HOST", cfg.Database.Host),
		Port:     cfg.Database.Port,
		User:     getEnvOrDefault("DB_USER", cfg.Database.User),
		Password: getEnvOrDefault("DB_PASSWORD", cfg.Database.Password),
		Database: getEnvOrDefault("DB_NAME", cfg.Database.Database),
		SSLMode:  cfg.Database.SSLMode,
	}
	if dbCfg.Port == 0 {
		dbCfg.Port = 5432
	}
	if dbCfg.SSLMode == "" {
		dbCfg.SSLMode = "disable"
	}

	db, err := database.NewPool(ctx, dbCfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()
	logger.Info().Msg("Connected to database")

	// Alpaca client
	var alpacaClient alpaca.TradingClient
	alpacaAPIKey := os.Getenv("ALPACA_API_KEY")
	alpacaSecretKey := os.Getenv("ALPACA_SECRET_KEY")
	alpacaPaper := os.Getenv("ALPACA_PAPER") != "false"

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

	// Initialize repository and handler
	repo := repository.NewRepository(db)
	h := handler.NewHandler(repo, alpacaClient)

	// Setup Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "EquiShare Portfolio Service",
		ErrorHandler: customErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(middleware.RequestID())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-User-ID",
		AllowMethods: "GET, POST, OPTIONS",
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "portfolio-service",
		})
	})

	// API routes
	api := app.Group("/api/v1")

	// Portfolio endpoints
	api.Get("/portfolio", h.GetPortfolio)                  // Full portfolio with summary
	api.Get("/portfolio/holdings", h.GetHoldings)          // All holdings
	api.Get("/portfolio/holdings/:symbol", h.GetHolding)   // Specific holding
	api.Get("/portfolio/allocation", h.GetAllocation)      // Allocation breakdown
	api.Get("/portfolio/performance", h.GetPerformance)    // Performance metrics

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8008"
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

	logger.Info().Msg("Shutting down Portfolio Service")
	if err := app.Shutdown(); err != nil {
		logger.Error().Err(err).Msg("Error during shutdown")
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
		"error": message,
		"code":  code,
	})
}
