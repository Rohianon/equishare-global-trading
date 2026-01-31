package metrics

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestHandler(t *testing.T) {
	app := fiber.New()
	app.Get("/metrics", Handler())

	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Status = %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check for Go runtime metrics
	if !strings.Contains(bodyStr, "go_goroutines") {
		t.Error("Should contain go_goroutines metric")
	}

	// Check for process metrics
	if !strings.Contains(bodyStr, "process_resident_memory_bytes") {
		t.Error("Should contain process_resident_memory_bytes metric")
	}
}

func TestMiddleware(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware(Config{
		ServiceName: "test-service",
		SkipPaths:   []string{"/health"},
	}))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("healthy")
	})
	app.Get("/metrics", Handler())

	// Make a request to /api/test
	req := httptest.NewRequest("GET", "/api/test", nil)
	resp, _ := app.Test(req)
	resp.Body.Close()

	// Make a request to /health (should be skipped)
	req = httptest.NewRequest("GET", "/health", nil)
	resp, _ = app.Test(req)
	resp.Body.Close()

	// Check metrics
	req = httptest.NewRequest("GET", "/metrics", nil)
	resp, _ = app.Test(req)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Should have recorded /api/test
	if !strings.Contains(bodyStr, "http_requests_total") {
		t.Error("Should contain http_requests_total metric")
	}

	// Verify the metric has our service name
	if !strings.Contains(bodyStr, "test-service") {
		t.Error("Should contain test-service label")
	}
}

func TestRecordDBPoolStats(t *testing.T) {
	RecordDBPoolStats("test-service", 5, 10)

	app := fiber.New()
	app.Get("/metrics", Handler())

	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "db_pool_connections_used") {
		t.Error("Should contain db_pool_connections_used metric")
	}
	if !strings.Contains(bodyStr, "db_pool_connections_max") {
		t.Error("Should contain db_pool_connections_max metric")
	}
}

func TestRecordDBQuery(t *testing.T) {
	RecordDBQuery("test-service", "select", 50*time.Millisecond)

	app := fiber.New()
	app.Get("/metrics", Handler())

	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "db_query_duration_seconds") {
		t.Error("Should contain db_query_duration_seconds metric")
	}
}

func TestRecordKafkaMetrics(t *testing.T) {
	RecordKafkaMessageProduced("test-service", "test-topic")
	RecordKafkaMessageConsumed("test-service", "test-topic", "test-group")
	RecordKafkaConsumerLag("test-service", "test-topic", "test-group", 0, 100)

	app := fiber.New()
	app.Get("/metrics", Handler())

	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "kafka_messages_produced_total") {
		t.Error("Should contain kafka_messages_produced_total metric")
	}
	if !strings.Contains(bodyStr, "kafka_messages_consumed_total") {
		t.Error("Should contain kafka_messages_consumed_total metric")
	}
	if !strings.Contains(bodyStr, "kafka_consumer_lag") {
		t.Error("Should contain kafka_consumer_lag metric")
	}
}

func TestRecordBusinessMetrics(t *testing.T) {
	RecordPaymentTransaction("payment-service", "deposit", "success", "mpesa")
	RecordTradingOrder("trading-service", "buy", "market", "filled")
	SetActiveUsers("auth-service", 100)

	app := fiber.New()
	app.Get("/metrics", Handler())

	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "payment_transactions_total") {
		t.Error("Should contain payment_transactions_total metric")
	}
	if !strings.Contains(bodyStr, "trading_orders_total") {
		t.Error("Should contain trading_orders_total metric")
	}
	if !strings.Contains(bodyStr, "active_users") {
		t.Error("Should contain active_users metric")
	}
}

func TestCustomMetricRegistration(t *testing.T) {
	counter := RegisterCounter("custom_counter_test", "Test counter", []string{"label1"})
	counter.WithLabelValues("value1").Inc()

	gauge := RegisterGauge("custom_gauge_test", "Test gauge", []string{"label2"})
	gauge.WithLabelValues("value2").Set(42)

	histogram := RegisterHistogram("custom_histogram_test", "Test histogram", []string{"label3"}, []float64{0.1, 0.5, 1.0})
	histogram.WithLabelValues("value3").Observe(0.25)

	app := fiber.New()
	app.Get("/metrics", Handler())

	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "custom_counter_test") {
		t.Error("Should contain custom_counter_test metric")
	}
	if !strings.Contains(bodyStr, "custom_gauge_test") {
		t.Error("Should contain custom_gauge_test metric")
	}
	if !strings.Contains(bodyStr, "custom_histogram_test") {
		t.Error("Should contain custom_histogram_test metric")
	}
}
