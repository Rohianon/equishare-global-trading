package database

import (
	"testing"
)

func TestConfig_ConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "basic config",
			config: Config{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "password",
				Database: "testdb",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=testdb sslmode=disable",
		},
		{
			name: "production config",
			config: Config{
				Host:     "db.example.com",
				Port:     5433,
				User:     "app_user",
				Password: "secret123",
				Database: "production",
				SSLMode:  "require",
			},
			expected: "host=db.example.com port=5433 user=app_user password=secret123 dbname=production sslmode=require",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ConnectionString()
			if got != tt.expected {
				t.Errorf("ConnectionString() = %v, want %v", got, tt.expected)
			}
		})
	}
}
