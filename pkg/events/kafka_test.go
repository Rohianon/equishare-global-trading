package events

import (
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestNewKafkaPublisher(t *testing.T) {
	brokers := []string{"localhost:9092", "localhost:9093"}
	publisher := NewKafkaPublisher(brokers)

	if publisher == nil {
		t.Fatal("NewKafkaPublisher should not return nil")
	}
	if len(publisher.brokers) != 2 {
		t.Errorf("brokers length = %d, want 2", len(publisher.brokers))
	}
	if publisher.writers == nil {
		t.Error("writers map should be initialized")
	}
}

func TestNewKafkaSubscriber(t *testing.T) {
	brokers := []string{"localhost:9092"}
	groupID := "test-group"
	subscriber := NewKafkaSubscriber(brokers, groupID)

	if subscriber == nil {
		t.Fatal("NewKafkaSubscriber should not return nil")
	}
	if subscriber.groupID != groupID {
		t.Errorf("groupID = %s, want %s", subscriber.groupID, groupID)
	}
	if subscriber.readers == nil {
		t.Error("readers slice should be initialized")
	}
}

func TestKafkaPublisher_getWriter(t *testing.T) {
	publisher := NewKafkaPublisher([]string{"localhost:9092"})
	defer publisher.Close()

	topic := "test-topic"
	writer := publisher.getWriter(topic)

	if writer == nil {
		t.Fatal("getWriter should return a writer")
	}

	writer2 := publisher.getWriter(topic)
	if writer != writer2 {
		t.Error("getWriter should return the same writer for the same topic")
	}
}

func TestKafkaPublisherImplementsInterface(t *testing.T) {
	var _ Publisher = (*KafkaPublisher)(nil)
}

func TestKafkaSubscriberImplementsInterface(t *testing.T) {
	var _ Subscriber = (*KafkaSubscriber)(nil)
}

func TestEventWithDefaults(t *testing.T) {
	event := &Event{
		EventType: "test.event.v1",
		Source:    "test-service",
		Payload:   map[string]string{"key": "value"},
	}

	if event.EventID != "" {
		t.Error("Event.EventID should be empty initially")
	}
	if !event.OccurredAt.IsZero() {
		t.Error("Event.OccurredAt should be zero initially")
	}

	event.EventID = "generated-id"
	event.OccurredAt = time.Now()

	if event.EventID == "" {
		t.Error("Event.EventID should be set")
	}
	if event.OccurredAt.IsZero() {
		t.Error("Event.OccurredAt should be set")
	}
}

func TestKafkaHeaderCarrier(t *testing.T) {
	headers := make([]kafka.Header, 0)
	carrier := &kafkaHeaderCarrier{headers: &headers}

	// Test Set
	carrier.Set("key1", "value1")
	if carrier.Get("key1") != "value1" {
		t.Errorf("Get(key1) = %v, want value1", carrier.Get("key1"))
	}

	// Test overwrite
	carrier.Set("key1", "value2")
	if carrier.Get("key1") != "value2" {
		t.Errorf("After overwrite, Get(key1) = %v, want value2", carrier.Get("key1"))
	}

	// Test Keys
	carrier.Set("key2", "value3")
	keys := carrier.Keys()
	if len(keys) != 2 {
		t.Errorf("Keys() length = %d, want 2", len(keys))
	}

	// Test Get non-existent key
	if carrier.Get("nonexistent") != "" {
		t.Error("Get for non-existent key should return empty string")
	}
}
