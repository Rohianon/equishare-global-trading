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
