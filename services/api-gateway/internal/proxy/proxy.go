package proxy

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/pkg/middleware"
)

// ServiceProxy handles proxying requests to backend services
type ServiceProxy struct {
	client *http.Client
}

// New creates a new service proxy
func New() *ServiceProxy {
	return &ServiceProxy{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// Forward creates a handler that forwards requests to the target service
func (p *ServiceProxy) Forward(targetURL string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Build target URL
		target := targetURL + c.OriginalURL()

		// Create request
		req, err := http.NewRequestWithContext(
			c.UserContext(),
			c.Method(),
			target,
			bytes.NewReader(c.Body()),
		)
		if err != nil {
			logger.Error().Err(err).Str("target", target).Msg("Failed to create proxy request")
			return fiber.NewError(fiber.StatusBadGateway, "Failed to create proxy request")
		}

		// Copy headers
		c.Request().Header.VisitAll(func(key, value []byte) {
			req.Header.Set(string(key), string(value))
		})

		// Add forwarding headers
		req.Header.Set("X-Forwarded-For", c.IP())
		req.Header.Set("X-Forwarded-Host", c.Hostname())
		req.Header.Set("X-Forwarded-Proto", c.Protocol())
		req.Header.Set("X-Request-ID", middleware.GetRequestID(c))

		// Add user context if authenticated
		if userID := middleware.GetUserID(c); userID != "" {
			req.Header.Set("X-User-ID", userID)
		}

		// Execute request
		resp, err := p.client.Do(req)
		if err != nil {
			logger.Error().Err(err).Str("target", target).Msg("Proxy request failed")
			return fiber.NewError(fiber.StatusBadGateway, "Service unavailable")
		}
		defer resp.Body.Close()

		// Copy response headers
		for key, values := range resp.Header {
			for _, value := range values {
				c.Response().Header.Add(key, value)
			}
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to read proxy response")
			return fiber.NewError(fiber.StatusBadGateway, "Failed to read response")
		}

		// Set status and send response
		c.Status(resp.StatusCode)
		return c.Send(body)
	}
}

// ForwardWithPath creates a handler that forwards to a specific path
func (p *ServiceProxy) ForwardWithPath(targetURL, basePath string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Strip the gateway prefix and build target URL
		path := c.Path()
		if len(path) > len(basePath) {
			path = path[len(basePath):]
		}
		target := targetURL + path

		if c.Request().URI().QueryString() != nil {
			target += "?" + string(c.Request().URI().QueryString())
		}

		// Create request
		req, err := http.NewRequestWithContext(
			c.UserContext(),
			c.Method(),
			target,
			bytes.NewReader(c.Body()),
		)
		if err != nil {
			logger.Error().Err(err).Str("target", target).Msg("Failed to create proxy request")
			return fiber.NewError(fiber.StatusBadGateway, "Failed to create proxy request")
		}

		// Copy headers
		c.Request().Header.VisitAll(func(key, value []byte) {
			req.Header.Set(string(key), string(value))
		})

		// Add forwarding headers
		req.Header.Set("X-Forwarded-For", c.IP())
		req.Header.Set("X-Forwarded-Host", c.Hostname())
		req.Header.Set("X-Forwarded-Proto", c.Protocol())
		req.Header.Set("X-Request-ID", middleware.GetRequestID(c))

		// Add user context if authenticated
		if userID := middleware.GetUserID(c); userID != "" {
			req.Header.Set("X-User-ID", userID)
		}

		// Execute request
		resp, err := p.client.Do(req)
		if err != nil {
			logger.Error().Err(err).Str("target", target).Msg("Proxy request failed")
			return fiber.NewError(fiber.StatusBadGateway, "Service unavailable")
		}
		defer resp.Body.Close()

		// Copy response headers
		for key, values := range resp.Header {
			for _, value := range values {
				c.Response().Header.Add(key, value)
			}
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to read proxy response")
			return fiber.NewError(fiber.StatusBadGateway, "Failed to read response")
		}

		// Set status and send response
		c.Status(resp.StatusCode)
		return c.Send(body)
	}
}
