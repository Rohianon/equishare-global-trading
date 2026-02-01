package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// =============================================================================
// Configuration Structures
// =============================================================================
// Config holds all service configuration. Configuration is loaded with the
// following precedence (highest to lowest):
//   1. Environment variables (EQUISHARE_* prefix)
//   2. Local config file (config.yaml)
//   3. Default values
// =============================================================================

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Kafka     KafkaConfig     `mapstructure:"kafka"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Auth      AuthConfig      `mapstructure:"auth"`
	MPesa     MPesaConfig     `mapstructure:"mpesa"`
	KYC       KYCConfig       `mapstructure:"kyc"`
	SMS       SMSConfig       `mapstructure:"sms"`
	Alpaca    AlpacaConfig    `mapstructure:"alpaca"`
}

type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type KafkaConfig struct {
	Brokers       []string `mapstructure:"brokers"`
	GroupID       string   `mapstructure:"group_id"`
	ClientID      string   `mapstructure:"client_id"`
	SecurityProto string   `mapstructure:"security_protocol"`
	SASLMechanism string   `mapstructure:"sasl_mechanism"`
	SASLUsername  string   `mapstructure:"sasl_username"`
	SASLPassword  string   `mapstructure:"sasl_password"`
}

type TelemetryConfig struct {
	ServiceName  string  `mapstructure:"service_name"`
	CollectorURL string  `mapstructure:"collector_url"`
	Enabled      bool    `mapstructure:"enabled"`
	SampleRate   float64 `mapstructure:"sample_rate"`
}

type JWTConfig struct {
	Secret             string        `mapstructure:"secret"`
	AccessTokenExpiry  time.Duration `mapstructure:"access_token_expiry"`
	RefreshTokenExpiry time.Duration `mapstructure:"refresh_token_expiry"`
	Issuer             string        `mapstructure:"issuer"`
}

type AuthConfig struct {
	TOTPIssuer       string `mapstructure:"totp_issuer"`
	PasswordMinLen   int    `mapstructure:"password_min_length"`
	MaxLoginAttempts int    `mapstructure:"max_login_attempts"`
	LockoutDuration  time.Duration `mapstructure:"lockout_duration"`
}

type MPesaConfig struct {
	Environment      string `mapstructure:"environment"` // sandbox or production
	ConsumerKey      string `mapstructure:"consumer_key"`
	ConsumerSecret   string `mapstructure:"consumer_secret"`
	PassKey          string `mapstructure:"pass_key"`
	ShortCode        string `mapstructure:"short_code"`
	InitiatorName    string `mapstructure:"initiator_name"`
	SecurityCredPath string `mapstructure:"security_cred_path"`
	CallbackURL      string `mapstructure:"callback_url"`
	TimeoutURL       string `mapstructure:"timeout_url"`
	ResultURL        string `mapstructure:"result_url"`
}

type KYCConfig struct {
	Provider string `mapstructure:"provider"` // smile_id, onfido, etc.
	APIKey   string `mapstructure:"api_key"`
	BaseURL  string `mapstructure:"base_url"`
	Partner  string `mapstructure:"partner_id"`
}

type SMSConfig struct {
	Provider   string `mapstructure:"provider"` // africastalking, twilio
	APIKey     string `mapstructure:"api_key"`
	Username   string `mapstructure:"username"`
	SenderID   string `mapstructure:"sender_id"`
	AccountSID string `mapstructure:"account_sid"`   // Twilio
	AuthToken  string `mapstructure:"auth_token"`    // Twilio
	FromNumber string `mapstructure:"from_number"`   // Twilio
}

type AlpacaConfig struct {
	Environment string `mapstructure:"environment"` // paper or live
	APIKey      string `mapstructure:"api_key"`
	APISecret   string `mapstructure:"api_secret"`
	BaseURL     string `mapstructure:"base_url"`
	DataURL     string `mapstructure:"data_url"`
}

// =============================================================================
// Configuration Loading
// =============================================================================

// Load loads configuration from files and environment variables.
// It first attempts to load .env file (for local development),
// then reads config files, and finally applies environment variable overrides.
func Load(configName string) (*Config, error) {
	// Load .env file if it exists (for local development)
	_ = godotenv.Load()
	_ = godotenv.Load(".env.local")

	v := viper.New()

	v.SetConfigName(configName)
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/equishare/")

	v.SetEnvPrefix("EQUISHARE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// LoadWithValidation loads configuration and validates required fields.
// This should be used in production to ensure all required config is present.
func LoadWithValidation(configName string, requirements Requirements) (*Config, error) {
	cfg, err := Load(configName)
	if err != nil {
		return nil, err
	}

	if err := Validate(cfg, requirements); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// MustLoad loads configuration and panics on error.
// Use this for application startup where missing config is fatal.
func MustLoad(configName string) *Config {
	cfg, err := Load(configName)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// MustLoadWithValidation loads and validates configuration, panics on error.
func MustLoadWithValidation(configName string, requirements Requirements) *Config {
	cfg, err := LoadWithValidation(configName, requirements)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 30*time.Second)
	v.SetDefault("server.write_timeout", 30*time.Second)
	v.SetDefault("server.shutdown_timeout", 30*time.Second)

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "")
	v.SetDefault("database.password", "")
	v.SetDefault("database.database", "")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", time.Hour)
	v.SetDefault("database.conn_max_idle_time", 30*time.Minute)

	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.min_idle_conns", 2)
	v.SetDefault("redis.dial_timeout", 5*time.Second)
	v.SetDefault("redis.read_timeout", 3*time.Second)
	v.SetDefault("redis.write_timeout", 3*time.Second)

	// Kafka defaults
	v.SetDefault("kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("kafka.group_id", "")
	v.SetDefault("kafka.client_id", "")
	v.SetDefault("kafka.security_protocol", "PLAINTEXT")
	v.SetDefault("kafka.sasl_mechanism", "")
	v.SetDefault("kafka.sasl_username", "")
	v.SetDefault("kafka.sasl_password", "")

	// Telemetry defaults
	v.SetDefault("telemetry.enabled", false)
	v.SetDefault("telemetry.sample_rate", 1.0)

	// JWT defaults
	v.SetDefault("jwt.secret", "")
	v.SetDefault("jwt.access_token_expiry", 15*time.Minute)
	v.SetDefault("jwt.refresh_token_expiry", 7*24*time.Hour)
	v.SetDefault("jwt.issuer", "equishare")

	// Auth defaults
	v.SetDefault("auth.totp_issuer", "EquiShare")
	v.SetDefault("auth.password_min_length", 8)
	v.SetDefault("auth.max_login_attempts", 5)
	v.SetDefault("auth.lockout_duration", 15*time.Minute)

	// MPesa defaults
	v.SetDefault("mpesa.environment", "sandbox")
	v.SetDefault("mpesa.consumer_key", "")
	v.SetDefault("mpesa.consumer_secret", "")
	v.SetDefault("mpesa.pass_key", "")
	v.SetDefault("mpesa.short_code", "")
	v.SetDefault("mpesa.initiator_name", "")
	v.SetDefault("mpesa.security_cred_path", "")
	v.SetDefault("mpesa.callback_url", "")
	v.SetDefault("mpesa.timeout_url", "")
	v.SetDefault("mpesa.result_url", "")

	// KYC defaults
	v.SetDefault("kyc.provider", "")
	v.SetDefault("kyc.api_key", "")
	v.SetDefault("kyc.base_url", "")
	v.SetDefault("kyc.partner_id", "")

	// SMS defaults
	v.SetDefault("sms.provider", "")
	v.SetDefault("sms.api_key", "")
	v.SetDefault("sms.username", "")
	v.SetDefault("sms.sender_id", "")
	v.SetDefault("sms.account_sid", "")
	v.SetDefault("sms.auth_token", "")
	v.SetDefault("sms.from_number", "")

	// Alpaca defaults
	v.SetDefault("alpaca.environment", "paper")
	v.SetDefault("alpaca.api_key", "")
	v.SetDefault("alpaca.api_secret", "")
	v.SetDefault("alpaca.base_url", "https://paper-api.alpaca.markets")
	v.SetDefault("alpaca.data_url", "https://data.alpaca.markets")
}

// =============================================================================
// Validation
// =============================================================================

// Requirements specifies which configuration sections are required.
type Requirements struct {
	Database  bool
	Redis     bool
	Kafka     bool
	JWT       bool
	MPesa     bool
	KYC       bool
	SMS       bool
	Alpaca    bool
	Telemetry bool
}

// ValidationError contains details about configuration validation failures.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	var sb strings.Builder
	sb.WriteString("configuration validation failed:\n")
	for _, err := range e {
		sb.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
	}
	return sb.String()
}

// Validate checks that required configuration is present.
func Validate(cfg *Config, req Requirements) error {
	var errors ValidationErrors

	// Database validation
	if req.Database {
		if cfg.Database.User == "" {
			errors = append(errors, ValidationError{"database.user", "required"})
		}
		if cfg.Database.Password == "" {
			errors = append(errors, ValidationError{"database.password", "required"})
		}
		if cfg.Database.Database == "" {
			errors = append(errors, ValidationError{"database.database", "required"})
		}
	}

	// JWT validation
	if req.JWT {
		if cfg.JWT.Secret == "" {
			errors = append(errors, ValidationError{"jwt.secret", "required"})
		}
		if len(cfg.JWT.Secret) < 32 {
			errors = append(errors, ValidationError{"jwt.secret", "must be at least 32 characters"})
		}
	}

	// MPesa validation
	if req.MPesa {
		if cfg.MPesa.ConsumerKey == "" {
			errors = append(errors, ValidationError{"mpesa.consumer_key", "required"})
		}
		if cfg.MPesa.ConsumerSecret == "" {
			errors = append(errors, ValidationError{"mpesa.consumer_secret", "required"})
		}
		if cfg.MPesa.ShortCode == "" {
			errors = append(errors, ValidationError{"mpesa.short_code", "required"})
		}
	}

	// KYC validation
	if req.KYC {
		if cfg.KYC.APIKey == "" {
			errors = append(errors, ValidationError{"kyc.api_key", "required"})
		}
		if cfg.KYC.Provider == "" {
			errors = append(errors, ValidationError{"kyc.provider", "required"})
		}
	}

	// SMS validation
	if req.SMS {
		if cfg.SMS.Provider == "" {
			errors = append(errors, ValidationError{"sms.provider", "required"})
		}
		if cfg.SMS.APIKey == "" && cfg.SMS.AuthToken == "" {
			errors = append(errors, ValidationError{"sms.api_key or sms.auth_token", "at least one required"})
		}
	}

	// Alpaca validation
	if req.Alpaca {
		if cfg.Alpaca.APIKey == "" {
			errors = append(errors, ValidationError{"alpaca.api_key", "required"})
		}
		if cfg.Alpaca.APISecret == "" {
			errors = append(errors, ValidationError{"alpaca.api_secret", "required"})
		}
	}

	// Telemetry validation
	if req.Telemetry && cfg.Telemetry.Enabled {
		if cfg.Telemetry.CollectorURL == "" {
			errors = append(errors, ValidationError{"telemetry.collector_url", "required when telemetry is enabled"})
		}
		if cfg.Telemetry.ServiceName == "" {
			errors = append(errors, ValidationError{"telemetry.service_name", "required when telemetry is enabled"})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// =============================================================================
// Helper Methods
// =============================================================================

// DSN returns the PostgreSQL connection string.
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// RedisAddr returns the Redis connection address.
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IsDevelopment returns true if running in development mode.
func IsDevelopment() bool {
	env := os.Getenv("EQUISHARE_ENV")
	return env == "" || env == "development" || env == "dev"
}

// IsProduction returns true if running in production mode.
func IsProduction() bool {
	env := os.Getenv("EQUISHARE_ENV")
	return env == "production" || env == "prod"
}

// Environment returns the current environment name.
func Environment() string {
	env := os.Getenv("EQUISHARE_ENV")
	if env == "" {
		return "development"
	}
	return env
}
