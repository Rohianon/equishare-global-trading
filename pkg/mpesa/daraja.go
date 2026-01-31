package mpesa

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Config struct {
	ConsumerKey    string
	ConsumerSecret string
	PassKey        string
	ShortCode      string
	CallbackURL    string
	Sandbox        bool
}

type Client struct {
	config      *Config
	httpClient  *http.Client
	baseURL     string
	accessToken string
	tokenExpiry time.Time
	mu          sync.RWMutex
}

func NewClient(cfg *Config) *Client {
	baseURL := "https://api.safaricom.co.ke"
	if cfg.Sandbox {
		baseURL = "https://sandbox.safaricom.co.ke"
	}

	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		token := c.accessToken
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	url := fmt.Sprintf("%s/oauth/v1/generate?grant_type=client_credentials", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString(
		[]byte(c.config.ConsumerKey + ":" + c.config.ConsumerSecret),
	)
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(55 * time.Minute)

	return c.accessToken, nil
}

type STKPushRequest struct {
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

type STKPushResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

func (c *Client) STKPush(ctx context.Context, phone string, amount int, reference string) (*STKPushResponse, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format("20060102150405")
	password := base64.StdEncoding.EncodeToString(
		[]byte(c.config.ShortCode + c.config.PassKey + timestamp),
	)

	reqBody := STKPushRequest{
		BusinessShortCode: c.config.ShortCode,
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            amount,
		PartyA:            phone,
		PartyB:            c.config.ShortCode,
		PhoneNumber:       phone,
		CallBackURL:       c.config.CallbackURL,
		AccountReference:  reference,
		TransactionDesc:   "EquiShare Deposit",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/mpesa/stkpush/v1/processrequest", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send STK push: %w", err)
	}
	defer resp.Body.Close()

	var stkResp STKPushResponse
	if err := json.NewDecoder(resp.Body).Decode(&stkResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if stkResp.ResponseCode != "0" {
		return nil, fmt.Errorf("STK push failed: %s", stkResp.ResponseDescription)
	}

	return &stkResp, nil
}

type STKCallback struct {
	Body struct {
		StkCallback struct {
			MerchantRequestID string `json:"MerchantRequestID"`
			CheckoutRequestID string `json:"CheckoutRequestID"`
			ResultCode        int    `json:"ResultCode"`
			ResultDesc        string `json:"ResultDesc"`
			CallbackMetadata  *struct {
				Item []CallbackItem `json:"Item"`
			} `json:"CallbackMetadata"`
		} `json:"stkCallback"`
	} `json:"Body"`
}

type CallbackItem struct {
	Name  string `json:"Name"`
	Value any    `json:"Value"`
}

type CallbackData struct {
	MerchantRequestID string
	CheckoutRequestID string
	ResultCode        int
	ResultDesc        string
	Amount            float64
	MpesaReceiptNo    string
	TransactionDate   string
	PhoneNumber       string
}

func ParseCallback(callback *STKCallback) *CallbackData {
	data := &CallbackData{
		MerchantRequestID: callback.Body.StkCallback.MerchantRequestID,
		CheckoutRequestID: callback.Body.StkCallback.CheckoutRequestID,
		ResultCode:        callback.Body.StkCallback.ResultCode,
		ResultDesc:        callback.Body.StkCallback.ResultDesc,
	}

	if callback.Body.StkCallback.CallbackMetadata != nil {
		for _, item := range callback.Body.StkCallback.CallbackMetadata.Item {
			switch item.Name {
			case "Amount":
				if v, ok := item.Value.(float64); ok {
					data.Amount = v
				}
			case "MpesaReceiptNumber":
				if v, ok := item.Value.(string); ok {
					data.MpesaReceiptNo = v
				}
			case "TransactionDate":
				if v, ok := item.Value.(float64); ok {
					data.TransactionDate = fmt.Sprintf("%.0f", v)
				}
			case "PhoneNumber":
				if v, ok := item.Value.(float64); ok {
					data.PhoneNumber = fmt.Sprintf("%.0f", v)
				}
			}
		}
	}

	return data
}

type MockClient struct {
	Requests    []MockSTKRequest
	B2CRequests []MockB2CRequest
}

type MockSTKRequest struct {
	Phone     string
	Amount    int
	Reference string
}

func NewMockClient() *MockClient {
	return &MockClient{
		Requests:    make([]MockSTKRequest, 0),
		B2CRequests: make([]MockB2CRequest, 0),
	}
}

func (c *MockClient) STKPush(ctx context.Context, phone string, amount int, reference string) (*STKPushResponse, error) {
	c.Requests = append(c.Requests, MockSTKRequest{
		Phone:     phone,
		Amount:    amount,
		Reference: reference,
	})

	return &STKPushResponse{
		MerchantRequestID:   fmt.Sprintf("mock-merchant-%d", time.Now().UnixNano()),
		CheckoutRequestID:   fmt.Sprintf("mock-checkout-%d", time.Now().UnixNano()),
		ResponseCode:        "0",
		ResponseDescription: "Success. Request accepted for processing",
		CustomerMessage:     "Success. Request accepted for processing",
	}, nil
}

// =============================================================================
// B2C (Business to Customer) - Withdrawals
// =============================================================================

// B2CConfig holds additional config needed for B2C transactions
type B2CConfig struct {
	InitiatorName     string
	InitiatorPassword string
	SecurityCredential string // Encrypted password
	QueueTimeoutURL   string
	ResultURL         string
}

// B2CRequest represents a B2C payout request
type B2CRequest struct {
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

// B2CResponse represents the response from a B2C request
type B2CResponse struct {
	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// B2CCommandID represents the type of B2C transaction
type B2CCommandID string

const (
	// BusinessPayment is for payment of salaries, bonuses, etc.
	BusinessPayment B2CCommandID = "BusinessPayment"
	// SalaryPayment is specifically for salary disbursements
	SalaryPayment B2CCommandID = "SalaryPayment"
	// PromotionPayment is for promotional payments
	PromotionPayment B2CCommandID = "PromotionPayment"
)

// B2C initiates a Business to Customer payment (withdrawal)
func (c *Client) B2C(ctx context.Context, phone string, amount int, reference string, b2cConfig *B2CConfig) (*B2CResponse, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	reqBody := B2CRequest{
		InitiatorName:      b2cConfig.InitiatorName,
		SecurityCredential: b2cConfig.SecurityCredential,
		CommandID:          string(BusinessPayment),
		Amount:             amount,
		PartyA:             c.config.ShortCode,
		PartyB:             phone,
		Remarks:            "EquiShare Withdrawal",
		QueueTimeOutURL:    b2cConfig.QueueTimeoutURL,
		ResultURL:          b2cConfig.ResultURL,
		Occasion:           reference,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/mpesa/b2c/v1/paymentrequest", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send B2C request: %w", err)
	}
	defer resp.Body.Close()

	var b2cResp B2CResponse
	if err := json.NewDecoder(resp.Body).Decode(&b2cResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if b2cResp.ResponseCode != "0" {
		return nil, fmt.Errorf("B2C request failed: %s", b2cResp.ResponseDescription)
	}

	return &b2cResp, nil
}

// B2CCallback represents the callback data from M-Pesa for B2C transactions
type B2CCallback struct {
	Result struct {
		ResultType               int    `json:"ResultType"`
		ResultCode               int    `json:"ResultCode"`
		ResultDesc               string `json:"ResultDesc"`
		OriginatorConversationID string `json:"OriginatorConversationID"`
		ConversationID           string `json:"ConversationID"`
		TransactionID            string `json:"TransactionID"`
		ResultParameters         *struct {
			ResultParameter []ResultParameter `json:"ResultParameter"`
		} `json:"ResultParameters"`
	} `json:"Result"`
}

// ResultParameter represents a key-value pair in the result parameters
type ResultParameter struct {
	Key   string `json:"Key"`
	Value any    `json:"Value"`
}

// B2CCallbackData represents parsed B2C callback data
type B2CCallbackData struct {
	ResultCode               int
	ResultDesc               string
	OriginatorConversationID string
	ConversationID           string
	TransactionID            string
	TransactionAmount        float64
	TransactionReceipt       string
	ReceiverPartyPublicName  string
	TransactionCompletedTime string
	B2CUtilityAccountBalance float64
	B2CWorkingAccountBalance float64
	IsSuccess                bool
}

// ParseB2CCallback parses a B2C callback into structured data
func ParseB2CCallback(callback *B2CCallback) *B2CCallbackData {
	data := &B2CCallbackData{
		ResultCode:               callback.Result.ResultCode,
		ResultDesc:               callback.Result.ResultDesc,
		OriginatorConversationID: callback.Result.OriginatorConversationID,
		ConversationID:           callback.Result.ConversationID,
		TransactionID:            callback.Result.TransactionID,
		IsSuccess:                callback.Result.ResultCode == 0,
	}

	if callback.Result.ResultParameters != nil {
		for _, param := range callback.Result.ResultParameters.ResultParameter {
			switch param.Key {
			case "TransactionAmount":
				if v, ok := param.Value.(float64); ok {
					data.TransactionAmount = v
				}
			case "TransactionReceipt":
				if v, ok := param.Value.(string); ok {
					data.TransactionReceipt = v
				}
			case "ReceiverPartyPublicName":
				if v, ok := param.Value.(string); ok {
					data.ReceiverPartyPublicName = v
				}
			case "TransactionCompletedDateTime":
				if v, ok := param.Value.(string); ok {
					data.TransactionCompletedTime = v
				}
			case "B2CUtilityAccountAvailableFunds":
				if v, ok := param.Value.(float64); ok {
					data.B2CUtilityAccountBalance = v
				}
			case "B2CWorkingAccountAvailableFunds":
				if v, ok := param.Value.(float64); ok {
					data.B2CWorkingAccountBalance = v
				}
			}
		}
	}

	return data
}

// =============================================================================
// Withdrawal Status
// =============================================================================

// WithdrawalStatus represents the status of a withdrawal
type WithdrawalStatus string

const (
	WithdrawalPending    WithdrawalStatus = "pending"
	WithdrawalProcessing WithdrawalStatus = "processing"
	WithdrawalSucceeded  WithdrawalStatus = "succeeded"
	WithdrawalFailed     WithdrawalStatus = "failed"
	WithdrawalReversed   WithdrawalStatus = "reversed"
)

// Withdrawal represents a withdrawal record
type Withdrawal struct {
	ID                       string           `json:"id"`
	UserID                   string           `json:"user_id"`
	Phone                    string           `json:"phone"`
	Amount                   int              `json:"amount"`
	Fee                      int              `json:"fee"`
	NetAmount                int              `json:"net_amount"`
	Status                   WithdrawalStatus `json:"status"`
	Reference                string           `json:"reference"`
	ConversationID           string           `json:"conversation_id,omitempty"`
	OriginatorConversationID string           `json:"originator_conversation_id,omitempty"`
	TransactionID            string           `json:"transaction_id,omitempty"`
	ResultCode               int              `json:"result_code,omitempty"`
	ResultDesc               string           `json:"result_desc,omitempty"`
	CreatedAt                time.Time        `json:"created_at"`
	UpdatedAt                time.Time        `json:"updated_at"`
	CompletedAt              *time.Time       `json:"completed_at,omitempty"`
}

// IsStatusTransitionValid checks if a status transition is valid
func IsStatusTransitionValid(from, to WithdrawalStatus) bool {
	transitions := map[WithdrawalStatus][]WithdrawalStatus{
		WithdrawalPending:    {WithdrawalProcessing, WithdrawalFailed},
		WithdrawalProcessing: {WithdrawalSucceeded, WithdrawalFailed},
		WithdrawalSucceeded:  {WithdrawalReversed},
		WithdrawalFailed:     {WithdrawalPending}, // Can retry
		WithdrawalReversed:   {},                  // Terminal
	}

	for _, valid := range transitions[from] {
		if valid == to {
			return true
		}
	}
	return false
}

// =============================================================================
// Mock B2C Client
// =============================================================================

type MockB2CRequest struct {
	Phone     string
	Amount    int
	Reference string
}

func (c *MockClient) B2C(ctx context.Context, phone string, amount int, reference string, b2cConfig *B2CConfig) (*B2CResponse, error) {
	c.B2CRequests = append(c.B2CRequests, MockB2CRequest{
		Phone:     phone,
		Amount:    amount,
		Reference: reference,
	})

	return &B2CResponse{
		ConversationID:           fmt.Sprintf("mock-conv-%d", time.Now().UnixNano()),
		OriginatorConversationID: fmt.Sprintf("mock-orig-%d", time.Now().UnixNano()),
		ResponseCode:             "0",
		ResponseDescription:      "Accept the service request successfully.",
	}, nil
}
