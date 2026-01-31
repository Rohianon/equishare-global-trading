package handler

import (
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Rohianon/equishare-global-trading/pkg/response"
)

var startTime = time.Now()

// Handler provides API Gateway endpoints
type Handler struct{}

// New creates a new handler instance
func New() *Handler {
	return &Handler{}
}

// Health returns service health status
func (h *Handler) Health(c *fiber.Ctx) error {
	return response.Success(c, fiber.Map{
		"status":  "healthy",
		"service": "api-gateway",
		"uptime":  time.Since(startTime).String(),
	})
}

// Ready returns readiness status for k8s probes
func (h *Handler) Ready(c *fiber.Ctx) error {
	// Add any dependency checks here (db, redis, etc.)
	return response.Success(c, fiber.Map{
		"status": "ready",
	})
}

// Live returns liveness status for k8s probes
func (h *Handler) Live(c *fiber.Ctx) error {
	return response.Success(c, fiber.Map{
		"status": "alive",
	})
}

// Root returns API welcome message
func (h *Handler) Root(c *fiber.Ctx) error {
	return response.Success(c, fiber.Map{
		"message": "Welcome to EquiShare API",
		"version": "1.0.0",
		"docs":    "/docs",
	})
}

// Info returns detailed service information
func (h *Handler) Info(c *fiber.Ctx) error {
	return response.Success(c, fiber.Map{
		"service":    "api-gateway",
		"version":    "1.0.0",
		"go_version": runtime.Version(),
		"uptime":     time.Since(startTime).String(),
		"endpoints": fiber.Map{
			"auth":     "/api/v1/auth/*",
			"users":    "/api/v1/users/*",
			"payments": "/api/v1/payments/*",
			"trading":  "/api/v1/trading/*",
		},
	})
}

// NotFound handles 404 responses
func (h *Handler) NotFound(c *fiber.Ctx) error {
	return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Endpoint not found")
}
