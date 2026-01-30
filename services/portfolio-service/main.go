package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/pkg/middleware"
)

func main() {
	logger.Init("portfolio-service", "info", true)
	logger.Info().Msg("Starting Portfolio Service")

	app := fiber.New(fiber.Config{AppName: "EquiShare Portfolio Service"})
	app.Use(recover.New())
	app.Use(middleware.RequestID())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy", "service": "portfolio-service"})
	})

	go func() {
		if err := app.Listen(":8008"); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down Portfolio Service")
	app.Shutdown()
}
