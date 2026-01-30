package events

import (
	"context"
	"time"
)

type Event struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Source   string            `json:"source"`
	Time     time.Time         `json:"time"`
	Data     any               `json:"data"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

const (
	TopicOrderCreated   = "equishare.orders.created"
	TopicOrderUpdated   = "equishare.orders.updated"
	TopicOrderFilled    = "equishare.orders.filled"
	TopicOrderCancelled = "equishare.orders.cancelled"

	TopicPaymentInitiated = "equishare.payments.initiated"
	TopicPaymentCompleted = "equishare.payments.completed"
	TopicPaymentFailed    = "equishare.payments.failed"

	TopicWithdrawalInitiated = "equishare.withdrawals.initiated"
	TopicWithdrawalCompleted = "equishare.withdrawals.completed"

	TopicKYCSubmitted = "equishare.kyc.submitted"
	TopicKYCVerified  = "equishare.kyc.verified"
	TopicKYCRejected  = "equishare.kyc.rejected"

	TopicPriceUpdate    = "equishare.prices.realtime"
	TopicAlertTriggered = "equishare.alerts.triggered"

	TopicNotificationSend = "equishare.notifications.send"
)

type Publisher interface {
	Publish(ctx context.Context, topic string, event *Event) error
	Close() error
}

type Subscriber interface {
	Subscribe(ctx context.Context, topic string, handler func(*Event) error) error
	Close() error
}
