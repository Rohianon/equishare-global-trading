package kyc

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// =============================================================================
// KYC Provider Integration
// =============================================================================
// This package integrates with Smile Identity (or similar KYC providers) for
// identity verification. It handles:
// - Creating verification sessions
// - Submitting user documents
// - Processing webhook callbacks
// - Tracking verification status
// =============================================================================

// Status represents the KYC verification status
type Status string

const (
	StatusPending    Status = "pending"
	StatusSubmitted  Status = "submitted"
	StatusProcessing Status = "processing"
	StatusApproved   Status = "approved"
	StatusRejected   Status = "rejected"
	StatusExpired    Status = "expired"
	StatusFailed     Status = "failed"
)

// Tier represents the KYC tier/level
type Tier int

const (
	TierNone     Tier = 0 // No KYC
	TierBasic    Tier = 1 // Basic identity verification
	TierStandard Tier = 2 // ID document + selfie
	TierEnhanced Tier = 3 // Full document verification + address
)

// DocumentType represents the type of identity document
type DocumentType string

const (
	DocTypeNationalID     DocumentType = "national_id"
	DocTypePassport       DocumentType = "passport"
	DocTypeDriversLicense DocumentType = "drivers_license"
	DocTypeSelfie         DocumentType = "selfie"
	DocTypeProofOfAddress DocumentType = "proof_of_address"
)

// Config holds KYC provider configuration
type Config struct {
	APIKey      string
	PartnerID   string
	BaseURL     string
	WebhookKey  string // For webhook signature verification
	CallbackURL string
	Environment string // "sandbox" or "production"
}

// Client handles KYC provider API interactions
type Client struct {
	config     *Config
	httpClient *http.Client
}

