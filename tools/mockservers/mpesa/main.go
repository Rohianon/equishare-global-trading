package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/google/uuid"
)

// =============================================================================
// M-Pesa Mock Server
// =============================================================================
// This server simulates the Safaricom M-Pesa Daraja API for integration testing.
// It supports:
// - OAuth token generation
// - STK Push (Lipa Na M-Pesa Online)
// - B2C (Business to Customer) payments
// - Callback simulation
// =============================================================================

type Server struct {
	mu           sync.RWMutex
	tokens       map[string]tokenInfo
	stkRequests  map[string]*STKRequest
	b2cRequests  map[string]*B2CRequest
	callbackURL  string
	callbackChan chan CallbackPayload
}

type tokenInfo struct {
	token     string
	expiresAt time.Time
}

type STKRequest struct {
	CheckoutRequestID string
	MerchantRequestID string
	Phone             string
	Amount            int
	Reference         string
	Status            string // pending, success, failed, cancelled
	CreatedAt         time.Time
}

type B2CRequest struct {
	ConversationID           string
	OriginatorConversationID string
	Phone                    string
	Amount                   int
	Reference                string
	Status                   string // pending, success, failed
	CreatedAt                time.Time
}

type CallbackPayload struct {
	Type    string // stk, b2c
	Request any
	Success bool
	Delay   time.Duration
}

func NewServer() *Server {
	return &Server{
		tokens:       make(map[string]tokenInfo),
		stkRequests:  make(map[string]*STKRequest),
		b2cRequests:  make(map[string]*B2CRequest),
		callbackChan: make(chan CallbackPayload, 100),
	}
}

func main() {
	server := NewServer()

	app := fiber.New(fiber.Config{
		AppName: "M-Pesa Mock Server",
	})

	app.Use(logger.New())

	// OAuth endpoint
	app.Get("/oauth/v1/generate", server.generateToken)

	// STK Push endpoint
	app.Post("/mpesa/stkpush/v1/processrequest", server.handleSTKPush)

	// STK Query endpoint
	app.Post("/mpesa/stkpushquery/v1/query", server.handleSTKQuery)

	// B2C endpoint
	app.Post("/mpesa/b2c/v1/paymentrequest", server.handleB2C)

	// Admin endpoints for testing
	app.Post("/admin/trigger-callback/:id", server.triggerCallback)
	app.Get("/admin/requests", server.listRequests)
	app.Post("/admin/reset", server.reset)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy", "service": "mpesa-mock"})
	})

	// Start callback processor
	go server.processCallbacks()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	log.Printf("M-Pesa Mock Server starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}

// =============================================================================
// OAuth
// =============================================================================

func (s *Server) generateToken(c *fiber.Ctx) error {
	auth := c.Get("Authorization")
	if auth == "" || len(auth) < 7 {
		return c.Status(401).JSON(fiber.Map{
			"errorCode":    "401.001",
			"errorMessage": "Invalid credentials",
		})
	}

	token := uuid.New().String()
	s.mu.Lock()
	s.tokens[token] = tokenInfo{
		token:     token,
		expiresAt: time.Now().Add(1 * time.Hour),
	}
	s.mu.Unlock()

	return c.JSON(fiber.Map{
		"access_token": token,
		"expires_in":   "3599",
	})
}

func (s *Server) validateToken(c *fiber.Ctx) bool {
	auth := c.Get("Authorization")
	if auth == "" || len(auth) < 8 {
		return false
	}

	token := auth[7:] // Remove "Bearer "
	s.mu.RLock()
	info, exists := s.tokens[token]
	s.mu.RUnlock()

	return exists && time.Now().Before(info.expiresAt)
}

// =============================================================================
// STK Push
// =============================================================================

type stkPushRequest struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            int    `json:"Amount"`
	PartyA            string `json:"PartyA"`
	PartyB            string `json:"PartyB"`
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

func (s *Server) handleSTKPush(c *fiber.Ctx) error {
	if !s.validateToken(c) {
		return c.Status(401).JSON(fiber.Map{
			"errorCode":    "401.002",
			"errorMessage": "Invalid access token",
		})
	}

	var req stkPushRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"errorCode":    "400.001",
			"errorMessage": "Invalid request body",
		})
	}

	merchantReqID := fmt.Sprintf("ws_CO_%s_%d", time.Now().Format("20060102150405"), time.Now().UnixNano()%100000)
	checkoutReqID := fmt.Sprintf("ws_CO_%s_%d", time.Now().Format("20060102150405"), time.Now().UnixNano()%100000+1)

	stkReq := &STKRequest{
		CheckoutRequestID: checkoutReqID,
		MerchantRequestID: merchantReqID,
		Phone:             req.PhoneNumber,
		Amount:            req.Amount,
		Reference:         req.AccountReference,
		Status:            "pending",
		CreatedAt:         time.Now(),
	}

	s.mu.Lock()
	s.stkRequests[checkoutReqID] = stkReq
	s.callbackURL = req.CallBackURL
	s.mu.Unlock()

	// Schedule automatic callback (simulates user completing payment)
	go func() {
		time.Sleep(2 * time.Second) // Simulate user interaction delay
		s.callbackChan <- CallbackPayload{
			Type:    "stk",
			Request: stkReq,
			Success: true,
			Delay:   0,
		}
	}()

	return c.JSON(fiber.Map{
		"MerchantRequestID":   merchantReqID,
		"CheckoutRequestID":   checkoutReqID,
		"ResponseCode":        "0",
		"ResponseDescription": "Success. Request accepted for processing",
		"CustomerMessage":     "Success. Request accepted for processing",
	})
}

