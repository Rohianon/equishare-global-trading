package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
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

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	writer := p.getWriter(topic)
	err = writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.ID),
		Value: data,
	})
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

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
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := reader.ReadMessage(ctx)
				if err != nil {
					continue
				}

				var event Event
				if err := json.Unmarshal(msg.Value, &event); err != nil {
					continue
				}

				if err := handler(&event); err != nil {
					continue
				}
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