// NewClient creates a new KYC client
func NewClient(config *Config) *Client {
	if config.BaseURL == "" {
		if config.Environment == "production" {
			config.BaseURL = "https://api.smileidentity.com/v1"
		} else {
			config.BaseURL = "https://testapi.smileidentity.com/v1"
		}
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// =============================================================================
// Data Models
// =============================================================================

// Session represents a KYC verification session
type Session struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"user_id"`
	ProviderRef    string                 `json:"provider_ref"`    // Provider's reference ID
	Status         Status                 `json:"status"`
	Tier           Tier                   `json:"tier"`
	JobType        string                 `json:"job_type"`
	SubmittedAt    *time.Time             `json:"submitted_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	ExpiresAt      time.Time              `json:"expires_at"`
	ResultCode     string                 `json:"result_code,omitempty"`
	ResultMessage  string                 `json:"result_message,omitempty"`
	Confidence     float64                `json:"confidence,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// Document represents a submitted KYC document
type Document struct {
	ID          string       `json:"id"`
	SessionID   string       `json:"session_id"`
	Type        DocumentType `json:"type"`
	FileName    string       `json:"file_name,omitempty"`
	FileURL     string       `json:"file_url,omitempty"`
	FileHash    string       `json:"file_hash,omitempty"`
	Status      Status       `json:"status"`
	UploadedAt  time.Time    `json:"uploaded_at"`
	VerifiedAt  *time.Time   `json:"verified_at,omitempty"`
}

// UserInfo contains user information for KYC verification
type UserInfo struct {
	UserID      string `json:"user_id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	MiddleName  string `json:"middle_name,omitempty"`
	DateOfBirth string `json:"dob,omitempty"` // YYYY-MM-DD
	IDNumber    string `json:"id_number,omitempty"`
	IDType      string `json:"id_type,omitempty"`
	Country     string `json:"country"` // ISO 3166-1 alpha-2
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
}

// CreateSessionRequest is the request to create a KYC session
type CreateSessionRequest struct {
	UserInfo    UserInfo     `json:"user_info"`
	JobType     string       `json:"job_type"`     // e.g., "biometric_kyc", "document_verification"
	RequestedTier Tier       `json:"requested_tier"`
	CallbackURL string       `json:"callback_url,omitempty"`
}

// CreateSessionResponse is the response from creating a KYC session
type CreateSessionResponse struct {
	SessionID   string    `json:"session_id"`
	ProviderRef string    `json:"provider_ref"`
	UploadURL   string    `json:"upload_url,omitempty"` // Pre-signed URL for document upload
	ExpiresAt   time.Time `json:"expires_at"`
}

// SubmitDocumentRequest is the request to submit a document
type SubmitDocumentRequest struct {
	SessionID string       `json:"session_id"`
	DocType   DocumentType `json:"doc_type"`
	ImageData []byte       `json:"-"`              // Base64 encoded in JSON
	ImageB64  string       `json:"image,omitempty"`
	FileURL   string       `json:"file_url,omitempty"`
}

// WebhookPayload represents the webhook callback from the provider
type WebhookPayload struct {
	PartnerID     string                 `json:"partner_id"`
	SessionID     string                 `json:"session_id"`
	JobID         string                 `json:"job_id"`
	ResultCode    string                 `json:"result_code"`
	ResultText    string                 `json:"result_text"`
	Confidence    float64                `json:"confidence"`
	IsFinal       bool                   `json:"is_final"`
	Timestamp     time.Time              `json:"timestamp"`
	Actions       map[string]interface{} `json:"actions,omitempty"`
	Source        string                 `json:"source"`
	Signature     string                 `json:"signature"`
	RawPayload    []byte                 `json:"-"`
}

// =============================================================================
// API Methods
// =============================================================================

// CreateSession creates a new KYC verification session
func (c *Client) CreateSession(ctx context.Context, req *CreateSessionRequest) (*CreateSessionResponse, error) {
	if req.CallbackURL == "" {
		req.CallbackURL = c.config.CallbackURL
	}

	// Build provider request
	providerReq := map[string]interface{}{
		"partner_id":   c.config.PartnerID,
		"timestamp":    time.Now().Unix(),
		"partner_params": map[string]interface{}{
			"user_id":  req.UserInfo.UserID,
			"job_id":   fmt.Sprintf("kyc_%s_%d", req.UserInfo.UserID, time.Now().Unix()),
			"job_type": req.JobType,
		},
		"id_info": map[string]interface{}{
			"first_name":  req.UserInfo.FirstName,
			"last_name":   req.UserInfo.LastName,
			"middle_name": req.UserInfo.MiddleName,
			"dob":         req.UserInfo.DateOfBirth,
			"id_number":   req.UserInfo.IDNumber,
			"id_type":     req.UserInfo.IDType,
			"country":     req.UserInfo.Country,
		},
		"callback_url": req.CallbackURL,
	}

	body, err := json.Marshal(providerReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/upload", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result struct {
		UploadURL string `json:"upload_url"`
		RefID     string `json:"ref_id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &CreateSessionResponse{
		SessionID:   fmt.Sprintf("kyc_%s_%d", req.UserInfo.UserID, time.Now().Unix()),
		ProviderRef: result.RefID,
		UploadURL:   result.UploadURL,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}, nil
}

// CheckStatus queries the provider for the current status of a session
func (c *Client) CheckStatus(ctx context.Context, providerRef string) (*Session, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/job_status?partner_id=%s&job_id=%s",
			c.config.BaseURL, c.config.PartnerID, providerRef), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result struct {
		ResultCode string  `json:"result_code"`
		ResultText string  `json:"result_text"`
		Confidence float64 `json:"confidence"`
		IsFinal    bool    `json:"is_final"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	status := StatusProcessing
	if result.IsFinal {
		if result.ResultCode == "0810" || result.ResultCode == "0820" {
			status = StatusApproved
		} else {
			status = StatusRejected
		}
	}

	return &Session{
		ProviderRef:   providerRef,
		Status:        status,
		ResultCode:    result.ResultCode,
		ResultMessage: result.ResultText,
		Confidence:    result.Confidence,
	}, nil
}

// =============================================================================
// Webhook Handling
// =============================================================================

// VerifyWebhookSignature verifies the webhook signature
func (c *Client) VerifyWebhookSignature(payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(c.config.WebhookKey))
	mac.Write(payload)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ParseWebhook parses and validates a webhook payload
func (c *Client) ParseWebhook(body []byte, signature string) (*WebhookPayload, error) {
	if !c.VerifyWebhookSignature(body, signature) {
		return nil, fmt.Errorf("invalid webhook signature")
	}

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	payload.RawPayload = body
	return &payload, nil
}

// MapWebhookToSession maps a webhook payload to a session update
func (c *Client) MapWebhookToSession(payload *WebhookPayload) (*Session, error) {
	status := StatusProcessing
	if payload.IsFinal {
		// Smile Identity result codes:
		// 0810 - Exact match
		// 0820 - Partial match (verified)
		// 1000+ - Various failure codes
		if payload.ResultCode == "0810" || payload.ResultCode == "0820" {
			status = StatusApproved
		} else {
			status = StatusRejected
		}
	}

	now := time.Now()
	return &Session{
		ProviderRef:   payload.JobID,
		Status:        status,
		ResultCode:    payload.ResultCode,
		ResultMessage: payload.ResultText,
		Confidence:    payload.Confidence,
		CompletedAt:   &now,
		UpdatedAt:     now,
	}, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// DetermineKYCTier determines the KYC tier based on completed verifications
func DetermineKYCTier(session *Session, docs []Document) Tier {
	if session.Status != StatusApproved {
		return TierNone
	}

	hasSelfie := false
	hasID := false
	hasAddress := false

	for _, doc := range docs {
		if doc.Status != StatusApproved {
			continue
		}
		switch doc.Type {
		case DocTypeSelfie:
			hasSelfie = true
		case DocTypeNationalID, DocTypePassport, DocTypeDriversLicense:
			hasID = true
		case DocTypeProofOfAddress:
			hasAddress = true
		}
	}

	if hasID && hasSelfie && hasAddress {
		return TierEnhanced
	}
	if hasID && hasSelfie {
		return TierStandard
	}
	if hasID || hasSelfie {
		return TierBasic
	}
	return TierNone
}

// IsTransitionAllowed checks if a status transition is valid
func IsTransitionAllowed(from, to Status) bool {
	allowed := map[Status][]Status{
		StatusPending:    {StatusSubmitted, StatusExpired, StatusFailed},
		StatusSubmitted:  {StatusProcessing, StatusFailed},
		StatusProcessing: {StatusApproved, StatusRejected, StatusFailed},
		StatusApproved:   {}, // Terminal state
		StatusRejected:   {StatusPending}, // Can retry
		StatusExpired:    {StatusPending}, // Can retry
		StatusFailed:     {StatusPending}, // Can retry
	}

	for _, s := range allowed[from] {
		if s == to {
			return true
		}
	}
	return false
}
