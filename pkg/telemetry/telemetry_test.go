package telemetry

import (
	"context"
	"net/http"
	"testing"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestInit_Disabled(t *testing.T) {
	ctx := context.Background()
	cfg := &Config{
		ServiceName: "test-service",
		Enabled:     false,
	}

	provider, err := Init(ctx, cfg)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if provider == nil {
		t.Fatal("provider should not be nil")
	}

	// Shutdown should work even when disabled
	if err := provider.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestTracer(t *testing.T) {
	tracer := Tracer("test-tracer")
	if tracer == nil {
		t.Fatal("tracer should not be nil")
	}
}

func TestStartSpan(t *testing.T) {
	// Set up a simple trace provider for testing
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-span")

	if span == nil {
		t.Fatal("span should not be nil")
	}

	if !span.SpanContext().IsValid() {
		t.Error("span context should be valid")
	}

	span.End()
}

func TestTraceID(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-span")
	defer span.End()

	traceID := TraceID(ctx)
	if traceID == "" {
		t.Error("trace ID should not be empty")
	}

	if len(traceID) != 32 {
		t.Errorf("trace ID should be 32 chars, got %d", len(traceID))
	}
}

func TestSpanID(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-span")
	defer span.End()

	spanID := SpanID(ctx)
	if spanID == "" {
		t.Error("span ID should not be empty")
	}

	if len(spanID) != 16 {
		t.Errorf("span ID should be 16 chars, got %d", len(spanID))
	}
}

func TestSpanFromContext(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-span")
	defer span.End()

	retrieved := SpanFromContext(ctx)
	if retrieved != span {
		t.Error("retrieved span should match created span")
	}
}

func TestWrapHTTPClient(t *testing.T) {
	client := WrapHTTPClient(nil)
	if client == nil {
		t.Fatal("client should not be nil")
	}

	if client.Transport == nil {
		t.Error("transport should not be nil")
	}
}

func TestNewTracedHTTPClient(t *testing.T) {
	client := NewTracedHTTPClient()
	if client == nil {
		t.Fatal("client should not be nil")
	}

	if client.Transport == nil {
		t.Error("transport should not be nil")
	}
}

func TestKafkaHeaderCarrier(t *testing.T) {
	headers := []kafka.Header{
		{Key: "test-key", Value: []byte("test-value")},
	}
	carrier := KafkaHeaderCarrier{Headers: &headers}

	// Test Get
	value := carrier.Get("test-key")
	if value != "test-value" {
		t.Errorf("expected 'test-value', got '%s'", value)
	}

	// Test Get non-existent
	value = carrier.Get("non-existent")
	if value != "" {
		t.Errorf("expected empty string, got '%s'", value)
	}

	// Test Set
	carrier.Set("new-key", "new-value")
	value = carrier.Get("new-key")
	if value != "new-value" {
		t.Errorf("expected 'new-value', got '%s'", value)
	}

	// Test Keys
	keys := carrier.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestInjectExtractTraceContext(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer tp.Shutdown(context.Background())

	// Create a span
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-span")
	defer span.End()

	originalTraceID := TraceID(ctx)

	// Inject into headers
	headers := make([]kafka.Header, 0)
	InjectTraceContext(ctx, &headers)

	if len(headers) == 0 {
		t.Error("headers should not be empty after injection")
	}

	// Extract from headers
	newCtx := ExtractTraceContext(context.Background(), headers)

	// Start a new span with extracted context
	newCtx, newSpan := StartSpan(newCtx, "child-span")
	defer newSpan.End()

	extractedTraceID := TraceID(newCtx)
	if extractedTraceID != originalTraceID {
		t.Errorf("trace ID should be preserved: expected %s, got %s", originalTraceID, extractedTraceID)
	}
}

func TestStartProducerSpan(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	ctx := context.Background()
	ctx, span := StartProducerSpan(ctx, "test-topic")

	if span == nil {
		t.Fatal("span should not be nil")
	}

	span.End()
}

func TestStartConsumerSpan(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	ctx := context.Background()
	ctx, span := StartConsumerSpan(ctx, "test-topic", 0, 100)

	if span == nil {
		t.Fatal("span should not be nil")
	}

	span.End()
}

func TestWrapHTTPClient_Existing(t *testing.T) {
	original := &http.Client{}
	wrapped := WrapHTTPClient(original)

	if wrapped != original {
		t.Error("should return the same client instance")
	}

	if wrapped.Transport == nil {
		t.Error("transport should be wrapped")
	}
}
