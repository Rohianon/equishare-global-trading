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
	Requests []MockSTKRequest
}

type MockSTKRequest struct {
	Phone     string
	Amount    int
	Reference string
}

func NewMockClient() *MockClient {
	return &MockClient{
		Requests: make([]MockSTKRequest, 0),
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
