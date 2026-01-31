package events

import (
	"testing"
	"time"
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
		Type:   "test.event",
		Source: "test-service",
		Data:   map[string]string{"key": "value"},
	}

	if event.ID != "" {
		t.Error("Event.ID should be empty initially")
	}
	if !event.Time.IsZero() {
		t.Error("Event.Time should be zero initially")
	}

	event.ID = "generated-id"
	event.Time = time.Now()

	if event.ID == "" {
		t.Error("Event.ID should be set")
	}
	if event.Time.IsZero() {
		t.Error("Event.Time should be set")
	}
}
