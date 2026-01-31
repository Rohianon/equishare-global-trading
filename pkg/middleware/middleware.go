package middleware

import (
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Rohianon/equishare-global-trading/pkg/logger"
)

func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Locals("request_id", requestID)
		c.Set("X-Request-ID", requestID)

		return c.Next()
	}
}

func GetRequestID(c *fiber.Ctx) string {
	if id, ok := c.Locals("request_id").(string); ok {
		return id
	}
	return ""
}

func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		return c.Next()
	}
}

func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		logger.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Dur("latency", time.Since(start)).
			Str("request_id", GetRequestID(c)).
			Msg("request")

		return err
	}
}

type RateLimitConfig struct {
	Max      int
	Duration time.Duration
}

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	config   RateLimitConfig
}

type visitor struct {
	count    int
	lastSeen time.Time
}

func RateLimiter(config RateLimitConfig) fiber.Handler {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		config:   config,
	}

	go rl.cleanup()

	return func(c *fiber.Ctx) error {
		ip := c.IP()

		rl.mu.Lock()
		v, exists := rl.visitors[ip]
		if !exists {
			rl.visitors[ip] = &visitor{count: 1, lastSeen: time.Now()}
			rl.mu.Unlock()
			return c.Next()
		}

		if time.Since(v.lastSeen) > rl.config.Duration {
			v.count = 1
			v.lastSeen = time.Now()
			rl.mu.Unlock()
			return c.Next()
		}

		if v.count >= rl.config.Max {
			rl.mu.Unlock()
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "rate limit exceeded",
			})
		}

		v.count++
		v.lastSeen = time.Now()
		rl.mu.Unlock()

		return c.Next()
	}
}

func (rl *rateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > rl.config.Duration*2 {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

type Claims struct {
	UserID string `json:"user_id"`
	Phone  string `json:"phone"`
	jwt.RegisteredClaims
}

func Auth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing authorization header",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authorization header format",
			})
		}

		tokenString := parts[1]

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token",
			})
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token claims",
			})
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("phone", claims.Phone)

		return c.Next()
	}
}

func GetUserID(c *fiber.Ctx) string {
	if id, ok := c.Locals("user_id").(string); ok {
		return id
	}
	return ""
}

type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           int
}

func CORS(config CORSConfig) fiber.Handler {
	allowOrigins := strings.Join(config.AllowOrigins, ",")
	if len(config.AllowOrigins) == 0 {
		allowOrigins = "*"
	}

	allowMethods := strings.Join(config.AllowMethods, ",")
	if len(config.AllowMethods) == 0 {
		allowMethods = "GET,POST,PUT,DELETE,OPTIONS"
	}

	allowHeaders := strings.Join(config.AllowHeaders, ",")
	if len(config.AllowHeaders) == 0 {
		allowHeaders = "Origin,Content-Type,Accept,Authorization,X-Request-ID"
	}

	return func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", allowOrigins)
		c.Set("Access-Control-Allow-Methods", allowMethods)
		c.Set("Access-Control-Allow-Headers", allowHeaders)

		if config.AllowCredentials {
			c.Set("Access-Control-Allow-Credentials", "true")
		}

		if config.MaxAge > 0 {
			c.Set("Access-Control-Max-Age", string(rune(config.MaxAge)))
		}

		if c.Method() == "OPTIONS" {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}
