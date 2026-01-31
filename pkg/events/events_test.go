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
		{"TopicOrderFilled", TopicOrderFilled},
		{"TopicOrderPartialFill", TopicOrderPartialFill},
		{"TopicOrderCancelled", TopicOrderCancelled},
		{"TopicOrderRejected", TopicOrderRejected},
		{"TopicPaymentInitiated", TopicPaymentInitiated},
		{"TopicPaymentCompleted", TopicPaymentCompleted},
		{"TopicPaymentFailed", TopicPaymentFailed},
		{"TopicWithdrawalInitiated", TopicWithdrawalInitiated},
		{"TopicWithdrawalCompleted", TopicWithdrawalCompleted},
		{"TopicWithdrawalFailed", TopicWithdrawalFailed},
		{"TopicKYCSubmitted", TopicKYCSubmitted},
		{"TopicKYCVerified", TopicKYCVerified},
		{"TopicKYCRejected", TopicKYCRejected},
		{"TopicUserRegistered", TopicUserRegistered},
		{"TopicUserVerified", TopicUserVerified},
		{"TopicPriceUpdate", TopicPriceUpdate},
		{"TopicMarketOpen", TopicMarketOpen},
		{"TopicMarketClose", TopicMarketClose},
		{"TopicAlertTriggered", TopicAlertTriggered},
		{"TopicNotificationSend", TopicNotificationSend},
	}

	for _, tt := range topics {
		t.Run(tt.name, func(t *testing.T) {
			if tt.topic == "" {
				t.Errorf("%s should not be empty", tt.name)
			}
			// All topics should start with "equishare."
			if len(tt.topic) < 10 || tt.topic[:10] != "equishare." {
				t.Errorf("%s topic should start with 'equishare.': %s", tt.name, tt.topic)
			}
		})
	}
}

func TestAllTopics(t *testing.T) {
	if len(AllTopics) == 0 {
		t.Error("AllTopics should not be empty")
	}

	// Verify all topics in AllTopics are valid
	for _, topic := range AllTopics {
		if topic == "" {
			t.Error("AllTopics contains empty topic")
		}
	}
}

func TestNewEvent(t *testing.T) {
	payload := OrderCreatedPayload{
		OrderID: "order-123",
		UserID:  "user-456",
		Symbol:  "AAPL",
		Side:    "buy",
		Amount:  100.00,
	}

	event := NewEvent(EventTypeOrderCreated, "trading-service", payload)

	if event.EventID == "" {
		t.Error("EventID should be auto-generated")
	}
	if event.EventType != EventTypeOrderCreated {
		t.Errorf("EventType = %v, want %v", event.EventType, EventTypeOrderCreated)
	}
	if event.Source != "trading-service" {
		t.Errorf("Source = %v, want trading-service", event.Source)
	}
	if event.OccurredAt.IsZero() {
		t.Error("OccurredAt should be auto-set")
	}
	if event.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
}

func TestEventWithCorrelationID(t *testing.T) {
	event := NewEvent(EventTypeOrderCreated, "trading-service", nil)
	event.WithCorrelationID("corr-123")

	if event.CorrelationID != "corr-123" {
		t.Errorf("CorrelationID = %v, want corr-123", event.CorrelationID)
	}
}

func TestEventWithMetadata(t *testing.T) {
	event := NewEvent(EventTypeOrderCreated, "trading-service", nil)
	event.WithMetadata("trace_id", "trace-123").
		WithMetadata("request_id", "req-456")

	if event.Metadata["trace_id"] != "trace-123" {
		t.Errorf("Metadata[trace_id] = %v, want trace-123", event.Metadata["trace_id"])
	}
	if event.Metadata["request_id"] != "req-456" {
		t.Errorf("Metadata[request_id] = %v, want req-456", event.Metadata["request_id"])
	}
}

func TestEventTypes(t *testing.T) {
	types := []struct {
		name      string
		eventType string
	}{
		{"EventTypeOrderCreated", EventTypeOrderCreated},
		{"EventTypeOrderFilled", EventTypeOrderFilled},
		{"EventTypePaymentInitiated", EventTypePaymentInitiated},
		{"EventTypePaymentCompleted", EventTypePaymentCompleted},
		{"EventTypeKYCVerified", EventTypeKYCVerified},
		{"EventTypeUserRegistered", EventTypeUserRegistered},
		{"EventTypePriceUpdate", EventTypePriceUpdate},
	}

	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			// All event types should end with version (v1, v2, etc.)
			if len(tt.eventType) < 3 {
				t.Errorf("%s is too short", tt.eventType)
			}
			lastThree := tt.eventType[len(tt.eventType)-3:]
			if lastThree[0] != '.' || lastThree[1] != 'v' {
				t.Errorf("%s should end with version (e.g., .v1): %s", tt.name, tt.eventType)
			}
		})
	}
}

func TestEventStruct(t *testing.T) {
	event := Event{
		EventID:       "test-id",
		EventType:     "order.created.v1",
		Source:        "trading-service",
		OccurredAt:    time.Now(),
		CorrelationID: "corr-123",
		Payload: map[string]any{
			"order_id": "123",
			"amount":   100.50,
		},
		Metadata: map[string]string{
			"trace_id": "abc-123",
		},
	}

	if event.EventID != "test-id" {
		t.Errorf("Event.EventID = %v, want test-id", event.EventID)
	}
	if event.EventType != "order.created.v1" {
		t.Errorf("Event.EventType = %v, want order.created.v1", event.EventType)
	}
	if event.Source != "trading-service" {
		t.Errorf("Event.Source = %v, want trading-service", event.Source)
	}
	if event.CorrelationID != "corr-123" {
		t.Errorf("Event.CorrelationID = %v, want corr-123", event.CorrelationID)
	}
	if event.Metadata["trace_id"] != "abc-123" {
		t.Errorf("Event.Metadata[trace_id] = %v, want abc-123", event.Metadata["trace_id"])
	}
}

func TestLegacyCompatibility(t *testing.T) {
	// TopicOrderUpdated should still work (deprecated alias)
	if TopicOrderUpdated != TopicOrderFilled {
		t.Errorf("TopicOrderUpdated should alias to TopicOrderFilled")
	}
}

func TestPayloadStructs(t *testing.T) {
	// Test that payload structs can be instantiated
	_ = OrderCreatedPayload{
		OrderID: "order-1",
		UserID:  "user-1",
		Symbol:  "AAPL",
		Side:    "buy",
		Amount:  100.00,
	}

	_ = PaymentCompletedPayload{
		UserID:        "user-1",
		WalletID:      "wallet-1",
		TransactionID: "tx-1",
		Amount:        500.00,
		Currency:      "KES",
		Provider:      "mpesa",
		ProviderRef:   "ABC123",
		CompletedAt:   time.Now(),
		NewBalance:    1500.00,
	}

	_ = PriceUpdatePayload{
		Symbol:    "AAPL",
		BidPrice:  149.50,
		AskPrice:  150.50,
		LastPrice: 150.00,
		Volume:    1000000,
		Timestamp: time.Now(),
	}
}
