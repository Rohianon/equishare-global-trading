package kyc

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Run("default sandbox URL", func(t *testing.T) {
		client := NewClient(&Config{
			APIKey:      "test-key",
			PartnerID:   "partner-123",
			Environment: "sandbox",
		})

		if client.config.BaseURL != "https://testapi.smileidentity.com/v1" {
			t.Errorf("BaseURL = %s, want sandbox URL", client.config.BaseURL)
		}
	})

	t.Run("default production URL", func(t *testing.T) {
		client := NewClient(&Config{
			APIKey:      "test-key",
			PartnerID:   "partner-123",
			Environment: "production",
		})

		if client.config.BaseURL != "https://api.smileidentity.com/v1" {
			t.Errorf("BaseURL = %s, want production URL", client.config.BaseURL)
		}
	})

	t.Run("custom URL", func(t *testing.T) {
		client := NewClient(&Config{
			APIKey:    "test-key",
			PartnerID: "partner-123",
			BaseURL:   "https://custom.api.com",
		})

		if client.config.BaseURL != "https://custom.api.com" {
			t.Errorf("BaseURL = %s, want custom URL", client.config.BaseURL)
		}
	})
}

func TestVerifyWebhookSignature(t *testing.T) {
	webhookKey := "test-webhook-secret"
	client := NewClient(&Config{
		APIKey:     "test-key",
		PartnerID:  "partner-123",
		WebhookKey: webhookKey,
	})

	payload := []byte(`{"result_code":"0810","session_id":"test-123"}`)

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(webhookKey))
	mac.Write(payload)
	validSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	t.Run("valid signature", func(t *testing.T) {
		if !client.VerifyWebhookSignature(payload, validSignature) {
			t.Error("Valid signature should verify")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		if client.VerifyWebhookSignature(payload, "invalid-signature") {
			t.Error("Invalid signature should not verify")
		}
	})

	t.Run("wrong payload", func(t *testing.T) {
		wrongPayload := []byte(`{"result_code":"0810","session_id":"different"}`)
		if client.VerifyWebhookSignature(wrongPayload, validSignature) {
			t.Error("Wrong payload should not verify")
		}
	})
}

func TestParseWebhook(t *testing.T) {
	webhookKey := "test-webhook-secret"
	client := NewClient(&Config{
		APIKey:     "test-key",
		PartnerID:  "partner-123",
		WebhookKey: webhookKey,
	})

	webhookData := WebhookPayload{
		PartnerID:  "partner-123",
		SessionID:  "session-456",
		JobID:      "job-789",
		ResultCode: "0810",
		ResultText: "Exact match",
		Confidence: 99.5,
		IsFinal:    true,
		Timestamp:  time.Now(),
	}

	payload, _ := json.Marshal(webhookData)

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(webhookKey))
	mac.Write(payload)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	t.Run("valid webhook", func(t *testing.T) {
		result, err := client.ParseWebhook(payload, signature)
		if err != nil {
			t.Fatalf("ParseWebhook() error = %v", err)
		}

		if result.ResultCode != "0810" {
			t.Errorf("ResultCode = %s, want 0810", result.ResultCode)
		}
		if result.Confidence != 99.5 {
			t.Errorf("Confidence = %f, want 99.5", result.Confidence)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		_, err := client.ParseWebhook(payload, "bad-sig")
		if err == nil {
			t.Error("ParseWebhook() should fail with invalid signature")
		}
	})
}

func TestMapWebhookToSession(t *testing.T) {
	client := NewClient(&Config{APIKey: "test"})

	t.Run("approved result", func(t *testing.T) {
		webhook := &WebhookPayload{
			JobID:      "job-123",
			ResultCode: "0810",
			ResultText: "Exact match",
			Confidence: 99.0,
			IsFinal:    true,
		}

		session, err := client.MapWebhookToSession(webhook)
		if err != nil {
			t.Fatalf("MapWebhookToSession() error = %v", err)
		}

		if session.Status != StatusApproved {
			t.Errorf("Status = %s, want approved", session.Status)
		}
		if session.Confidence != 99.0 {
			t.Errorf("Confidence = %f, want 99.0", session.Confidence)
		}
	})

	t.Run("partial match approved", func(t *testing.T) {
		webhook := &WebhookPayload{
			JobID:      "job-123",
			ResultCode: "0820",
			ResultText: "Partial match",
			Confidence: 85.0,
			IsFinal:    true,
		}

		session, _ := client.MapWebhookToSession(webhook)
		if session.Status != StatusApproved {
			t.Errorf("Status = %s, want approved for partial match", session.Status)
		}
	})

	t.Run("rejected result", func(t *testing.T) {
		webhook := &WebhookPayload{
			JobID:      "job-123",
			ResultCode: "1000",
			ResultText: "No match",
			Confidence: 20.0,
			IsFinal:    true,
		}

		session, _ := client.MapWebhookToSession(webhook)
		if session.Status != StatusRejected {
			t.Errorf("Status = %s, want rejected", session.Status)
		}
	})

	t.Run("processing (not final)", func(t *testing.T) {
		webhook := &WebhookPayload{
			JobID:      "job-123",
			ResultCode: "0810",
			IsFinal:    false,
		}

		session, _ := client.MapWebhookToSession(webhook)
		if session.Status != StatusProcessing {
			t.Errorf("Status = %s, want processing", session.Status)
		}
	})
}

