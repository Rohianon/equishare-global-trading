package middleware

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

	apperrors "github.com/Rohianon/equishare-global-trading/pkg/errors"
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

		logEvent := logger.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Dur("latency", time.Since(start)).
			Str("request_id", GetRequestID(c))

		// Add trace_id if present
		if traceID := GetTraceID(c); traceID != "" {
			logEvent = logEvent.Str("trace_id", traceID)
		}

		logEvent.Msg("request")

		return err
	}
}

// Tracing returns OpenTelemetry tracing middleware for Fiber
func Tracing(serviceName string) fiber.Handler {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(c *fiber.Ctx) error {
		// Extract trace context from incoming headers
		ctx := propagator.Extract(c.UserContext(), propagation.HeaderCarrier(c.GetReqHeaders()))

		// Start a new span
		spanName := c.Method() + " " + c.Path()
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethod(c.Method()),
				semconv.HTTPRoute(c.Route().Path),
				semconv.HTTPURL(c.OriginalURL()),
				semconv.HTTPScheme(c.Protocol()),
				semconv.NetHostName(c.Hostname()),
				semconv.UserAgentOriginal(c.Get("User-Agent")),
			),
		)
		defer span.End()

		// Store trace info in context and locals
		c.SetUserContext(ctx)
		c.Locals("trace_id", span.SpanContext().TraceID().String())
		c.Locals("span_id", span.SpanContext().SpanID().String())

		// Set trace headers in response
		c.Set("X-Trace-ID", span.SpanContext().TraceID().String())

		// Process request
		err := c.Next()

		// Record response status
		status := c.Response().StatusCode()
		span.SetAttributes(semconv.HTTPStatusCode(status))

		if status >= 400 {
			span.SetAttributes(attribute.Bool("error", true))
		}

		if err != nil {
			span.RecordError(err)
		}

		return err
	}
}

// GetTraceID returns the trace ID from the request context
func GetTraceID(c *fiber.Ctx) string {
	if id, ok := c.Locals("trace_id").(string); ok {
		return id
	}
	return ""
}

// GetSpanID returns the span ID from the request context
func GetSpanID(c *fiber.Ctx) string {
	if id, ok := c.Locals("span_id").(string); ok {
		return id
	}
	return ""
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
		// Use user ID if authenticated, otherwise IP
		key := c.IP()
		if userID := GetUserID(c); userID != "" {
			key = "user:" + userID
		}

		rl.mu.Lock()
		v, exists := rl.visitors[key]
		now := time.Now()

		if !exists {
			rl.visitors[key] = &visitor{count: 1, lastSeen: now}
			rl.mu.Unlock()
			setRateLimitHeaders(c, config.Max, config.Max-1, now.Add(config.Duration))
			return c.Next()
		}

		// Reset if window expired
		if time.Since(v.lastSeen) > rl.config.Duration {
			v.count = 1
			v.lastSeen = now
			rl.mu.Unlock()
			setRateLimitHeaders(c, config.Max, config.Max-1, now.Add(config.Duration))
			return c.Next()
		}

		remaining := config.Max - v.count - 1
		resetTime := v.lastSeen.Add(config.Duration)

		if v.count >= rl.config.Max {
			rl.mu.Unlock()
			setRateLimitHeaders(c, config.Max, 0, resetTime)
			return apperrors.ErrRateLimited
		}

		v.count++
		v.lastSeen = now
		rl.mu.Unlock()

		setRateLimitHeaders(c, config.Max, remaining, resetTime)
		return c.Next()
	}
}

