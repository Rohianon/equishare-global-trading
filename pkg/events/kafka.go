package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type KafkaPublisher struct {
	writers map[string]*kafka.Writer
	brokers []string
}

func NewKafkaPublisher(brokers []string) *KafkaPublisher {
	return &KafkaPublisher{
		writers: make(map[string]*kafka.Writer),
		brokers: brokers,
	}
}

func (p *KafkaPublisher) getWriter(topic string) *kafka.Writer {
	if w, ok := p.writers[topic]; ok {
		return w
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(p.brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
	}
	p.writers[topic] = w
	return w
}

func (p *KafkaPublisher) Publish(ctx context.Context, topic string, event *Event) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Time.IsZero() {
		event.Time = time.Now()
	}

	// Start producer span
	tracer := otel.Tracer("kafka-producer")
	ctx, span := tracer.Start(ctx, topic+" publish",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination.name", topic),
			attribute.String("messaging.operation", "publish"),
			attribute.String("messaging.message.id", event.ID),
			attribute.String("event.type", event.Type),
		),
	)
	defer span.End()

	data, err := json.Marshal(event)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Inject trace context into message headers
	headers := make([]kafka.Header, 0)
	carrier := &kafkaHeaderCarrier{headers: &headers}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	writer := p.getWriter(topic)
	err = writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(event.ID),
		Value:   data,
		Headers: headers,
	})
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to publish event: %w", err)
	}

	span.SetAttributes(attribute.Int("messaging.message.body.size", len(data)))
	return nil
}

func (p *KafkaPublisher) Close() error {
	for _, w := range p.writers {
		if err := w.Close(); err != nil {
			return err
		}
	}
	return nil
}

type KafkaSubscriber struct {
	brokers []string
	groupID string
	readers []*kafka.Reader
}

func NewKafkaSubscriber(brokers []string, groupID string) *KafkaSubscriber {
	return &KafkaSubscriber{
		brokers: brokers,
		groupID: groupID,
		readers: make([]*kafka.Reader, 0),
	}
}

func (s *KafkaSubscriber) Subscribe(ctx context.Context, topic string, handler func(*Event) error) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  s.brokers,
		Topic:    topic,
		GroupID:  s.groupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	s.readers = append(s.readers, reader)

	go func() {
		tracer := otel.Tracer("kafka-consumer")
		propagator := otel.GetTextMapPropagator()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := reader.ReadMessage(ctx)
				if err != nil {
					continue
				}

				// Extract trace context from message headers
				carrier := &kafkaHeaderCarrier{headers: &msg.Headers}
				msgCtx := propagator.Extract(ctx, carrier)

				// Start consumer span
				msgCtx, span := tracer.Start(msgCtx, topic+" receive",
					trace.WithSpanKind(trace.SpanKindConsumer),
					trace.WithAttributes(
						attribute.String("messaging.system", "kafka"),
						attribute.String("messaging.destination.name", topic),
						attribute.String("messaging.operation", "receive"),
						attribute.Int64("messaging.kafka.message.offset", msg.Offset),
						attribute.Int("messaging.kafka.partition", msg.Partition),
						attribute.Int("messaging.message.body.size", len(msg.Value)),
					),
				)

				var event Event
				if err := json.Unmarshal(msg.Value, &event); err != nil {
					span.RecordError(err)
					span.End()
					continue
				}

				span.SetAttributes(
					attribute.String("messaging.message.id", event.ID),
					attribute.String("event.type", event.Type),
				)

				if err := handler(&event); err != nil {
					span.RecordError(err)
				}
				span.End()
			}
		}
	}()

	return nil
}

func (s *KafkaSubscriber) Close() error {
	for _, r := range s.readers {
		if err := r.Close(); err != nil {
			return err
		}
	}
	return nil
}

// kafkaHeaderCarrier implements propagation.TextMapCarrier for Kafka headers
type kafkaHeaderCarrier struct {
	headers *[]kafka.Header
}

func (c *kafkaHeaderCarrier) Get(key string) string {
	for _, h := range *c.headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *kafkaHeaderCarrier) Set(key, value string) {
	headers := *c.headers
	for i, h := range headers {
		if h.Key == key {
			headers = append(headers[:i], headers[i+1:]...)
			break
		}
	}
	*c.headers = append(headers, kafka.Header{Key: key, Value: []byte(value)})
}

func (c *kafkaHeaderCarrier) Keys() []string {
	keys := make([]string, len(*c.headers))
	for i, h := range *c.headers {
		keys[i] = h.Key
	}
	return keys
}
