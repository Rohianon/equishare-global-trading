package middleware

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/Rohianon/equishare-global-trading/pkg/logger"
)

func init() {
	logger.Init("test", "error", false)
}

func TestRequestID(t *testing.T) {
	app := fiber.New()
	app.Use(RequestID())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(GetRequestID(c))
	})

	t.Run("generates new request ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test error: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if len(body) == 0 {
			t.Error("RequestID should be generated")
		}

		if resp.Header.Get("X-Request-ID") == "" {
			t.Error("X-Request-ID header should be set")
		}
	})

	t.Run("uses existing request ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Request-ID", "test-request-id")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test error: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "test-request-id" {
			t.Errorf("RequestID = %v, want test-request-id", string(body))
		}
	})
}

func TestGetRequestID(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		id := GetRequestID(c)
		return c.SendString(id)
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "" {
		t.Error("GetRequestID should return empty string when no ID is set")
	}
}

func TestSecurityHeaders(t *testing.T) {
	app := fiber.New()
	app.Use(SecurityHeaders())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for header, value := range expectedHeaders {
		if resp.Header.Get(header) != value {
			t.Errorf("%s = %v, want %v", header, resp.Header.Get(header), value)
		}
	}
}

func TestLogger(t *testing.T) {
	app := fiber.New()
	app.Use(RequestID())
	app.Use(Logger())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Status = %v, want 200", resp.StatusCode)
	}
}

func TestRateLimiter(t *testing.T) {
	app := fiber.New()
	app.Use(RateLimiter(RateLimitConfig{
		Max:      2,
		Duration: time.Second,
	}))
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		resp, _ := app.Test(req)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("Request %d should succeed, got status %d", i+1, resp.StatusCode)
		}
	}

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()
	if resp.StatusCode != 429 {
		t.Errorf("Request 3 should be rate limited, got status %d", resp.StatusCode)
	}
}

func TestAuth(t *testing.T) {
	jwtSecret := "test-secret"
	app := fiber.New()
	app.Use(Auth(jwtSecret))
	app.Get("/", func(c *fiber.Ctx) error {
		userID := GetUserID(c)
		return c.SendString(userID)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.StatusCode != 401 {
			t.Errorf("Status = %v, want 401", resp.StatusCode)
		}
	})

	t.Run("invalid authorization format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.StatusCode != 401 {
			t.Errorf("Status = %v, want 401", resp.StatusCode)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.StatusCode != 401 {
			t.Errorf("Status = %v, want 401", resp.StatusCode)
		}
	})

	t.Run("valid token", func(t *testing.T) {
		claims := &Claims{
			UserID: "user-123",
			Phone:  "+1234567890",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(jwtSecret))

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Status = %v, want 200", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "user-123" {
			t.Errorf("UserID = %v, want user-123", string(body))
		}
	})
}

func TestCORS(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		app := fiber.New()
		app.Use(CORS(CORSConfig{}))
		app.Get("/", func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})

		req := httptest.NewRequest("GET", "/", nil)
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Default CORS should allow all origins")
		}
	})

	t.Run("custom config", func(t *testing.T) {
		app := fiber.New()
		app.Use(CORS(CORSConfig{
			AllowOrigins:     []string{"https://example.com"},
			AllowMethods:     []string{"GET", "POST"},
			AllowHeaders:     []string{"Content-Type"},
			AllowCredentials: true,
		}))
		app.Get("/", func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})

		req := httptest.NewRequest("GET", "/", nil)
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.Header.Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Errorf("CORS origin = %v, want https://example.com", resp.Header.Get("Access-Control-Allow-Origin"))
		}
		if resp.Header.Get("Access-Control-Allow-Credentials") != "true" {
			t.Error("CORS credentials should be true")
		}
	})

	t.Run("preflight request", func(t *testing.T) {
		app := fiber.New()
		app.Use(CORS(CORSConfig{}))
		app.Get("/", func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})

		req := httptest.NewRequest("OPTIONS", "/", nil)
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.StatusCode != 204 {
			t.Errorf("Preflight status = %v, want 204", resp.StatusCode)
		}
	})
}

func TestGetUserID(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		id := GetUserID(c)
		return c.SendString(id)
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "" {
		t.Error("GetUserID should return empty string when no user is set")
	}
}
