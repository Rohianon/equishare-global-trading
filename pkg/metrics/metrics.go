package metrics

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// =============================================================================
// Prometheus Metrics
// =============================================================================
// This package provides Prometheus metrics collection for Go services.
// It includes:
// - HTTP request metrics (count, duration, response size)
// - Go runtime metrics (memory, goroutines, GC)
// - Custom business metrics
// =============================================================================

var (
	// Default registry
	registry = prometheus.NewRegistry()

	// HTTP metrics
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"service", "method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"service", "method", "path"},
	)

	httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"service", "method", "path"},
	)

	// Database metrics
	dbPoolConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_pool_connections_used",
			Help: "Number of database connections in use",
		},
		[]string{"service"},
	)

	dbPoolConnectionsMax = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_pool_connections_max",
			Help: "Maximum number of database connections",
		},
		[]string{"service"},
	)

	dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
		},
		[]string{"service", "query_type"},
	)

	// Kafka metrics
	kafkaMessagesProduced = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_produced_total",
			Help: "Total number of Kafka messages produced",
		},
		[]string{"service", "topic"},
	)

	kafkaMessagesConsumed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_consumed_total",
			Help: "Total number of Kafka messages consumed",
		},
		[]string{"service", "topic", "consumer_group"},
	)

	kafkaConsumerLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_consumer_lag",
			Help: "Kafka consumer lag",
		},
		[]string{"service", "topic", "consumer_group", "partition"},
	)

	// Business metrics
	paymentTransactions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_transactions_total",
			Help: "Total number of payment transactions",
		},
		[]string{"service", "type", "status", "provider"},
	)

	tradingOrders = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trading_orders_total",
			Help: "Total number of trading orders",
		},
		[]string{"service", "side", "type", "status"},
	)

	activeUsers = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_users",
			Help: "Number of active users (sessions)",
		},
		[]string{"service"},
	)
)

func init() {
	// Register default Go collectors
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// Register HTTP metrics
	registry.MustRegister(httpRequestsTotal)
	registry.MustRegister(httpRequestDuration)
	registry.MustRegister(httpResponseSize)

	// Register database metrics
	registry.MustRegister(dbPoolConnections)
	registry.MustRegister(dbPoolConnectionsMax)
	registry.MustRegister(dbQueryDuration)

	// Register Kafka metrics
	registry.MustRegister(kafkaMessagesProduced)
	registry.MustRegister(kafkaMessagesConsumed)
	registry.MustRegister(kafkaConsumerLag)

	// Register business metrics
	registry.MustRegister(paymentTransactions)
	registry.MustRegister(tradingOrders)
	registry.MustRegister(activeUsers)
}

// Registry returns the prometheus registry
func Registry() *prometheus.Registry {
	return registry
}

// Handler returns a Fiber handler for the /metrics endpoint
func Handler() fiber.Handler {
	return adaptor.HTTPHandler(promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))
}

// =============================================================================
// Middleware
// =============================================================================

// Config holds metrics middleware configuration
type Config struct {
	ServiceName string
	SkipPaths   []string
}

// Middleware returns Fiber middleware that records HTTP metrics
func Middleware(cfg Config) fiber.Handler {
	skipPaths := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *fiber.Ctx) error {
		// Skip metrics for certain paths
		if skipPaths[c.Path()] {
			return c.Next()
		}

		start := time.Now()

		// Process request
		err := c.Next()

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Response().StatusCode())
		method := c.Method()
		path := c.Route().Path

		httpRequestsTotal.WithLabelValues(cfg.ServiceName, method, path, status).Inc()
		httpRequestDuration.WithLabelValues(cfg.ServiceName, method, path).Observe(duration)
		httpResponseSize.WithLabelValues(cfg.ServiceName, method, path).Observe(float64(len(c.Response().Body())))

		return err
	}
}

// =============================================================================
// Metric Recording Functions
// =============================================================================

// RecordDBPoolStats records database connection pool statistics
func RecordDBPoolStats(service string, used, max int) {
	dbPoolConnections.WithLabelValues(service).Set(float64(used))
	dbPoolConnectionsMax.WithLabelValues(service).Set(float64(max))
}

// RecordDBQuery records database query duration
func RecordDBQuery(service, queryType string, duration time.Duration) {
	dbQueryDuration.WithLabelValues(service, queryType).Observe(duration.Seconds())
}

// RecordKafkaMessageProduced records a Kafka message production
func RecordKafkaMessageProduced(service, topic string) {
	kafkaMessagesProduced.WithLabelValues(service, topic).Inc()
}

// RecordKafkaMessageConsumed records a Kafka message consumption
func RecordKafkaMessageConsumed(service, topic, consumerGroup string) {
	kafkaMessagesConsumed.WithLabelValues(service, topic, consumerGroup).Inc()
}

// RecordKafkaConsumerLag records Kafka consumer lag
func RecordKafkaConsumerLag(service, topic, consumerGroup string, partition int, lag int64) {
	kafkaConsumerLag.WithLabelValues(service, topic, consumerGroup, strconv.Itoa(partition)).Set(float64(lag))
}

// RecordPaymentTransaction records a payment transaction
func RecordPaymentTransaction(service, txType, status, provider string) {
	paymentTransactions.WithLabelValues(service, txType, status, provider).Inc()
}

// RecordTradingOrder records a trading order
func RecordTradingOrder(service, side, orderType, status string) {
	tradingOrders.WithLabelValues(service, side, orderType, status).Inc()
}

// SetActiveUsers sets the number of active users
func SetActiveUsers(service string, count int) {
	activeUsers.WithLabelValues(service).Set(float64(count))
}

// =============================================================================
// Custom Metric Registration
// =============================================================================

// RegisterCounter registers a custom counter metric
func RegisterCounter(name, help string, labels []string) *prometheus.CounterVec {
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, labels)
	registry.MustRegister(counter)
	return counter
}

// RegisterGauge registers a custom gauge metric
func RegisterGauge(name, help string, labels []string) *prometheus.GaugeVec {
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	}, labels)
	registry.MustRegister(gauge)
	return gauge
}

// RegisterHistogram registers a custom histogram metric
func RegisterHistogram(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: buckets,
	}, labels)
	registry.MustRegister(histogram)
	return histogram
}
