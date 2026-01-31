package events

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Event represents the standard event envelope for all Kafka messages.
// All events published to Kafka MUST use this envelope format.
//
// Topic Naming Convention:
//
//	equishare.<domain>.<action>
//	Examples: equishare.orders.created, equishare.payments.completed
//
// Event Versioning:
//   - Event types include version: "order.created.v1"
//   - Payload structure is versioned independently
//   - Breaking changes require new version (v2, v3, etc.)
//   - Consumers should handle unknown fields gracefully
type Event struct {
	// EventID is a unique identifier for this event instance
	EventID string `json:"event_id"`

	// EventType describes the event in format: <domain>.<action>.v<version>
	// Examples: "order.created.v1", "payment.completed.v1"
	EventType string `json:"event_type"`

	// OccurredAt is when the event actually happened (not when it was published)
	OccurredAt time.Time `json:"occurred_at"`

	// CorrelationID links related events across services (e.g., same user request)
	CorrelationID string `json:"correlation_id,omitempty"`

	// Source identifies the service that produced this event
	Source string `json:"source"`

	// Payload contains the event-specific data
	Payload any `json:"payload"`

	// Metadata contains optional key-value pairs for tracing, debugging, etc.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NewEvent creates a new event with auto-generated ID and timestamp
func NewEvent(eventType, source string, payload any) *Event {
	return &Event{
		EventID:    uuid.New().String(),
		EventType:  eventType,
		OccurredAt: time.Now().UTC(),
		Source:     source,
		Payload:    payload,
		Metadata:   make(map[string]string),
	}
}

// WithCorrelationID sets the correlation ID for request tracing
func (e *Event) WithCorrelationID(id string) *Event {
	e.CorrelationID = id
	return e
}

// WithMetadata adds a metadata key-value pair
func (e *Event) WithMetadata(key, value string) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// =============================================================================
// Topic Registry
// =============================================================================
// All Kafka topics used in the system are defined here.
// Topic naming: equishare.<domain>.<action>
//
// Adding a new topic:
// 1. Add the constant below with documentation
// 2. Document the payload structure
// 3. Update the Topics slice
// =============================================================================

const (
	// Order Domain
	// Published by: trading-service
	// Consumed by: portfolio-service, notification-service

	// TopicOrderCreated is published when a new order is submitted
	// Payload: OrderCreatedPayload
	TopicOrderCreated = "equishare.orders.created"

	// TopicOrderFilled is published when an order is completely filled
	// Payload: OrderFilledPayload
	TopicOrderFilled = "equishare.orders.filled"

	// TopicOrderPartialFill is published when an order is partially filled
	// Payload: OrderPartialFillPayload
	TopicOrderPartialFill = "equishare.orders.partial_fill"

	// TopicOrderCancelled is published when an order is cancelled
	// Payload: OrderCancelledPayload
	TopicOrderCancelled = "equishare.orders.cancelled"

	// TopicOrderRejected is published when an order is rejected by the exchange
	// Payload: OrderRejectedPayload
	TopicOrderRejected = "equishare.orders.rejected"

	// Payment Domain
	// Published by: payment-service
	// Consumed by: notification-service, trading-service

	// TopicPaymentInitiated is published when a deposit/payment is initiated
	// Payload: PaymentInitiatedPayload
	TopicPaymentInitiated = "equishare.payments.initiated"

	// TopicPaymentCompleted is published when a payment succeeds
	// Payload: PaymentCompletedPayload
	TopicPaymentCompleted = "equishare.payments.completed"

	// TopicPaymentFailed is published when a payment fails
	// Payload: PaymentFailedPayload
	TopicPaymentFailed = "equishare.payments.failed"

	// Withdrawal Domain
	// Published by: payment-service
	// Consumed by: notification-service

	// TopicWithdrawalInitiated is published when a withdrawal is requested
	// Payload: WithdrawalInitiatedPayload
	TopicWithdrawalInitiated = "equishare.withdrawals.initiated"

	// TopicWithdrawalCompleted is published when a withdrawal succeeds
	// Payload: WithdrawalCompletedPayload
	TopicWithdrawalCompleted = "equishare.withdrawals.completed"

	// TopicWithdrawalFailed is published when a withdrawal fails
	// Payload: WithdrawalFailedPayload
	TopicWithdrawalFailed = "equishare.withdrawals.failed"

	// KYC Domain
	// Published by: user-service
	// Consumed by: notification-service, trading-service

	// TopicKYCSubmitted is published when KYC documents are submitted
	// Payload: KYCSubmittedPayload
	TopicKYCSubmitted = "equishare.kyc.submitted"

	// TopicKYCVerified is published when KYC is approved
	// Payload: KYCVerifiedPayload
	TopicKYCVerified = "equishare.kyc.verified"

	// TopicKYCRejected is published when KYC is rejected
	// Payload: KYCRejectedPayload
	TopicKYCRejected = "equishare.kyc.rejected"

	// User Domain
	// Published by: auth-service, user-service
	// Consumed by: notification-service

	// TopicUserRegistered is published when a new user registers
	// Payload: UserRegisteredPayload
	TopicUserRegistered = "equishare.users.registered"

	// TopicUserVerified is published when a user verifies their phone/email
	// Payload: UserVerifiedPayload
	TopicUserVerified = "equishare.users.verified"

	// Market Data Domain
	// Published by: market-data-service
	// Consumed by: trading-service, portfolio-service

	// TopicPriceUpdate is published for real-time price updates
	// Payload: PriceUpdatePayload
	TopicPriceUpdate = "equishare.prices.update"

	// TopicMarketOpen is published when market opens
	// Payload: MarketStatusPayload
	TopicMarketOpen = "equishare.market.open"

	// TopicMarketClose is published when market closes
	// Payload: MarketStatusPayload
	TopicMarketClose = "equishare.market.close"

	// Notification Domain
	// Published by: various services
	// Consumed by: notification-service

	// TopicNotificationSend is published to trigger notifications
	// Payload: NotificationPayload
	TopicNotificationSend = "equishare.notifications.send"

	// Alert Domain
	// Published by: portfolio-service
	// Consumed by: notification-service

	// TopicAlertTriggered is published when a price alert is triggered
	// Payload: AlertTriggeredPayload
	TopicAlertTriggered = "equishare.alerts.triggered"
)

// AllTopics returns all registered topics for admin/setup purposes
var AllTopics = []string{
	TopicOrderCreated,
	TopicOrderFilled,
	TopicOrderPartialFill,
	TopicOrderCancelled,
	TopicOrderRejected,
	TopicPaymentInitiated,
	TopicPaymentCompleted,
	TopicPaymentFailed,
	TopicWithdrawalInitiated,
	TopicWithdrawalCompleted,
	TopicWithdrawalFailed,
	TopicKYCSubmitted,
	TopicKYCVerified,
	TopicKYCRejected,
	TopicUserRegistered,
	TopicUserVerified,
	TopicPriceUpdate,
	TopicMarketOpen,
	TopicMarketClose,
	TopicNotificationSend,
	TopicAlertTriggered,
}

// =============================================================================
// Event Types (versioned)
// =============================================================================

const (
	// Order events
	EventTypeOrderCreated     = "order.created.v1"
	EventTypeOrderFilled      = "order.filled.v1"
	EventTypeOrderPartialFill = "order.partial_fill.v1"
	EventTypeOrderCancelled   = "order.cancelled.v1"
	EventTypeOrderRejected    = "order.rejected.v1"

	// Payment events
	EventTypePaymentInitiated = "payment.initiated.v1"
	EventTypePaymentCompleted = "payment.completed.v1"
	EventTypePaymentFailed    = "payment.failed.v1"

	// Withdrawal events
	EventTypeWithdrawalInitiated = "withdrawal.initiated.v1"
	EventTypeWithdrawalCompleted = "withdrawal.completed.v1"
	EventTypeWithdrawalFailed    = "withdrawal.failed.v1"

	// KYC events
	EventTypeKYCSubmitted = "kyc.submitted.v1"
	EventTypeKYCVerified  = "kyc.verified.v1"
	EventTypeKYCRejected  = "kyc.rejected.v1"

	// User events
	EventTypeUserRegistered = "user.registered.v1"
	EventTypeUserVerified   = "user.verified.v1"

	// Market events
	EventTypePriceUpdate = "price.update.v1"
	EventTypeMarketOpen  = "market.open.v1"
	EventTypeMarketClose = "market.close.v1"

	// Notification events
	EventTypeNotificationSend = "notification.send.v1"

	// Alert events
	EventTypeAlertTriggered = "alert.triggered.v1"
)

// =============================================================================
// Interfaces
// =============================================================================

// Publisher publishes events to Kafka topics
type Publisher interface {
	// Publish sends an event to the specified topic
	Publish(ctx context.Context, topic string, event *Event) error

	// Close closes the publisher and releases resources
	Close() error
}

// Subscriber consumes events from Kafka topics
type Subscriber interface {
	// Subscribe registers a handler for events on the specified topic
	Subscribe(ctx context.Context, topic string, handler func(*Event) error) error

	// Close closes the subscriber and releases resources
	Close() error
}

// =============================================================================
// Legacy compatibility - these will be removed in v2
// =============================================================================

// Deprecated: Use TopicOrderFilled instead
var TopicOrderUpdated = TopicOrderFilled
