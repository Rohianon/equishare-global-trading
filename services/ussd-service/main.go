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

	"github.com/Rohianon/equishare-global-trading/pkg/cache"
	"github.com/Rohianon/equishare-global-trading/pkg/config"
	"github.com/Rohianon/equishare-global-trading/pkg/database"
	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/pkg/middleware"
	"github.com/Rohianon/equishare-global-trading/services/ussd-service/internal/handler"
	"github.com/Rohianon/equishare-global-trading/services/ussd-service/internal/session"
)

func main() {
	logger.Init("ussd-service", "info", true)
	logger.Info().Msg("Starting USSD Service")

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

	sessionMgr := session.NewManager(redisCache)
	h := handler.New(sessionMgr, db)

	app := fiber.New(fiber.Config{
		AppName:      "EquiShare USSD Service",
		ErrorHandler: errorHandler,
	})
	app.Use(recover.New())
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy", "service": "ussd-service"})
	})

	app.Post("/ussd/callback", h.Callback)

	port := getEnvOrDefault("PORT", "8005")
	go func() {
		if err := app.Listen(":" + port); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()
	logger.Info().Str("port", port).Msg("USSD Service started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down USSD Service")
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
	logger.Error().Err(err).Msg("USSD error")
	return c.SendString("END System error. Please try again.")
}
