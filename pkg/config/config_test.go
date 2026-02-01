package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_WithConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  host: "127.0.0.1"
  port: 9000
database:
  host: "db.example.com"
  port: 5433
  user: "testuser"
  password: "testpass"
  database: "testdb"
  ssl_mode: "require"
redis:
  host: "redis.example.com"
  port: 6380
  password: "redispass"
  db: 1
kafka:
  brokers:
    - "kafka1:9092"
    - "kafka2:9092"
  group_id: "test-group"
telemetry:
  service_name: "test-service"
  collector_url: "http://collector:4317"
  enabled: true
jwt:
  secret: "super-secret-jwt-key-that-is-at-least-32-chars"
  issuer: "test-issuer"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(origDir)

	cfg, err := Load("config")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %v, want 127.0.0.1", cfg.Server.Host)
	}
	if cfg.Server.Port != 9000 {
		t.Errorf("Server.Port = %v, want 9000", cfg.Server.Port)
	}
	if cfg.Database.Host != "db.example.com" {
		t.Errorf("Database.Host = %v, want db.example.com", cfg.Database.Host)
	}
	if cfg.Database.SSLMode != "require" {
		t.Errorf("Database.SSLMode = %v, want require", cfg.Database.SSLMode)
	}
	if len(cfg.Kafka.Brokers) != 2 {
		t.Errorf("Kafka.Brokers length = %v, want 2", len(cfg.Kafka.Brokers))
	}
	if !cfg.Telemetry.Enabled {
		t.Error("Telemetry.Enabled should be true")
	}
	if cfg.JWT.Issuer != "test-issuer" {
		t.Errorf("JWT.Issuer = %v, want test-issuer", cfg.JWT.Issuer)
	}
}

func TestLoad_WithDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(origDir)

	cfg, err := Load("nonexistent")
	if err != nil {
		t.Fatalf("Load() should not error when config file not found: %v", err)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Default Server.Host = %v, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Default Server.Port = %v, want 8080", cfg.Server.Port)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("Default Database.Host = %v, want localhost", cfg.Database.Host)
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Default Database.Port = %v, want 5432", cfg.Database.Port)
	}
	if cfg.JWT.Issuer != "equishare" {
		t.Errorf("Default JWT.Issuer = %v, want equishare", cfg.JWT.Issuer)
	}
}

func TestLoad_WithEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(origDir)

	os.Setenv("EQUISHARE_SERVER_PORT", "3000")
	defer os.Unsetenv("EQUISHARE_SERVER_PORT")

	cfg, err := Load("nonexistent")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("Server.Port = %v, want 3000 (from env)", cfg.Server.Port)
	}
}

func TestLoad_WithMultipleEnvOverrides(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Set multiple env vars
	os.Setenv("EQUISHARE_SERVER_PORT", "4000")
	os.Setenv("EQUISHARE_DATABASE_USER", "envuser")
	os.Setenv("EQUISHARE_DATABASE_PASSWORD", "envpass")
	os.Setenv("EQUISHARE_JWT_SECRET", "env-secret-key-that-is-long-enough")
	defer func() {
		os.Unsetenv("EQUISHARE_SERVER_PORT")
		os.Unsetenv("EQUISHARE_DATABASE_USER")
		os.Unsetenv("EQUISHARE_DATABASE_PASSWORD")
		os.Unsetenv("EQUISHARE_JWT_SECRET")
	}()

	cfg, err := Load("nonexistent")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 4000 {
		t.Errorf("Server.Port = %v, want 4000", cfg.Server.Port)
	}
	if cfg.Database.User != "envuser" {
		t.Errorf("Database.User = %v, want envuser", cfg.Database.User)
	}
	if cfg.Database.Password != "envpass" {
		t.Errorf("Database.Password = %v, want envpass", cfg.Database.Password)
	}
}

func TestValidate_RequiredDatabase(t *testing.T) {
	cfg := &Config{}
	req := Requirements{Database: true}

	err := Validate(cfg, req)
	if err == nil {
		t.Error("Validate() should return error for missing database config")
	}

	errs, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}

	// Should have errors for user, password, database
	if len(errs) != 3 {
		t.Errorf("expected 3 validation errors, got %d: %v", len(errs), errs)
	}
}

