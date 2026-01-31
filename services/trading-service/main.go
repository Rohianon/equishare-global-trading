package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/Rohianon/equishare-global-trading/pkg/alpaca"
	"github.com/Rohianon/equishare-global-trading/pkg/config"
	"github.com/Rohianon/equishare-global-trading/pkg/database"
	"github.com/Rohianon/equishare-global-trading/pkg/events"
	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/pkg/middleware"
	"github.com/Rohianon/equishare-global-trading/services/trading-service/internal/handler"
	"github.com/Rohianon/equishare-global-trading/services/trading-service/internal/repository"
)

func main() {
	logger.Init("trading-service", "info", true)
	logger.Info().Msg("Starting Trading Service")

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

	// Kafka publisher
	var publisher events.Publisher
	if brokers := cfg.Kafka.Brokers; len(brokers) > 0 && brokers[0] != "" {
		publisher = events.NewKafkaPublisher(brokers)
		defer publisher.Close()
		logger.Info().Msg("Connected to Kafka")
	} else {
		logger.Warn().Msg("Kafka not configured, events will not be published")
	}

	// Repositories
	userRepo := repository.NewUserRepository(db)
	walletRepo := repository.NewWalletRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	holdingRepo := repository.NewHoldingRepository(db)

	// Handler
	h := handler.New(userRepo, walletRepo, orderRepo, holdingRepo, alpacaClient, publisher)

	// JWT secret
	jwtSecret := getEnvOrDefault("JWT_SECRET", "dev-secret-change-in-production")

	// Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "EquiShare Trading Service",
		ErrorHandler: errorHandler,
	})
	app.Use(recover.New())
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger())
	app.Use(middleware.SecurityHeaders())

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy", "service": "trading-service"})
	})

	// Webhook endpoint (no auth required)
	app.Post("/webhooks/alpaca/orders", h.AlpacaWebhook)

	// API routes (auth required)
	api := app.Group("/api/v1", middleware.Auth(jwtSecret))

	// Orders
	orders := api.Group("/orders")
	orders.Post("/", h.PlaceOrder)
	orders.Get("/", h.ListOrders)
	orders.Get("/:id", h.GetOrder)
	orders.Delete("/:id", h.CancelOrder)

	// Portfolio
	api.Get("/portfolio", h.GetPortfolio)

	// Market data
	api.Get("/quotes/:symbol", h.GetQuote)
	api.Get("/assets/search", h.SearchAssets)

	// Start server
	port := getEnvOrDefault("PORT", "8003")
	go func() {
		if err := app.Listen(":" + port); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()
	logger.Info().Str("port", port).Msg("Trading Service started")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down Trading Service")
	if err := app.Shutdown(); err != nil {
		logger.Error().Err(err).Msg("Error during shutdown")
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error": message,
	})
}