func TestDetermineKYCTier(t *testing.T) {
	approvedSession := &Session{Status: StatusApproved}
	rejectedSession := &Session{Status: StatusRejected}

	t.Run("no verification", func(t *testing.T) {
		tier := DetermineKYCTier(rejectedSession, nil)
		if tier != TierNone {
			t.Errorf("Tier = %d, want TierNone", tier)
		}
	})

	t.Run("basic tier (ID only)", func(t *testing.T) {
		docs := []Document{
			{Type: DocTypeNationalID, Status: StatusApproved},
		}
		tier := DetermineKYCTier(approvedSession, docs)
		if tier != TierBasic {
			t.Errorf("Tier = %d, want TierBasic", tier)
		}
	})

	t.Run("basic tier (selfie only)", func(t *testing.T) {
		docs := []Document{
			{Type: DocTypeSelfie, Status: StatusApproved},
		}
		tier := DetermineKYCTier(approvedSession, docs)
		if tier != TierBasic {
			t.Errorf("Tier = %d, want TierBasic", tier)
		}
	})

	t.Run("standard tier (ID + selfie)", func(t *testing.T) {
		docs := []Document{
			{Type: DocTypeNationalID, Status: StatusApproved},
			{Type: DocTypeSelfie, Status: StatusApproved},
		}
		tier := DetermineKYCTier(approvedSession, docs)
		if tier != TierStandard {
			t.Errorf("Tier = %d, want TierStandard", tier)
		}
	})

	t.Run("enhanced tier (ID + selfie + address)", func(t *testing.T) {
		docs := []Document{
			{Type: DocTypePassport, Status: StatusApproved},
			{Type: DocTypeSelfie, Status: StatusApproved},
			{Type: DocTypeProofOfAddress, Status: StatusApproved},
		}
		tier := DetermineKYCTier(approvedSession, docs)
		if tier != TierEnhanced {
			t.Errorf("Tier = %d, want TierEnhanced", tier)
		}
	})

	t.Run("ignores pending documents", func(t *testing.T) {
		docs := []Document{
			{Type: DocTypeNationalID, Status: StatusApproved},
			{Type: DocTypeSelfie, Status: StatusPending}, // Should be ignored
		}
		tier := DetermineKYCTier(approvedSession, docs)
		if tier != TierBasic {
			t.Errorf("Tier = %d, want TierBasic (pending selfie ignored)", tier)
		}
	})
}

func TestIsTransitionAllowed(t *testing.T) {
	tests := []struct {
		from    Status
		to      Status
		allowed bool
	}{
		{StatusPending, StatusSubmitted, true},
		{StatusPending, StatusExpired, true},
		{StatusSubmitted, StatusProcessing, true},
		{StatusProcessing, StatusApproved, true},
		{StatusProcessing, StatusRejected, true},
		{StatusRejected, StatusPending, true},   // Can retry
		{StatusApproved, StatusRejected, false}, // Terminal
		{StatusApproved, StatusPending, false},  // Terminal
		{StatusPending, StatusApproved, false},  // Can't skip
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			result := IsTransitionAllowed(tt.from, tt.to)
			if result != tt.allowed {
				t.Errorf("IsTransitionAllowed(%s, %s) = %v, want %v",
					tt.from, tt.to, result, tt.allowed)
			}
		})
	}
}

func TestCreateSession(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/upload" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Check authorization header
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Return success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"upload_url": "https://upload.example.com/presigned",
			"ref_id":     "provider-ref-123",
		})
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:      "test-api-key",
		PartnerID:   "partner-123",
		BaseURL:     server.URL,
		CallbackURL: "https://example.com/webhook",
	})

	req := &CreateSessionRequest{
		UserInfo: UserInfo{
			UserID:    "user-123",
			FirstName: "John",
			LastName:  "Doe",
			Country:   "KE",
		},
		JobType:       "biometric_kyc",
		RequestedTier: TierStandard,
	}

	resp, err := client.CreateSession(testContext(), req)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if resp.ProviderRef != "provider-ref-123" {
		t.Errorf("ProviderRef = %s, want provider-ref-123", resp.ProviderRef)
	}
	if resp.UploadURL != "https://upload.example.com/presigned" {
		t.Errorf("UploadURL = %s, want presigned URL", resp.UploadURL)
	}
}

func TestCheckStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/job_status" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result_code": "0810",
			"result_text": "Exact match",
			"confidence":  98.5,
			"is_final":    true,
		})
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:    "test-api-key",
		PartnerID: "partner-123",
		BaseURL:   server.URL,
	})

	session, err := client.CheckStatus(testContext(), "job-123")
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}

	if session.Status != StatusApproved {
		t.Errorf("Status = %s, want approved", session.Status)
	}
	if session.Confidence != 98.5 {
		t.Errorf("Confidence = %f, want 98.5", session.Confidence)
	}
}

// Helper for test context - uses standard context
func testContext() context.Context {
	return context.Background()
}