func TestValidate_RequiredJWT(t *testing.T) {
	cfg := &Config{}
	req := Requirements{JWT: true}

	err := Validate(cfg, req)
	if err == nil {
		t.Error("Validate() should return error for missing JWT secret")
	}

	// Test with short secret
	cfg.JWT.Secret = "short"
	err = Validate(cfg, req)
	if err == nil {
		t.Error("Validate() should return error for short JWT secret")
	}

	// Test with valid secret
	cfg.JWT.Secret = "this-is-a-valid-secret-key-that-is-long-enough"
	err = Validate(cfg, req)
	if err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestValidate_RequiredMPesa(t *testing.T) {
	cfg := &Config{}
	req := Requirements{MPesa: true}

	err := Validate(cfg, req)
	if err == nil {
		t.Error("Validate() should return error for missing MPesa config")
	}

	cfg.MPesa.ConsumerKey = "key"
	cfg.MPesa.ConsumerSecret = "secret"
	cfg.MPesa.ShortCode = "123456"

	err = Validate(cfg, req)
	if err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestValidate_RequiredAlpaca(t *testing.T) {
	cfg := &Config{}
	req := Requirements{Alpaca: true}

	err := Validate(cfg, req)
	if err == nil {
		t.Error("Validate() should return error for missing Alpaca config")
	}

	cfg.Alpaca.APIKey = "key"
	cfg.Alpaca.APISecret = "secret"

	err = Validate(cfg, req)
	if err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestValidate_NoRequirements(t *testing.T) {
	cfg := &Config{}
	req := Requirements{}

	err := Validate(cfg, req)
	if err != nil {
		t.Errorf("Validate() with no requirements should not error: %v", err)
	}
}

func TestLoadWithValidation(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
database:
  user: "testuser"
  password: "testpass"
  database: "testdb"
jwt:
  secret: "super-secret-jwt-key-that-is-at-least-32-chars"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(origDir)

	req := Requirements{Database: true, JWT: true}
	cfg, err := LoadWithValidation("config", req)
	if err != nil {
		t.Fatalf("LoadWithValidation() error = %v", err)
	}

	if cfg.Database.User != "testuser" {
		t.Errorf("Database.User = %v, want testuser", cfg.Database.User)
	}
}

func TestLoadWithValidation_Fails(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(origDir)

	req := Requirements{Database: true}
	_, err := LoadWithValidation("nonexistent", req)
	if err == nil {
		t.Error("LoadWithValidation() should return error for missing required config")
	}
}

func TestMustLoad_Panics(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	if err := os.WriteFile(configPath, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(origDir)

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustLoad() should panic on invalid config")
		}
	}()

	MustLoad("invalid")
}

func TestDatabaseConfig_DSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		Database: "testdb",
		SSLMode:  "disable",
	}

	expected := "host=localhost port=5432 user=user password=pass dbname=testdb sslmode=disable"
	if dsn := cfg.DSN(); dsn != expected {
		t.Errorf("DSN() = %v, want %v", dsn, expected)
	}
}

func TestRedisConfig_Addr(t *testing.T) {
	cfg := RedisConfig{
		Host: "redis.example.com",
		Port: 6379,
	}

	expected := "redis.example.com:6379"
	if addr := cfg.Addr(); addr != expected {
		t.Errorf("Addr() = %v, want %v", addr, expected)
	}
}

func TestEnvironmentHelpers(t *testing.T) {
	// Test default (development)
	os.Unsetenv("EQUISHARE_ENV")
	if !IsDevelopment() {
		t.Error("IsDevelopment() should be true by default")
	}
	if IsProduction() {
		t.Error("IsProduction() should be false by default")
	}
	if Environment() != "development" {
		t.Errorf("Environment() = %v, want development", Environment())
	}

	// Test production
	os.Setenv("EQUISHARE_ENV", "production")
	defer os.Unsetenv("EQUISHARE_ENV")

	if IsDevelopment() {
		t.Error("IsDevelopment() should be false in production")
	}
	if !IsProduction() {
		t.Error("IsProduction() should be true in production")
	}
	if Environment() != "production" {
		t.Errorf("Environment() = %v, want production", Environment())
	}
}

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{Field: "test.field", Message: "is required"}
	expected := "test.field: is required"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}

func TestValidationErrors_Error(t *testing.T) {
	var errs ValidationErrors
	if errs.Error() != "no validation errors" {
		t.Errorf("Empty ValidationErrors.Error() = %v", errs.Error())
	}

	errs = append(errs, ValidationError{"field1", "error1"})
	errs = append(errs, ValidationError{"field2", "error2"})

	errStr := errs.Error()
	if errStr == "no validation errors" {
		t.Error("ValidationErrors.Error() should not be empty")
	}
}
