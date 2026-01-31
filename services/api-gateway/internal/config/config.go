package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all API Gateway configuration
type Config struct {
	// Server settings
	Port        string
	Environment string

	// JWT settings
	JWTSecret string

	// Rate limiting
	RateLimit         int
	RateLimitDuration time.Duration

	// Service URLs
	AuthServiceURL    string
	UserServiceURL    string
	PaymentServiceURL string
	TradingServiceURL string

	// CORS settings
	CORSAllowOrigins []string

	// Telemetry
	OTLPEndpoint string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8000"),
		Environment: getEnv("ENV", "development"),

		JWTSecret: getEnv("JWT_SECRET", "dev-secret-change-in-prod"),

		RateLimit:         getEnvInt("RATE_LIMIT", 100),
		RateLimitDuration: getEnvDuration("RATE_LIMIT_DURATION", time.Minute),

		AuthServiceURL:    getEnv("AUTH_SERVICE_URL", "http://localhost:8001"),
		UserServiceURL:    getEnv("USER_SERVICE_URL", "http://localhost:8002"),
		PaymentServiceURL: getEnv("PAYMENT_SERVICE_URL", "http://localhost:8003"),
		TradingServiceURL: getEnv("TRADING_SERVICE_URL", "http://localhost:8004"),

		CORSAllowOrigins: getEnvSlice("CORS_ALLOW_ORIGINS", []string{"*"}),

		OTLPEndpoint: getEnv("OTLP_ENDPOINT", ""),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}

func getEnvSlice(key string, fallback []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return fallback
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}