func (s *Server) handleSTKQuery(c *fiber.Ctx) error {
	if !s.validateToken(c) {
		return c.Status(401).JSON(fiber.Map{
			"errorCode": "401.002",
		})
	}

	var req struct {
		BusinessShortCode string `json:"BusinessShortCode"`
		Password          string `json:"Password"`
		Timestamp         string `json:"Timestamp"`
		CheckoutRequestID string `json:"CheckoutRequestID"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"errorCode": "400.001"})
	}

	s.mu.RLock()
	stkReq, exists := s.stkRequests[req.CheckoutRequestID]
	s.mu.RUnlock()

	if !exists {
		return c.JSON(fiber.Map{
			"ResponseCode":        "1",
			"ResponseDescription": "The transaction is not found",
			"CheckoutRequestID":   req.CheckoutRequestID,
		})
	}

	resultCode := "1032" // cancelled
	resultDesc := "Request cancelled by user"
	if stkReq.Status == "success" {
		resultCode = "0"
		resultDesc = "The service request is processed successfully."
	}

	return c.JSON(fiber.Map{
		"ResponseCode":        "0",
		"ResponseDescription": "The service request has been accepted successfully",
		"MerchantRequestID":   stkReq.MerchantRequestID,
		"CheckoutRequestID":   stkReq.CheckoutRequestID,
		"ResultCode":          resultCode,
		"ResultDesc":          resultDesc,
	})
}

// =============================================================================
// B2C
// =============================================================================

type b2cRequest struct {
	InitiatorName      string `json:"InitiatorName"`
	SecurityCredential string `json:"SecurityCredential"`
	CommandID          string `json:"CommandID"`
	Amount             int    `json:"Amount"`
	PartyA             string `json:"PartyA"`
	PartyB             string `json:"PartyB"`
	Remarks            string `json:"Remarks"`
	QueueTimeOutURL    string `json:"QueueTimeOutURL"`
	ResultURL          string `json:"ResultURL"`
	Occasion           string `json:"Occasion"`
}

func (s *Server) handleB2C(c *fiber.Ctx) error {
	if !s.validateToken(c) {
		return c.Status(401).JSON(fiber.Map{"errorCode": "401.002"})
	}

	var req b2cRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"errorCode": "400.001"})
	}

	convID := fmt.Sprintf("AG_%s_%d", time.Now().Format("20060102150405"), time.Now().UnixNano()%100000)
	origConvID := fmt.Sprintf("%d-%d", time.Now().Unix(), time.Now().UnixNano()%1000000)

	b2cReq := &B2CRequest{
		ConversationID:           convID,
		OriginatorConversationID: origConvID,
		Phone:                    req.PartyB,
		Amount:                   req.Amount,
		Reference:                req.Occasion,
		Status:                   "pending",
		CreatedAt:                time.Now(),
	}

	s.mu.Lock()
	s.b2cRequests[convID] = b2cReq
	s.callbackURL = req.ResultURL
	s.mu.Unlock()

	// Schedule automatic callback
	go func() {
		time.Sleep(2 * time.Second)
		s.callbackChan <- CallbackPayload{
			Type:    "b2c",
			Request: b2cReq,
			Success: true,
			Delay:   0,
		}
	}()

	return c.JSON(fiber.Map{
		"ConversationID":           convID,
		"OriginatorConversationID": origConvID,
		"ResponseCode":             "0",
		"ResponseDescription":      "Accept the service request successfully.",
	})
}

// =============================================================================
// Callbacks
// =============================================================================

func (s *Server) processCallbacks() {
	client := &http.Client{Timeout: 10 * time.Second}

	for payload := range s.callbackChan {
		if payload.Delay > 0 {
			time.Sleep(payload.Delay)
		}

		s.mu.RLock()
		callbackURL := s.callbackURL
		s.mu.RUnlock()

		if callbackURL == "" {
			continue
		}

		var body []byte
		switch payload.Type {
		case "stk":
			req := payload.Request.(*STKRequest)
			body = s.buildSTKCallback(req, payload.Success)
			s.mu.Lock()
			if payload.Success {
				req.Status = "success"
			} else {
				req.Status = "failed"
			}
			s.mu.Unlock()
		case "b2c":
			req := payload.Request.(*B2CRequest)
			body = s.buildB2CCallback(req, payload.Success)
			s.mu.Lock()
			if payload.Success {
				req.Status = "success"
			} else {
				req.Status = "failed"
			}
			s.mu.Unlock()
		}

		httpReq, err := http.NewRequest("POST", callbackURL, bytes.NewReader(body))
		if err != nil {
			log.Printf("Failed to create callback request: %v", err)
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(httpReq)
		if err != nil {
			log.Printf("Failed to send callback to %s: %v", callbackURL, err)
			continue
		}
		resp.Body.Close()
		log.Printf("Sent %s callback to %s, status: %d", payload.Type, callbackURL, resp.StatusCode)
	}
}

func (s *Server) buildSTKCallback(req *STKRequest, success bool) []byte {
	resultCode := 0
	resultDesc := "The service request is processed successfully."
	if !success {
		resultCode = 1032
		resultDesc = "Request cancelled by user"
	}

	callback := map[string]any{
		"Body": map[string]any{
			"stkCallback": map[string]any{
				"MerchantRequestID": req.MerchantRequestID,
				"CheckoutRequestID": req.CheckoutRequestID,
				"ResultCode":        resultCode,
				"ResultDesc":        resultDesc,
			},
		},
	}

	if success {
		callback["Body"].(map[string]any)["stkCallback"].(map[string]any)["CallbackMetadata"] = map[string]any{
			"Item": []map[string]any{
				{"Name": "Amount", "Value": float64(req.Amount)},
				{"Name": "MpesaReceiptNumber", "Value": fmt.Sprintf("QK%d", time.Now().UnixNano()%10000000000)},
				{"Name": "TransactionDate", "Value": float64(time.Now().Unix())},
				{"Name": "PhoneNumber", "Value": float64(254700000000)},
			},
		}
	}

	body, _ := json.Marshal(callback)
	return body
}

func (s *Server) buildB2CCallback(req *B2CRequest, success bool) []byte {
	resultCode := 0
	resultDesc := "The service request is processed successfully."
	if !success {
		resultCode = 1
		resultDesc = "The balance is insufficient for the transaction."
	}

	callback := map[string]any{
		"Result": map[string]any{
			"ResultType":               0,
			"ResultCode":               resultCode,
			"ResultDesc":               resultDesc,
			"OriginatorConversationID": req.OriginatorConversationID,
			"ConversationID":           req.ConversationID,
			"TransactionID":            fmt.Sprintf("QK%d", time.Now().UnixNano()%10000000000),
		},
	}

	if success {
		callback["Result"].(map[string]any)["ResultParameters"] = map[string]any{
			"ResultParameter": []map[string]any{
				{"Key": "TransactionAmount", "Value": float64(req.Amount)},
				{"Key": "TransactionReceipt", "Value": fmt.Sprintf("QK%d", time.Now().UnixNano()%10000000000)},
				{"Key": "ReceiverPartyPublicName", "Value": "254700***000 - JOHN DOE"},
				{"Key": "TransactionCompletedDateTime", "Value": time.Now().Format("02.01.2006 15:04:05")},
			},
		}
	}

	body, _ := json.Marshal(callback)
	return body
}

// =============================================================================
// Admin Endpoints
// =============================================================================

func (s *Server) triggerCallback(c *fiber.Ctx) error {
	id := c.Params("id")
	success := c.Query("success", "true") == "true"

	s.mu.RLock()
	stkReq, stkExists := s.stkRequests[id]
	b2cReq, b2cExists := s.b2cRequests[id]
	s.mu.RUnlock()

	if stkExists {
		s.callbackChan <- CallbackPayload{Type: "stk", Request: stkReq, Success: success}
		return c.JSON(fiber.Map{"status": "callback triggered", "type": "stk"})
	}

	if b2cExists {
		s.callbackChan <- CallbackPayload{Type: "b2c", Request: b2cReq, Success: success}
		return c.JSON(fiber.Map{"status": "callback triggered", "type": "b2c"})
	}

	return c.Status(404).JSON(fiber.Map{"error": "request not found"})
}

func (s *Server) listRequests(c *fiber.Ctx) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return c.JSON(fiber.Map{
		"stk_requests": s.stkRequests,
		"b2c_requests": s.b2cRequests,
	})
}

func (s *Server) reset(c *fiber.Ctx) error {
	s.mu.Lock()
	s.stkRequests = make(map[string]*STKRequest)
	s.b2cRequests = make(map[string]*B2CRequest)
	s.mu.Unlock()

	return c.JSON(fiber.Map{"status": "reset complete"})
}
