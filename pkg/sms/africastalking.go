package sms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	APIKey   string
	Username string
	Sender   string
	Sandbox  bool
}

type Client struct {
	config     *Config
	httpClient *http.Client
	baseURL    string
}

type SendResponse struct {
	SMSMessageData struct {
		Message    string `json:"Message"`
		Recipients []struct {
			StatusCode int    `json:"statusCode"`
			Number     string `json:"number"`
			Status     string `json:"status"`
			Cost       string `json:"cost"`
			MessageID  string `json:"messageId"`
		} `json:"Recipients"`
	} `json:"SMSMessageData"`
}

func NewClient(cfg *Config) *Client {
	baseURL := "https://api.africastalking.com"
	if cfg.Sandbox {
		baseURL = "https://api.sandbox.africastalking.com"
	}

	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

func (c *Client) Send(to, message string) error {
	endpoint := fmt.Sprintf("%s/version1/messaging", c.baseURL)

	data := url.Values{}
	data.Set("username", c.config.Username)
	data.Set("to", to)
	data.Set("message", message)
	if c.config.Sender != "" {
		data.Set("from", c.config.Sender)
	}

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("apiKey", c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SMS API returned status %d", resp.StatusCode)
	}

	var result SendResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.SMSMessageData.Recipients) > 0 {
		recipient := result.SMSMessageData.Recipients[0]
		if recipient.StatusCode != 101 {
			return fmt.Errorf("SMS failed: %s", recipient.Status)
		}
	}

	return nil
}

func (c *Client) SendBulk(recipients []string, message string) error {
	to := strings.Join(recipients, ",")
	return c.Send(to, message)
}

type MockClient struct {
	SentMessages []MockMessage
}

type MockMessage struct {
	To      string
	Message string
}

func NewMockClient() *MockClient {
	return &MockClient{
		SentMessages: make([]MockMessage, 0),
	}
}

func (c *MockClient) Send(to, message string) error {
	c.SentMessages = append(c.SentMessages, MockMessage{To: to, Message: message})
	return nil
}

var _ = bytes.NewBuffer
