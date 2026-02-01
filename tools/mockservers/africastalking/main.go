package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/google/uuid"
)

// =============================================================================
// Africa's Talking Mock Server
// =============================================================================
// This server simulates the Africa's Talking SMS API for integration testing.
// It supports:
// - SMS sending (single and bulk)
// - Delivery status callbacks
// - Message history
// =============================================================================

type Server struct {
	mu       sync.RWMutex
	messages []Message
	apiKey   string
}

type Message struct {
	ID        string    `json:"id"`
	To        string    `json:"to"`
	Message   string    `json:"message"`
	From      string    `json:"from"`
	Status    string    `json:"status"` // Sent, Delivered, Failed
	Cost      string    `json:"cost"`
	CreatedAt time.Time `json:"created_at"`
}

func NewServer() *Server {
	return &Server{
		messages: make([]Message, 0),
		apiKey:   os.Getenv("AT_API_KEY"),
	}
}

func main() {
	server := NewServer()

	app := fiber.New(fiber.Config{
		AppName: "Africa's Talking Mock Server",
	})

	app.Use(logger.New())

	// SMS endpoint
	app.Post("/version1/messaging", server.handleSMS)

	// Bulk SMS endpoint
	app.Post("/version1/messaging/bulk", server.handleBulkSMS)

	// Admin endpoints
	app.Get("/admin/messages", server.listMessages)
	app.Post("/admin/reset", server.reset)
	app.Post("/admin/trigger-delivery/:id", server.triggerDelivery)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy", "service": "africastalking-mock"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8091"
	}

	log.Printf("Africa's Talking Mock Server starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}

func (s *Server) validateAPIKey(c *fiber.Ctx) bool {
	apiKey := c.Get("apiKey")
	if s.apiKey != "" && apiKey != s.apiKey {
		return false
	}
	return apiKey != ""
}

func (s *Server) handleSMS(c *fiber.Ctx) error {
	if !s.validateAPIKey(c) {
		return c.Status(401).JSON(fiber.Map{
			"SMSMessageData": fiber.Map{
				"Message": "Invalid API key",
			},
		})
	}

	username := c.FormValue("username")
	to := c.FormValue("to")
	message := c.FormValue("message")
	from := c.FormValue("from")

	if username == "" || to == "" || message == "" {
		return c.Status(400).JSON(fiber.Map{
			"SMSMessageData": fiber.Map{
				"Message": "Missing required parameters",
			},
		})
	}

	// Parse multiple recipients
	recipients := parseRecipients(to)
	responseRecipients := make([]fiber.Map, 0, len(recipients))

	s.mu.Lock()
	for _, recipient := range recipients {
		msgID := uuid.New().String()
		msg := Message{
			ID:        msgID,
			To:        recipient,
			Message:   message,
			From:      from,
			Status:    "Sent",
			Cost:      "KES 0.80",
			CreatedAt: time.Now(),
		}
		s.messages = append(s.messages, msg)

		responseRecipients = append(responseRecipients, fiber.Map{
			"statusCode": 101,
			"number":     recipient,
			"status":     "Success",
			"cost":       "KES 0.80",
			"messageId":  msgID,
		})
	}
	s.mu.Unlock()

	return c.Status(201).JSON(fiber.Map{
		"SMSMessageData": fiber.Map{
			"Message":    fmt.Sprintf("Sent to %d/%d Total Cost: KES %.2f", len(recipients), len(recipients), float64(len(recipients))*0.80),
			"Recipients": responseRecipients,
		},
	})
}

func (s *Server) handleBulkSMS(c *fiber.Ctx) error {
	return s.handleSMS(c)
}

func parseRecipients(to string) []string {
	recipients := make([]string, 0)
	current := ""
	for _, ch := range to {
		if ch == ',' {
			if current != "" {
				recipients = append(recipients, current)
				current = ""
			}
		} else if ch != ' ' {
			current += string(ch)
		}
	}
	if current != "" {
		recipients = append(recipients, current)
	}
	return recipients
}

// =============================================================================
// Admin Endpoints
// =============================================================================

func (s *Server) listMessages(c *fiber.Ctx) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return c.JSON(fiber.Map{
		"messages": s.messages,
		"count":    len(s.messages),
	})
}

func (s *Server) reset(c *fiber.Ctx) error {
	s.mu.Lock()
	s.messages = make([]Message, 0)
	s.mu.Unlock()

	return c.JSON(fiber.Map{"status": "reset complete"})
}

func (s *Server) triggerDelivery(c *fiber.Ctx) error {
	id := c.Params("id")
	status := c.Query("status", "Delivered")

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.messages {
		if s.messages[i].ID == id {
			s.messages[i].Status = status
			return c.JSON(fiber.Map{
				"status":  "delivery status updated",
				"message": s.messages[i],
			})
		}
	}

	return c.Status(404).JSON(fiber.Map{"error": "message not found"})
}
