package events

import (
	"testing"
	"time"
)

func TestEventTopics(t *testing.T) {
	topics := []struct {
		name  string
		topic string
	}{
		{"TopicOrderCreated", TopicOrderCreated},
		{"TopicOrderUpdated", TopicOrderUpdated},
		{"TopicOrderFilled", TopicOrderFilled},
		{"TopicOrderCancelled", TopicOrderCancelled},
		{"TopicPaymentInitiated", TopicPaymentInitiated},
		{"TopicPaymentCompleted", TopicPaymentCompleted},
		{"TopicPaymentFailed", TopicPaymentFailed},
		{"TopicWithdrawalInitiated", TopicWithdrawalInitiated},
		{"TopicWithdrawalCompleted", TopicWithdrawalCompleted},
		{"TopicKYCSubmitted", TopicKYCSubmitted},
		{"TopicKYCVerified", TopicKYCVerified},
		{"TopicKYCRejected", TopicKYCRejected},
		{"TopicPriceUpdate", TopicPriceUpdate},
		{"TopicAlertTriggered", TopicAlertTriggered},
		{"TopicNotificationSend", TopicNotificationSend},
	}

	for _, tt := range topics {
		t.Run(tt.name, func(t *testing.T) {
			if tt.topic == "" {
				t.Errorf("%s should not be empty", tt.name)
			}
			if len(tt.topic) < 10 {
				t.Errorf("%s topic name too short: %s", tt.name, tt.topic)
			}
		})
	}
}

func TestEventStruct(t *testing.T) {
	event := Event{
		ID:     "test-id",
		Type:   "order.created",
		Source: "trading-service",
		Time:   time.Now(),
		Data: map[string]any{
			"order_id": "123",
			"amount":   100.50,
		},
		Metadata: map[string]string{
			"trace_id": "abc-123",
		},
	}

	if event.ID != "test-id" {
		t.Errorf("Event.ID = %v, want test-id", event.ID)
	}
	if event.Type != "order.created" {
		t.Errorf("Event.Type = %v, want order.created", event.Type)
	}
	if event.Source != "trading-service" {
		t.Errorf("Event.Source = %v, want trading-service", event.Source)
	}
	if event.Metadata["trace_id"] != "abc-123" {
		t.Errorf("Event.Metadata[trace_id] = %v, want abc-123", event.Metadata["trace_id"])
	}
}
