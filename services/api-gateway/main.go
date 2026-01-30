package main

import (
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/pkg/middleware"
)

func main() {
	logger.Init("api-gateway", "info", true)
	logger.Info().Msg("Starting API Gateway")

	app := fiber.New(fiber.Config{
		AppName:      "EquiShare API Gateway",
		ServerHeader: "EquiShare",
	})

	app.Use(recover.New())
	app.Use(middleware.RequestID())
	app.Use(middleware.SecurityHeaders())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy", "service": "api-gateway"})
	})

	api := app.Group("/api/v1")
	api.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Welcome to EquiShare API", "version": "1.0.0"})
	})

	go func() {
		if err := app.Listen(":8000"); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down API Gateway")
	if err := app.Shutdown(); err != nil {
		logger.Error().Err(err).Msg("Error during shutdown")
	}
}
