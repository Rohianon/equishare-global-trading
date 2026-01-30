package main

import (
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/pkg/middleware"
)

func main() {
	logger.Init("notification-service", "info", true)
	logger.Info().Msg("Starting Notification Service")

	app := fiber.New(fiber.Config{AppName: "EquiShare Notification Service"})
	app.Use(recover.New())
	app.Use(middleware.RequestID())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy", "service": "notification-service"})
	})

	go func() {
		if err := app.Listen(":8006"); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down Notification Service")
	if err := app.Shutdown(); err != nil {
		logger.Error().Err(err).Msg("Error during shutdown")
	}
}
