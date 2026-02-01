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
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/pkg/middleware"
	"github.com/Rohianon/equishare-global-trading/pkg/response"
	"github.com/Rohianon/equishare-global-trading/pkg/telemetry"
	"github.com/Rohianon/equishare-global-trading/services/api-gateway/internal/config"
	"github.com/Rohianon/equishare-global-trading/services/api-gateway/internal/handler"
	"github.com/Rohianon/equishare-global-trading/services/api-gateway/internal/proxy"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logLevel := "info"
	if cfg.IsDevelopment() {
		logLevel = "debug"
	}
	logger.Init("api-gateway", logLevel, cfg.IsDevelopment())
	logger.Info().Str("env", cfg.Environment).Msg("Starting API Gateway")

	// Initialize telemetry if configured
	var telemetryProvider *telemetry.Provider
	if cfg.OTLPEndpoint != "" {
		var err error
		telemetryProvider, err = telemetry.Init(context.Background(), &telemetry.Config{
			ServiceName:  "api-gateway",
			Version:      "1.0.0",
			Environment:  cfg.Environment,
			CollectorURL: cfg.OTLPEndpoint,
			Enabled:      true,
		})
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to initialize telemetry")
		}
	}

	// Create Fiber app with custom error handler
	app := fiber.New(fiber.Config{
		AppName:               "EquiShare API Gateway",
		ServerHeader:          "EquiShare",
		DisableStartupMessage: !cfg.IsDevelopment(),
		ErrorHandler:          response.ErrorHandler,
		ReadTimeout:           15 * time.Second,
		WriteTimeout:          15 * time.Second,
		IdleTimeout:           60 * time.Second,
	})

	// Global middleware
	app.Use(recover.New(recover.Config{
		EnableStackTrace: cfg.IsDevelopment(),
	}))
	app.Use(middleware.RequestID())
	app.Use(middleware.SecurityHeaders())

	// CORS
	allowCredentials := !contains(cfg.CORSAllowOrigins, "*")
	app.Use(cors.New(cors.Config{
		AllowOrigins:     joinStrings(cfg.CORSAllowOrigins),
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Request-ID",
		AllowCredentials: allowCredentials,
		MaxAge:           86400,
	}))

	// Logging (after request ID)
	app.Use(middleware.Logger())

	// Tracing (if enabled)
	if cfg.OTLPEndpoint != "" {
		app.Use(middleware.Tracing("api-gateway"))
	}

	// Rate limiting
	app.Use(middleware.RateLimiter(middleware.RateLimitConfig{
		Max:      cfg.RateLimit,
		Duration: cfg.RateLimitDuration,
	}))

	// Initialize handlers
	h := handler.New()
	p := proxy.New()

	// Health endpoints (no auth required)
	app.Get("/health", h.Health)
	app.Get("/ready", h.Ready)
	app.Get("/live", h.Live)

	// API v1 routes
	api := app.Group("/api/v1")

	// Root endpoint
	api.Get("/", h.Root)
	api.Get("/info", h.Info)

	// Auth routes (no auth required)
	auth := api.Group("/auth")
	auth.Post("/register", p.Forward(cfg.AuthServiceURL))
	auth.Post("/verify", p.Forward(cfg.AuthServiceURL))
	auth.Post("/login", p.Forward(cfg.AuthServiceURL))
	auth.Post("/refresh", p.Forward(cfg.AuthServiceURL))

	// Protected auth routes
	authProtected := auth.Group("", middleware.Auth(cfg.JWTSecret))
	authProtected.Post("/logout", p.Forward(cfg.AuthServiceURL))
	authProtected.Get("/me", p.Forward(cfg.AuthServiceURL))

	// User routes (protected)
	users := api.Group("/users", middleware.Auth(cfg.JWTSecret))
	users.All("/*", p.Forward(cfg.UserServiceURL))

	// Payment routes (protected)
	payments := api.Group("/payments", middleware.Auth(cfg.JWTSecret))
	payments.All("/*", p.Forward(cfg.PaymentServiceURL))

	// Trading routes (protected)
	trading := api.Group("/trading", middleware.Auth(cfg.JWTSecret))
	trading.All("/*", p.Forward(cfg.TradingServiceURL))

	// 404 handler
	app.Use(h.NotFound)

	// Start server
	go func() {
		addr := ":" + cfg.Port
		logger.Info().Str("addr", addr).Msg("API Gateway listening")
		if err := app.Listen(addr); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down API Gateway")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error().Err(err).Msg("Error during shutdown")
	}

	// Shutdown telemetry
	if telemetryProvider != nil {
		if err := telemetryProvider.Shutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("Error shutting down telemetry")
		}
	}

	logger.Info().Msg("API Gateway stopped")
}

func joinStrings(s []string) string {
	result := ""
	for i, str := range s {
		if i > 0 {
			result += ","
		}
		result += str
	}
	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
