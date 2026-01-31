package telemetry

import (
	"context"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// KafkaHeaderCarrier implements propagation.TextMapCarrier for Kafka headers
type KafkaHeaderCarrier struct {
	Headers *[]kafka.Header
}

// Get returns the value for the given key
func (c KafkaHeaderCarrier) Get(key string) string {
	for _, h := range *c.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

// Set sets the value for the given key
func (c KafkaHeaderCarrier) Set(key, value string) {
	// Remove existing header with same key
	headers := *c.Headers
	for i, h := range headers {
		if h.Key == key {
			headers = append(headers[:i], headers[i+1:]...)
			break
		}
	}
	*c.Headers = append(headers, kafka.Header{Key: key, Value: []byte(value)})
}

// Keys returns all keys in the carrier
func (c KafkaHeaderCarrier) Keys() []string {
	keys := make([]string, len(*c.Headers))
	for i, h := range *c.Headers {
		keys[i] = h.Key
	}
	return keys
}

// InjectTraceContext injects the trace context into Kafka message headers
func InjectTraceContext(ctx context.Context, headers *[]kafka.Header) {
	carrier := KafkaHeaderCarrier{Headers: headers}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// ExtractTraceContext extracts the trace context from Kafka message headers
func ExtractTraceContext(ctx context.Context, headers []kafka.Header) context.Context {
	carrier := KafkaHeaderCarrier{Headers: &headers}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// StartProducerSpan starts a span for producing a Kafka message
func StartProducerSpan(ctx context.Context, topic string) (context.Context, trace.Span) {
	tracer := otel.Tracer("kafka-producer")
	ctx, span := tracer.Start(ctx, topic+" publish",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination.name", topic),
			attribute.String("messaging.operation", "publish"),
		),
	)
	return ctx, span
}

// StartConsumerSpan starts a span for consuming a Kafka message
func StartConsumerSpan(ctx context.Context, topic string, partition int, offset int64) (context.Context, trace.Span) {
	tracer := otel.Tracer("kafka-consumer")
	ctx, span := tracer.Start(ctx, topic+" receive",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination.name", topic),
			attribute.String("messaging.operation", "receive"),
			attribute.Int64("messaging.kafka.message.offset", offset),
			attribute.Int("messaging.kafka.partition", partition),
		),
	)
	return ctx, span
}

// SetMessageAttributes sets common message attributes on a span
func SetMessageAttributes(ctx context.Context, key string, size int) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("messaging.message.id", key),
		attribute.Int("messaging.message.body.size", size),
	)
}
