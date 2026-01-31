package middleware

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/pkg/response"
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
	// Use error handler to properly convert AppErrors
	app := fiber.New(fiber.Config{
		ErrorHandler: response.ErrorHandler,
	})
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

	// Verify rate limit headers are set
	if resp.Header.Get("X-RateLimit-Limit") == "" {
		t.Error("X-RateLimit-Limit header should be set")
	}
	if resp.Header.Get("X-RateLimit-Remaining") == "" {
		t.Error("X-RateLimit-Remaining header should be set")
	}
	if resp.Header.Get("X-RateLimit-Reset") == "" {
		t.Error("X-RateLimit-Reset header should be set")
	}
}

func TestRateLimiterReset(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: response.ErrorHandler,
	})
	app.Use(RateLimiter(RateLimitConfig{
		Max:      1,
		Duration: 100 * time.Millisecond,
	}))
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// First request should succeed
	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("First request should succeed, got status %d", resp.StatusCode)
	}

	// Second request should be rate limited
	req = httptest.NewRequest("GET", "/", nil)
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != 429 {
		t.Errorf("Second request should be rate limited, got status %d", resp.StatusCode)
	}

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// Third request should succeed after reset
	req = httptest.NewRequest("GET", "/", nil)
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("Request after reset should succeed, got status %d", resp.StatusCode)
	}
}

func TestAuth(t *testing.T) {
	jwtSecret := "test-secret"
	// Use error handler to properly convert AppErrors
	app := fiber.New(fiber.Config{
		ErrorHandler: response.ErrorHandler,
	})
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

	t.Run("expired token", func(t *testing.T) {
		claims := &Claims{
			UserID: "user-123",
			Phone:  "+1234567890",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(jwtSecret))

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.StatusCode != 401 {
			t.Errorf("Status = %v, want 401", resp.StatusCode)
		}
	})
}

