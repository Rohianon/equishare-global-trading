package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/viper"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

type APIError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func New() *Client {
	return &Client{
		baseURL: viper.GetString("api_url"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) do(method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error != "" {
			return fmt.Errorf("%s", apiErr.Error)
		}
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Message != "" {
			return fmt.Errorf("%s", apiErr.Message)
		}
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// Auth endpoints

type RegisterRequest struct {
	Phone    string `json:"phone"`
	FullName string `json:"full_name,omitempty"`
}

type RegisterResponse struct {
	Message   string `json:"message"`
	ExpiresIn int    `json:"expires_in"`
}

func (c *Client) Register(phone, fullName string) (*RegisterResponse, error) {
	var resp RegisterResponse
	err := c.do("POST", "/api/v1/auth/register", RegisterRequest{
		Phone:    phone,
		FullName: fullName,
	}, &resp)
	return &resp, err
}

type VerifyRequest struct {
	Phone string `json:"phone"`
	OTP   string `json:"otp"`
	PIN   string `json:"pin"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	User         struct {
		ID       string `json:"id"`
		Phone    string `json:"phone"`
		FullName string `json:"full_name"`
	} `json:"user"`
}

func (c *Client) Verify(phone, otp, pin string) (*AuthResponse, error) {
	var resp AuthResponse
	err := c.do("POST", "/api/v1/auth/verify", VerifyRequest{
		Phone: phone,
		OTP:   otp,
		PIN:   pin,
	}, &resp)
	return &resp, err
}

type LoginRequest struct {
	Phone string `json:"phone"`
	PIN   string `json:"pin"`
}

func (c *Client) Login(phone, pin string) (*AuthResponse, error) {
	var resp AuthResponse
	err := c.do("POST", "/api/v1/auth/login", LoginRequest{
		Phone: phone,
		PIN:   pin,
	}, &resp)
	return &resp, err
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (c *Client) Refresh(refreshToken string) (*AuthResponse, error) {
	var resp AuthResponse
	err := c.do("POST", "/api/v1/auth/refresh", RefreshRequest{
		RefreshToken: refreshToken,
	}, &resp)
	return &resp, err
}

type MeResponse struct {
	ID        string `json:"id"`
	Phone     string `json:"phone"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	KYCStatus string `json:"kyc_status"`
	CreatedAt string `json:"created_at"`
}

func (c *Client) Me() (*MeResponse, error) {
	var resp MeResponse
	err := c.do("GET", "/api/v1/auth/me", nil, &resp)
	return &resp, err
}

// Wallet endpoints

type WalletBalance struct {
	Currency  string  `json:"currency"`
	Available float64 `json:"available"`
	Pending   float64 `json:"pending"`
	Total     float64 `json:"total"`
}

func (c *Client) GetWalletBalance() (*WalletBalance, error) {
	var resp WalletBalance
	err := c.do("GET", "/api/v1/payments/wallet/balance", nil, &resp)
	return &resp, err
}

type DepositRequest struct {
	Amount      float64 `json:"amount"`
	PhoneNumber string  `json:"phone_number"`
}

type DepositResponse struct {
	TransactionID   string `json:"transaction_id"`
	CheckoutID      string `json:"checkout_request_id"`
	Status          string `json:"status"`
	Message         string `json:"message"`
	MerchantRequest string `json:"merchant_request_id"`
}

func (c *Client) InitiateDeposit(amount float64, phone string) (*DepositResponse, error) {
	var resp DepositResponse
	err := c.do("POST", "/api/v1/payments/deposit", DepositRequest{
		Amount:      amount,
		PhoneNumber: phone,
	}, &resp)
	return &resp, err
}

type Transaction struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Status      string  `json:"status"`
	Description string  `json:"description"`
	CreatedAt   string  `json:"created_at"`
}

type TransactionsResponse struct {
	Transactions []Transaction `json:"transactions"`
	Total        int           `json:"total"`
	Page         int           `json:"page"`
	PerPage      int           `json:"per_page"`
}

func (c *Client) GetTransactions(page, perPage int) (*TransactionsResponse, error) {
	var resp TransactionsResponse
	path := fmt.Sprintf("/api/v1/payments/transactions?page=%d&per_page=%d", page, perPage)
	err := c.do("GET", path, nil, &resp)
	return &resp, err
}
