package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// =============================================================================
// Integration Test Harness
// =============================================================================
// This package provides utilities for running integration tests against
// mock provider servers. It handles setup, teardown, and HTTP client helpers.
// =============================================================================

// Config holds the mock server URLs
type Config struct {
	MPesaURL          string
	AfricasTalkingURL string
	AlpacaURL         string
}

// DefaultConfig returns the default configuration for local testing
func DefaultConfig() *Config {
	return &Config{
		MPesaURL:          getEnvOrDefault("MPESA_MOCK_URL", "http://localhost:8090"),
		AfricasTalkingURL: getEnvOrDefault("AT_MOCK_URL", "http://localhost:8091"),
		AlpacaURL:         getEnvOrDefault("ALPACA_MOCK_URL", "http://localhost:8092"),
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// Harness provides utilities for integration tests
type Harness struct {
	t      *testing.T
	config *Config
	client *http.Client
}

// NewHarness creates a new test harness
func NewHarness(t *testing.T) *Harness {
	return &Harness{
		t:      t,
		config: DefaultConfig(),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Config returns the harness configuration
func (h *Harness) Config() *Config {
	return h.config
}

// =============================================================================
// HTTP Helpers
// =============================================================================

// Request represents an HTTP request configuration
type Request struct {
	Method  string
	URL     string
	Body    any
	Headers map[string]string
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Do executes an HTTP request and returns the response
func (h *Harness) Do(req Request) (*Response, error) {
	var body io.Reader
	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		body = bytes.NewReader(jsonBody)
	}

	httpReq, err := http.NewRequest(req.Method, req.URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := h.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Headers:    resp.Header,
	}, nil
}

// JSON unmarshals the response body into the given value
func (r *Response) JSON(v any) error {
	return json.Unmarshal(r.Body, v)
}

// =============================================================================
// Mock Server Helpers
// =============================================================================

// ResetMPesa resets the M-Pesa mock server state
func (h *Harness) ResetMPesa() error {
	resp, err := h.Do(Request{
		Method: "POST",
		URL:    h.config.MPesaURL + "/admin/reset",
	})
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("reset failed with status %d", resp.StatusCode)
	}
	return nil
}

// ResetAfricasTalking resets the Africa's Talking mock server state
func (h *Harness) ResetAfricasTalking() error {
	resp, err := h.Do(Request{
		Method: "POST",
		URL:    h.config.AfricasTalkingURL + "/admin/reset",
	})
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("reset failed with status %d", resp.StatusCode)
	}
	return nil
}

// ResetAlpaca resets the Alpaca mock server state
func (h *Harness) ResetAlpaca() error {
	resp, err := h.Do(Request{
		Method: "POST",
		URL:    h.config.AlpacaURL + "/admin/reset",
	})
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("reset failed with status %d", resp.StatusCode)
	}
	return nil
}

// ResetAll resets all mock servers
func (h *Harness) ResetAll() error {
	if err := h.ResetMPesa(); err != nil {
		return fmt.Errorf("mpesa reset failed: %w", err)
	}
	if err := h.ResetAfricasTalking(); err != nil {
		return fmt.Errorf("africastalking reset failed: %w", err)
	}
	if err := h.ResetAlpaca(); err != nil {
		return fmt.Errorf("alpaca reset failed: %w", err)
	}
	return nil
}

// =============================================================================
// Health Checks
// =============================================================================

// WaitForMPesa waits for the M-Pesa mock server to be ready
func (h *Harness) WaitForMPesa(timeout time.Duration) error {
	return h.waitForHealth(h.config.MPesaURL+"/health", timeout)
}

// WaitForAfricasTalking waits for the Africa's Talking mock server to be ready
func (h *Harness) WaitForAfricasTalking(timeout time.Duration) error {
	return h.waitForHealth(h.config.AfricasTalkingURL+"/health", timeout)
}

// WaitForAlpaca waits for the Alpaca mock server to be ready
func (h *Harness) WaitForAlpaca(timeout time.Duration) error {
	return h.waitForHealth(h.config.AlpacaURL+"/health", timeout)
}

// WaitForAll waits for all mock servers to be ready
func (h *Harness) WaitForAll(timeout time.Duration) error {
	if err := h.WaitForMPesa(timeout); err != nil {
		return fmt.Errorf("mpesa not ready: %w", err)
	}
	if err := h.WaitForAfricasTalking(timeout); err != nil {
		return fmt.Errorf("africastalking not ready: %w", err)
	}
	if err := h.WaitForAlpaca(timeout); err != nil {
		return fmt.Errorf("alpaca not ready: %w", err)
	}
	return nil
}

func (h *Harness) waitForHealth(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := h.client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", url)
}

// =============================================================================
// Assertions
// =============================================================================

// AssertStatus checks that the response has the expected status code
func (h *Harness) AssertStatus(resp *Response, expected int) {
	if resp.StatusCode != expected {
		h.t.Errorf("Expected status %d, got %d. Body: %s", expected, resp.StatusCode, string(resp.Body))
	}
}

// AssertJSONField checks that a JSON response has a field with the expected value
func (h *Harness) AssertJSONField(resp *Response, field string, expected any) {
	var data map[string]any
	if err := resp.JSON(&data); err != nil {
		h.t.Errorf("Failed to parse JSON: %v", err)
		return
	}

	actual, ok := data[field]
	if !ok {
		h.t.Errorf("Field %s not found in response", field)
		return
	}

	if actual != expected {
		h.t.Errorf("Field %s: expected %v, got %v", field, expected, actual)
	}
}
