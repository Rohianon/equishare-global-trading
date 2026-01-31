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

	"github.com/Rohianon/equishare-global-trading/pkg/config"
	"github.com/Rohianon/equishare-global-trading/pkg/database"
	"github.com/Rohianon/equishare-global-trading/pkg/events"
	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/pkg/middleware"
	"github.com/Rohianon/equishare-global-trading/pkg/mpesa"
	"github.com/Rohianon/equishare-global-trading/pkg/sms"
	"github.com/Rohianon/equishare-global-trading/services/payment-service/internal/handler"
	"github.com/Rohianon/equishare-global-trading/services/payment-service/internal/repository"
)

func main() {
	logger.Init("payment-service", "info", true)
	logger.Info().Msg("Starting Payment Service")

	cfg, err := config.Load("config")
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to load config, using defaults")
	}

	ctx := context.Background()

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

	var mpesaClient handler.MpesaClient
	if os.Getenv("MPESA_SANDBOX") == "true" || os.Getenv("MPESA_CONSUMER_KEY") == "" {
		logger.Warn().Msg("Using mock M-Pesa client")
		mpesaClient = mpesa.NewMockClient()
	} else {
		mpesaClient = mpesa.NewClient(&mpesa.Config{
			ConsumerKey:    os.Getenv("MPESA_CONSUMER_KEY"),
			ConsumerSecret: os.Getenv("MPESA_CONSUMER_SECRET"),
			PassKey:        os.Getenv("MPESA_PASSKEY"),
			ShortCode:      os.Getenv("MPESA_SHORTCODE"),
			CallbackURL:    os.Getenv("MPESA_CALLBACK_URL"),
			Sandbox:        os.Getenv("MPESA_SANDBOX") == "true",
		})
	}

	var smsClient handler.SMSClient
	if os.Getenv("SMS_SANDBOX") == "true" || os.Getenv("AT_API_KEY") == "" {
		logger.Warn().Msg("Using mock SMS client")
		smsClient = sms.NewMockClient()
	} else {
		smsClient = sms.NewClient(&sms.Config{
			APIKey:   os.Getenv("AT_API_KEY"),
			Username: os.Getenv("AT_USERNAME"),
			Sender:   os.Getenv("AT_SENDER"),
			Sandbox:  os.Getenv("AT_SANDBOX") == "true",
		})
	}

	var publisher events.Publisher
	if brokers := cfg.Kafka.Brokers; len(brokers) > 0 && brokers[0] != "" {
		publisher = events.NewKafkaPublisher(brokers)
		defer publisher.Close()
		logger.Info().Msg("Connected to Kafka")
	} else {
		logger.Warn().Msg("Kafka not configured, events will not be published")
	}

	userRepo := repository.NewUserRepository(db)
	walletRepo := repository.NewWalletRepository(db)
	mpesaRepo := repository.NewMpesaRepository(db)

	h := handler.New(userRepo, walletRepo, mpesaRepo, mpesaClient, smsClient, publisher)

	jwtSecret := getEnvOrDefault("JWT_SECRET", "dev-secret-change-in-production")

	app := fiber.New(fiber.Config{
		AppName:      "EquiShare Payment Service",
		ErrorHandler: errorHandler,
	})
	app.Use(recover.New())
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger())
	app.Use(middleware.SecurityHeaders())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy", "service": "payment-service"})
	})

	app.Post("/webhooks/mpesa/stk-callback", h.STKCallback)

	api := app.Group("/api/v1")
	wallet := api.Group("/wallet", middleware.Auth(jwtSecret))
	wallet.Post("/deposit", h.Deposit)
	wallet.Get("/balance", h.GetWalletBalance)

	port := getEnvOrDefault("PORT", "8004")
	go func() {
		if err := app.Listen(":" + port); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()
	logger.Info().Str("port", port).Msg("Payment Service started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down Payment Service")
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
