package types

import "time"

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeSMS   NotificationType = "sms"
	NotificationTypeEmail NotificationType = "email"
	NotificationTypePush  NotificationType = "push"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusSent      NotificationStatus = "sent"
	StatusFailed    NotificationStatus = "failed"
	StatusDelivered NotificationStatus = "delivered"
)

// TemplateType represents predefined notification templates
type TemplateType string

const (
	TemplateOrderPlaced     TemplateType = "order_placed"
	TemplateOrderFilled     TemplateType = "order_filled"
	TemplatePaymentReceived TemplateType = "payment_received"
	TemplateWithdrawal      TemplateType = "withdrawal"
	TemplatePriceAlert      TemplateType = "price_alert"
	TemplateOTP             TemplateType = "otp"
	TemplateWelcome         TemplateType = "welcome"
)

// SendNotificationRequest represents a request to send a notification
type SendNotificationRequest struct {
	UserID   string                 `json:"user_id"`
	Type     NotificationType       `json:"type"`
	Template TemplateType           `json:"template,omitempty"`
	Phone    string                 `json:"phone,omitempty"`
	Email    string                 `json:"email,omitempty"`
	Subject  string                 `json:"subject,omitempty"`
	Message  string                 `json:"message,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// SendNotificationResponse represents the response after sending
type SendNotificationResponse struct {
	ID      string             `json:"id"`
	Status  NotificationStatus `json:"status"`
	Message string             `json:"message"`
}

// Notification represents a stored notification
type Notification struct {
	ID        string             `json:"id"`
	UserID    string             `json:"user_id"`
	Type      NotificationType   `json:"type"`
	Template  TemplateType       `json:"template,omitempty"`
	Recipient string             `json:"recipient"`
	Subject   string             `json:"subject,omitempty"`
	Message   string             `json:"message"`
	Status    NotificationStatus `json:"status"`
	SentAt    *time.Time         `json:"sent_at,omitempty"`
	CreatedAt time.Time          `json:"created_at"`
}

// NotificationListResponse represents a list of notifications
type NotificationListResponse struct {
	Notifications []Notification `json:"notifications"`
	Total         int            `json:"total"`
}

// Template represents a notification template
type Template struct {
	Type    TemplateType `json:"type"`
	Name    string       `json:"name"`
	Subject string       `json:"subject,omitempty"`
	Body    string       `json:"body"`
}

// TemplateListResponse represents available templates
type TemplateListResponse struct {
	Templates []Template `json:"templates"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
