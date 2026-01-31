package logger

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		level       string
		pretty      bool
	}{
		{"info level pretty", "test-service", "info", true},
		{"debug level json", "test-service", "debug", false},
		{"invalid level defaults to info", "test-service", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.serviceName, tt.level, tt.pretty)

			if Logger.GetLevel() == zerolog.Disabled {
				t.Error("Logger should be enabled after Init")
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	Logger = zerolog.New(&buf).With().Timestamp().Logger()

	Debug().Msg("debug message")
	Info().Msg("info message")
	Warn().Msg("warn message")
	Error().Msg("error message")

	output := buf.String()

	if !strings.Contains(output, "debug message") {
		t.Error("Debug() should log messages")
	}
	if !strings.Contains(output, "info message") {
		t.Error("Info() should log messages")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn() should log messages")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error() should log messages")
	}
}

func TestWithContext(t *testing.T) {
	Init("test", "info", false)

	ctx := context.Background()
	logger := WithContext(ctx)

	if logger.GetLevel() == zerolog.Disabled {
		t.Error("WithContext should return a valid logger")
	}
}
