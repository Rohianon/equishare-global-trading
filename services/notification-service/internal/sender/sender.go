package sender

import (
	"fmt"
	"strings"

	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/services/notification-service/internal/types"
)

// SMSClient interface for sending SMS
type SMSClient interface {
	Send(to, message string) error
}

// Sender handles sending notifications via different channels
type Sender struct {
	sms       SMSClient
	templates map[types.TemplateType]types.Template
}

// NewSender creates a new notification sender
func NewSender(smsClient SMSClient) *Sender {
	s := &Sender{
		sms:       smsClient,
		templates: make(map[types.TemplateType]types.Template),
	}
	s.loadTemplates()
	return s
}

// loadTemplates loads predefined notification templates
func (s *Sender) loadTemplates() {
	s.templates = map[types.TemplateType]types.Template{
		types.TemplateOTP: {
			Type: types.TemplateOTP,
			Name: "OTP Verification",
			Body: "Your EquiShare verification code is: {{code}}. Valid for 5 minutes. Do not share this code.",
		},
		types.TemplateWelcome: {
			Type: types.TemplateWelcome,
			Name: "Welcome Message",
			Body: "Welcome to EquiShare! Your account has been created. Start investing in US stocks today.",
		},
		types.TemplateOrderPlaced: {
			Type: types.TemplateOrderPlaced,
			Name: "Order Placed",
			Body: "Your {{side}} order for {{quantity}} {{symbol}} has been placed. Order ID: {{order_id}}",
		},
		types.TemplateOrderFilled: {
			Type: types.TemplateOrderFilled,
			Name: "Order Filled",
			Body: "Your order for {{symbol}} has been filled at ${{price}}. Quantity: {{quantity}}. Total: ${{total}}",
		},
		types.TemplatePaymentReceived: {
			Type:    types.TemplatePaymentReceived,
			Name:    "Payment Received",
			Body:    "Your EquiShare wallet has been credited with KES {{amount}}. Receipt: {{receipt}}. New balance: KES {{balance}}",
		},
		types.TemplateWithdrawal: {
			Type: types.TemplateWithdrawal,
			Name: "Withdrawal Processed",
			Body: "Your withdrawal of KES {{amount}} has been processed. It will arrive in your M-Pesa within 24 hours.",
		},
		types.TemplatePriceAlert: {
			Type: types.TemplatePriceAlert,
			Name: "Price Alert",
			Body: "Price Alert: {{symbol}} is now at ${{price}}. Your target was ${{target}}.",
		},
	}
}

// GetTemplates returns all available templates
func (s *Sender) GetTemplates() []types.Template {
	templates := make([]types.Template, 0, len(s.templates))
	for _, t := range s.templates {
		templates = append(templates, t)
	}
	return templates
}

// GetTemplate returns a specific template
func (s *Sender) GetTemplate(templateType types.TemplateType) (*types.Template, bool) {
	t, ok := s.templates[templateType]
	if !ok {
		return nil, false
	}
	return &t, true
}

// SendSMS sends an SMS notification
func (s *Sender) SendSMS(phone, message string) error {
	if s.sms == nil {
		logger.Warn().Msg("SMS client not configured, skipping SMS")
		return nil
	}

	if err := s.sms.Send(phone, message); err != nil {
		logger.Error().Err(err).Str("phone", phone).Msg("Failed to send SMS")
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	logger.Info().Str("phone", phone).Msg("SMS sent successfully")
	return nil
}

// SendTemplatedSMS sends an SMS using a template
func (s *Sender) SendTemplatedSMS(phone string, templateType types.TemplateType, data map[string]interface{}) error {
	template, ok := s.GetTemplate(templateType)
	if !ok {
		return fmt.Errorf("template not found: %s", templateType)
	}

	message := s.renderTemplate(template.Body, data)
	return s.SendSMS(phone, message)
}

// renderTemplate replaces template placeholders with actual values
func (s *Sender) renderTemplate(template string, data map[string]interface{}) string {
	result := template
	for key, value := range data {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// SendEmail sends an email notification (placeholder for future implementation)
func (s *Sender) SendEmail(to, subject, body string) error {
	// TODO: Implement email sending
	logger.Warn().Str("to", to).Msg("Email sending not implemented yet")
	return nil
}

// SendPush sends a push notification (placeholder for future implementation)
func (s *Sender) SendPush(userID, title, body string) error {
	// TODO: Implement push notifications
	logger.Warn().Str("user_id", userID).Msg("Push notifications not implemented yet")
	return nil
}
