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
	"github.com/Rohianon/equishare-global-trading/pkg/sms"
	"github.com/Rohianon/equishare-global-trading/services/notification-service/internal/handler"
	"github.com/Rohianon/equishare-global-trading/services/notification-service/internal/sender"
)

func main() {
	logger.Init("notification-service", "info", true)
	logger.Info().Msg("Starting Notification Service")

	// SMS client
	var smsClient sender.SMSClient
	smsAPIKey := os.Getenv("SMS_API_KEY")
	smsUsername := os.Getenv("SMS_USERNAME")
	smsSandbox := os.Getenv("SMS_SANDBOX") == "true"

	if smsAPIKey == "" {
		logger.Warn().Msg("Using mock SMS client (no API key configured)")
		smsClient = sms.NewMockClient()
	} else {
		smsClient = sms.NewClient(&sms.Config{
			APIKey:   smsAPIKey,
			Username: smsUsername,
			Sender:   os.Getenv("SMS_SENDER"),
			Sandbox:  smsSandbox,
		})
		logger.Info().Bool("sandbox", smsSandbox).Msg("Connected to Africa's Talking SMS")
	}

	// Initialize sender and handler
	s := sender.NewSender(smsClient)
	h := handler.NewHandler(s)

	// Setup Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "EquiShare Notification Service",
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
			"service": "notification-service",
		})
	})

	// API routes
	api := app.Group("/api/v1")

	// Notification endpoints
	notifications := api.Group("/notifications")
	notifications.Post("/send", h.SendNotification)      // Send a notification
	notifications.Get("/", h.ListNotifications)          // List notifications
	notifications.Get("/templates", h.ListTemplates)     // List templates
	notifications.Get("/:id", h.GetNotification)         // Get specific notification

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8005"
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

	logger.Info().Msg("Shutting down Notification Service")
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
		"error": message,
		"code":  code,
	})
}