func TestOptionalAuth(t *testing.T) {
	jwtSecret := "test-secret"
	app := fiber.New()
	app.Use(OptionalAuth(jwtSecret))
	app.Get("/", func(c *fiber.Ctx) error {
		userID := GetUserID(c)
		if userID == "" {
			return c.SendString("anonymous")
		}
		return c.SendString(userID)
	})

	t.Run("no authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Status = %v, want 200", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "anonymous" {
			t.Errorf("Body = %v, want anonymous", string(body))
		}
	})

	t.Run("invalid token passes through", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Status = %v, want 200", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "anonymous" {
			t.Errorf("Body = %v, want anonymous", string(body))
		}
	})

	t.Run("valid token sets user", func(t *testing.T) {
		claims := &Claims{
			UserID: "user-456",
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

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "user-456" {
			t.Errorf("Body = %v, want user-456", string(body))
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

func TestGetPhone(t *testing.T) {
	jwtSecret := "test-secret"
	app := fiber.New()
	app.Use(OptionalAuth(jwtSecret))
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(GetPhone(c))
	})

	t.Run("no phone set", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "" {
			t.Error("GetPhone should return empty string when no user is set")
		}
	})

	t.Run("phone from token", func(t *testing.T) {
		claims := &Claims{
			UserID: "user-123",
			Phone:  "+254712345678",
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

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "+254712345678" {
			t.Errorf("GetPhone = %v, want +254712345678", string(body))
		}
	})
}

// =============================================================================
// 2FA Middleware Tests
// =============================================================================

// Mock2FAValidator implements TwoFactorValidator for testing
type Mock2FAValidator struct {
	users map[string]struct {
		enabled       bool
		totpCode      string
		recoveryCodes []string
	}
}

func NewMock2FAValidator() *Mock2FAValidator {
	return &Mock2FAValidator{
		users: make(map[string]struct {
			enabled       bool
			totpCode      string
			recoveryCodes []string
		}),
	}
}

func (m *Mock2FAValidator) SetUser(userID string, enabled bool, totpCode string, recoveryCodes []string) {
	m.users[userID] = struct {
		enabled       bool
		totpCode      string
		recoveryCodes []string
	}{enabled, totpCode, recoveryCodes}
}

func (m *Mock2FAValidator) Is2FAEnabled(userID string) (bool, error) {
	user, exists := m.users[userID]
	if !exists {
		return false, nil
	}
	return user.enabled, nil
}

func (m *Mock2FAValidator) ValidateCode(userID, code string) (bool, error) {
	user, exists := m.users[userID]
	if !exists {
		return false, nil
	}
	return user.totpCode == code, nil
}

func (m *Mock2FAValidator) ValidateRecoveryCode(userID, code string) (bool, error) {
	user, exists := m.users[userID]
	if !exists {
		return false, nil
	}
	for _, rc := range user.recoveryCodes {
		if rc == code {
			return true, nil
		}
	}
	return false, nil
}

func TestRequire2FA(t *testing.T) {
	validator := NewMock2FAValidator()

	// Set up user with 2FA enabled
	validator.SetUser("user-with-2fa", true, "123456", []string{"ABCD-1234"})
	// User without 2FA
	validator.SetUser("user-without-2fa", false, "", nil)

	t.Run("no authentication", func(t *testing.T) {
		app := fiber.New(fiber.Config{
			ErrorHandler: response.ErrorHandler,
		})
		app.Use(Require2FA(validator))
		app.Get("/", func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})

		req := httptest.NewRequest("GET", "/", nil)
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.StatusCode != 401 {
			t.Errorf("Status = %v, want 401 (unauthorized)", resp.StatusCode)
		}
	})

	t.Run("2FA not enabled passes through", func(t *testing.T) {
		app := fiber.New(fiber.Config{
			ErrorHandler: response.ErrorHandler,
		})
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("user_id", "user-without-2fa")
			return c.Next()
		})
		app.Use(Require2FA(validator))
		app.Get("/", func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})

		req := httptest.NewRequest("GET", "/", nil)
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Status = %v, want 200 (2FA not required)", resp.StatusCode)
		}
	})

	t.Run("2FA required but no code", func(t *testing.T) {
		app := fiber.New(fiber.Config{
			ErrorHandler: response.ErrorHandler,
		})
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("user_id", "user-with-2fa")
			return c.Next()
		})
		app.Use(Require2FA(validator))
		app.Get("/", func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})

		req := httptest.NewRequest("GET", "/", nil)
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.StatusCode != 403 {
			t.Errorf("Status = %v, want 403 (2FA required)", resp.StatusCode)
		}
	})

	t.Run("valid TOTP code", func(t *testing.T) {
		app := fiber.New(fiber.Config{
			ErrorHandler: response.ErrorHandler,
		})
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("user_id", "user-with-2fa")
			return c.Next()
		})
		app.Use(Require2FA(validator))
		app.Get("/", func(c *fiber.Ctx) error {
			verified := Get2FAVerified(c)
			if verified {
				return c.SendString("verified")
			}
			return c.SendString("not-verified")
		})

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-2FA-Code", "123456")
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Status = %v, want 200", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "verified" {
			t.Errorf("Body = %v, want verified", string(body))
		}
	})

	t.Run("invalid TOTP code", func(t *testing.T) {
		app := fiber.New(fiber.Config{
			ErrorHandler: response.ErrorHandler,
		})
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("user_id", "user-with-2fa")
			return c.Next()
		})
		app.Use(Require2FA(validator))
		app.Get("/", func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-2FA-Code", "000000")
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.StatusCode != 400 {
			t.Errorf("Status = %v, want 400 (invalid code)", resp.StatusCode)
		}
	})

	t.Run("valid recovery code", func(t *testing.T) {
		app := fiber.New(fiber.Config{
			ErrorHandler: response.ErrorHandler,
		})
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("user_id", "user-with-2fa")
			return c.Next()
		})
		app.Use(Require2FA(validator))
		app.Get("/", func(c *fiber.Ctx) error {
			usedRecovery := GetUsedRecoveryCode(c)
			if usedRecovery {
				return c.SendString("used-recovery")
			}
			return c.SendString("totp")
		})

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-2FA-Code", "ABCD-1234")
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Status = %v, want 200", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "used-recovery" {
			t.Errorf("Body = %v, want used-recovery", string(body))
		}
	})
}

func TestGet2FAVerified(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		if Get2FAVerified(c) {
			return c.SendString("verified")
		}
		return c.SendString("not-verified")
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "not-verified" {
		t.Error("Get2FAVerified should return false when not set")
	}
}

func TestGetUsedRecoveryCode(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		if GetUsedRecoveryCode(c) {
			return c.SendString("used-recovery")
		}
		return c.SendString("not-used")
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "not-used" {
		t.Error("GetUsedRecoveryCode should return false when not set")
	}
}
