package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/services/notification-service/internal/sender"
	"github.com/Rohianon/equishare-global-trading/services/notification-service/internal/types"
)

// Handler handles notification HTTP requests
type Handler struct {
	sender        *sender.Sender
	notifications []types.Notification // In-memory storage for demo; use DB in production
}

// NewHandler creates a new notification handler
func NewHandler(s *sender.Sender) *Handler {
	return &Handler{
		sender:        s,
		notifications: make([]types.Notification, 0),
	}
}

// SendNotification sends a notification
// POST /notifications/send
func (h *Handler) SendNotification(c *fiber.Ctx) error {
	var req types.SendNotificationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
			Code:    400,
		})
	}

	// Validate required fields
	if req.Type == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Error:   "bad_request",
			Message: "Notification type is required",
			Code:    400,
		})
	}

	var err error
	var message string
	var recipient string

	switch req.Type {
	case types.NotificationTypeSMS:
		if req.Phone == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
				Error:   "bad_request",
				Message: "Phone number is required for SMS",
				Code:    400,
			})
		}
		recipient = req.Phone

		// Use template if provided
		if req.Template != "" {
			template, ok := h.sender.GetTemplate(req.Template)
			if !ok {
				return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
					Error:   "bad_request",
					Message: "Invalid template type",
					Code:    400,
				})
			}
			message = renderTemplate(template.Body, req.Data)
			err = h.sender.SendSMS(req.Phone, message)
		} else if req.Message != "" {
			message = req.Message
			err = h.sender.SendSMS(req.Phone, req.Message)
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
				Error:   "bad_request",
				Message: "Message or template is required",
				Code:    400,
			})
		}

	case types.NotificationTypeEmail:
		if req.Email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
				Error:   "bad_request",
				Message: "Email is required for email notifications",
				Code:    400,
			})
		}
		recipient = req.Email
		message = req.Message
		err = h.sender.SendEmail(req.Email, req.Subject, req.Message)

	case types.NotificationTypePush:
		if req.UserID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
				Error:   "bad_request",
				Message: "User ID is required for push notifications",
				Code:    400,
			})
		}
		recipient = req.UserID
		message = req.Message
		err = h.sender.SendPush(req.UserID, req.Subject, req.Message)

	default:
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid notification type",
			Code:    400,
		})
	}

	// Create notification record
	now := time.Now()
	notification := types.Notification{
		ID:        uuid.New().String(),
		UserID:    req.UserID,
		Type:      req.Type,
		Template:  req.Template,
		Recipient: recipient,
		Subject:   req.Subject,
		Message:   message,
		Status:    types.StatusSent,
		SentAt:    &now,
		CreatedAt: now,
	}

	if err != nil {
		notification.Status = types.StatusFailed
		logger.Error().Err(err).Str("type", string(req.Type)).Msg("Failed to send notification")
	}

	// Store notification (in-memory for demo)
	h.notifications = append(h.notifications, notification)

	status := "sent"
	statusMsg := "Notification sent successfully"
	if err != nil {
		status = "failed"
		statusMsg = "Failed to send notification"
	}

	return c.JSON(types.SendNotificationResponse{
		ID:      notification.ID,
		Status:  types.NotificationStatus(status),
		Message: statusMsg,
	})
}

// ListNotifications lists notification history
// GET /notifications?user_id=xxx
func (h *Handler) ListNotifications(c *fiber.Ctx) error {
	userID := c.Query("user_id")

	filtered := make([]types.Notification, 0)
	for _, n := range h.notifications {
		if userID == "" || n.UserID == userID {
			filtered = append(filtered, n)
		}
	}

	// Return most recent first
	for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}

	// Limit to last 100
	if len(filtered) > 100 {
		filtered = filtered[:100]
	}

	return c.JSON(types.NotificationListResponse{
		Notifications: filtered,
		Total:         len(filtered),
	})
}

// GetNotification retrieves a specific notification
// GET /notifications/:id
func (h *Handler) GetNotification(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Error:   "bad_request",
			Message: "Notification ID is required",
			Code:    400,
		})
	}

	for _, n := range h.notifications {
		if n.ID == id {
			return c.JSON(n)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(types.ErrorResponse{
		Error:   "not_found",
		Message: "Notification not found",
		Code:    404,
	})
}

// ListTemplates lists available notification templates
// GET /notifications/templates
func (h *Handler) ListTemplates(c *fiber.Ctx) error {
	templates := h.sender.GetTemplates()
	return c.JSON(types.TemplateListResponse{
		Templates: templates,
	})
}

// Helper function to render template
func renderTemplate(template string, data map[string]interface{}) string {
	result := template
	if data == nil {
		return result
	}
	for key, value := range data {
		placeholder := "{{" + key + "}}"
		result = replaceAll(result, placeholder, toString(value))
	}
	return result
}

func replaceAll(s, old, new string) string {
	for {
		idx := indexOf(s, old)
		if idx == -1 {
			return s
		}
		s = s[:idx] + new + s[idx+len(old):]
	}
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return itoa(val)
	case int64:
		return itoa64(val)
	case float64:
		return ftoa(val)
	default:
		return ""
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	result := ""
	negative := i < 0
	if negative {
		i = -i
	}
	for i > 0 {
		result = string(rune('0'+i%10)) + result
		i /= 10
	}
	if negative {
		result = "-" + result
	}
	return result
}

func itoa64(i int64) string {
	return itoa(int(i))
}

func ftoa(f float64) string {
	// Simple float to string, 2 decimal places
	intPart := int(f)
	fracPart := int((f - float64(intPart)) * 100)
	if fracPart < 0 {
		fracPart = -fracPart
	}
	return itoa(intPart) + "." + padLeft(itoa(fracPart), 2, '0')
}

func padLeft(s string, length int, pad rune) string {
	for len(s) < length {
		s = string(pad) + s
	}
	return s
}