func setRateLimitHeaders(c *fiber.Ctx, limit, remaining int, reset time.Time) {
	c.Set("X-RateLimit-Limit", strconv.Itoa(limit))
	c.Set("X-RateLimit-Remaining", strconv.Itoa(max(0, remaining)))
	c.Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
			return apperrors.ErrUnauthorized.WithDetails("Missing authorization header")
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return apperrors.ErrUnauthorized.WithDetails("Invalid authorization header format")
		}

		tokenString := parts[1]

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, apperrors.ErrInvalidToken
			}
			return []byte(jwtSecret), nil
		})

		if err != nil {
			if strings.Contains(err.Error(), "expired") {
				return apperrors.ErrTokenExpired
			}
			return apperrors.ErrInvalidToken
		}

		if !token.Valid {
			return apperrors.ErrInvalidToken
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			return apperrors.ErrInvalidToken.WithDetails("Invalid token claims")
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("phone", claims.Phone)

		return c.Next()
	}
}

// OptionalAuth validates JWT if present but doesn't require it
func OptionalAuth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Next()
		}

		tokenString := parts[1]

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, apperrors.ErrInvalidToken
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Next()
		}

		if claims, ok := token.Claims.(*Claims); ok {
			c.Locals("user_id", claims.UserID)
			c.Locals("phone", claims.Phone)
		}

		return c.Next()
	}
}

func GetUserID(c *fiber.Ctx) string {
	if id, ok := c.Locals("user_id").(string); ok {
		return id
	}
	return ""
}

// =============================================================================
// 2FA Middleware
// =============================================================================

// TwoFactorValidator defines the interface for validating 2FA codes
type TwoFactorValidator interface {
	// Is2FAEnabled checks if 2FA is enabled for a user
	Is2FAEnabled(userID string) (bool, error)
	// ValidateCode validates a TOTP code for a user
	ValidateCode(userID, code string) (bool, error)
	// ValidateRecoveryCode validates and consumes a recovery code
	ValidateRecoveryCode(userID, code string) (bool, error)
}

// Require2FA creates middleware that enforces 2FA for sensitive actions
// The 2FA code should be provided in the X-2FA-Code header
func Require2FA(validator TwoFactorValidator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := GetUserID(c)
		if userID == "" {
			return apperrors.ErrUnauthorized.WithDetails("Authentication required before 2FA")
		}

		// Check if user has 2FA enabled
		enabled, err := validator.Is2FAEnabled(userID)
		if err != nil {
			logger.Error().Err(err).Str("user_id", userID).Msg("Failed to check 2FA status")
			return apperrors.ErrInternal
		}

		// If 2FA is not enabled, allow the request (graceful degradation)
		if !enabled {
			return c.Next()
		}

		// Get 2FA code from header
		code := c.Get("X-2FA-Code")
		if code == "" {
			return apperrors.Err2FARequired
		}

		// Try TOTP code first
		valid, err := validator.ValidateCode(userID, code)
		if err != nil {
			logger.Error().Err(err).Str("user_id", userID).Msg("Failed to validate 2FA code")
			return apperrors.ErrInternal
		}

		if valid {
			c.Locals("2fa_verified", true)
			return c.Next()
		}

		// Try recovery code as fallback
		valid, err = validator.ValidateRecoveryCode(userID, code)
		if err != nil {
			logger.Error().Err(err).Str("user_id", userID).Msg("Failed to validate recovery code")
			return apperrors.ErrInternal
		}

		if valid {
			c.Locals("2fa_verified", true)
			c.Locals("used_recovery_code", true)
			return c.Next()
		}

		return apperrors.ErrInvalid2FACode
	}
}

// Get2FAVerified returns whether 2FA was verified for this request
func Get2FAVerified(c *fiber.Ctx) bool {
	if verified, ok := c.Locals("2fa_verified").(bool); ok {
		return verified
	}
	return false
}

// GetUsedRecoveryCode returns whether a recovery code was used for 2FA
func GetUsedRecoveryCode(c *fiber.Ctx) bool {
	if used, ok := c.Locals("used_recovery_code").(bool); ok {
		return used
	}
	return false
}

// GetPhone returns the phone from the request context
func GetPhone(c *fiber.Ctx) string {
	if phone, ok := c.Locals("phone").(string); ok {
		return phone
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
