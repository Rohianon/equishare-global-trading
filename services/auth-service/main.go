package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/Rohianon/equishare-global-trading/pkg/auth"
	"github.com/Rohianon/equishare-global-trading/pkg/cache"
	"github.com/Rohianon/equishare-global-trading/pkg/config"
	"github.com/Rohianon/equishare-global-trading/pkg/database"
	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/pkg/middleware"
	"github.com/Rohianon/equishare-global-trading/pkg/sms"
	"github.com/Rohianon/equishare-global-trading/services/auth-service/internal/handler"
	"github.com/Rohianon/equishare-global-trading/services/auth-service/internal/repository"
)

func main() {
	logger.Init("auth-service", "info", true)
	logger.Info().Msg("Starting Auth Service")

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

	redisCfg := &cache.Config{
		Host:     getEnvOrDefault("REDIS_HOST", cfg.Redis.Host),
		Port:     cfg.Redis.Port,
		Password: getEnvOrDefault("REDIS_PASSWORD", cfg.Redis.Password),
		DB:       cfg.Redis.DB,
	}
	if redisCfg.Port == 0 {
		redisCfg.Port = 6379
	}

	redisCache, err := cache.NewRedisCache(redisCfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer redisCache.Close()
	logger.Info().Msg("Connected to Redis")

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

	jwtManager := auth.NewJWTManager(&auth.Config{
		Secret:          getEnvOrDefault("JWT_SECRET", "dev-secret-change-in-production"),
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	})

	userRepo := repository.NewUserRepository(db)
	walletRepo := repository.NewWalletRepository(db)

	h := handler.New(userRepo, walletRepo, redisCache, smsClient, jwtManager)

	app := fiber.New(fiber.Config{
		AppName:      "EquiShare Auth Service",
		ErrorHandler: errorHandler,
	})
	app.Use(recover.New())
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger())
	app.Use(middleware.SecurityHeaders())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy", "service": "auth-service"})
	})

	api := app.Group("/api/v1")
	authGroup := api.Group("/auth")
	authGroup.Post("/register", h.Register)
	authGroup.Post("/verify", h.Verify)
	authGroup.Post("/login", h.Login)
	authGroup.Post("/refresh", h.RefreshToken)

	port := getEnvOrDefault("PORT", "8001")
	go func() {
		if err := app.Listen(":" + port); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()
	logger.Info().Str("port", port).Msg("Auth Service started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down Auth Service")
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
