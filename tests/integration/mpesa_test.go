package integration

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestMPesaSTKPush(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	h := NewHarness(t)

	// Wait for mock server
	if err := h.WaitForMPesa(30 * time.Second); err != nil {
		t.Skipf("M-Pesa mock not available: %v", err)
	}

	// Reset state
	if err := h.ResetMPesa(); err != nil {
		t.Fatalf("Failed to reset: %v", err)
	}

	// Get OAuth token
	tokenResp, err := h.Do(Request{
		Method: "GET",
		URL:    h.Config().MPesaURL + "/oauth/v1/generate?grant_type=client_credentials",
		Headers: map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte("test:secret")),
		},
	})
	if err != nil {
		t.Fatalf("Token request failed: %v", err)
	}
	h.AssertStatus(tokenResp, 200)

	var tokenData map[string]string
	if err := tokenResp.JSON(&tokenData); err != nil {
		t.Fatalf("Failed to parse token response: %v", err)
	}

	token := tokenData["access_token"]
	if token == "" {
		t.Fatal("No access token in response")
	}

	// Initiate STK Push
	stkResp, err := h.Do(Request{
		Method: "POST",
		URL:    h.Config().MPesaURL + "/mpesa/stkpush/v1/processrequest",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		Body: map[string]any{
			"BusinessShortCode": "174379",
			"Password":          "test",
			"Timestamp":         "20240101120000",
			"TransactionType":   "CustomerPayBillOnline",
			"Amount":            100,
			"PartyA":            "254700000000",
			"PartyB":            "174379",
			"PhoneNumber":       "254700000000",
			"CallBackURL":       "http://localhost:8000/callback",
			"AccountReference":  "TEST123",
			"TransactionDesc":   "Test Payment",
		},
	})
	if err != nil {
		t.Fatalf("STK push failed: %v", err)
	}
	h.AssertStatus(stkResp, 200)

	var stkData map[string]any
	if err := stkResp.JSON(&stkData); err != nil {
		t.Fatalf("Failed to parse STK response: %v", err)
	}

	if stkData["ResponseCode"] != "0" {
		t.Errorf("Expected ResponseCode 0, got %v", stkData["ResponseCode"])
	}

	checkoutReqID, ok := stkData["CheckoutRequestID"].(string)
	if !ok || checkoutReqID == "" {
		t.Error("No CheckoutRequestID in response")
	}

	// Verify request was recorded
	requestsResp, err := h.Do(Request{
		Method: "GET",
		URL:    h.Config().MPesaURL + "/admin/requests",
	})
	if err != nil {
		t.Fatalf("Failed to get requests: %v", err)
	}
	h.AssertStatus(requestsResp, 200)
}

func TestMPesaB2C(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	h := NewHarness(t)

	if err := h.WaitForMPesa(30 * time.Second); err != nil {
		t.Skipf("M-Pesa mock not available: %v", err)
	}

	if err := h.ResetMPesa(); err != nil {
		t.Fatalf("Failed to reset: %v", err)
	}

	// Get token
	tokenResp, err := h.Do(Request{
		Method: "GET",
		URL:    h.Config().MPesaURL + "/oauth/v1/generate?grant_type=client_credentials",
		Headers: map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte("test:secret")),
		},
	})
	if err != nil {
		t.Fatalf("Token request failed: %v", err)
	}

	var tokenData map[string]string
	tokenResp.JSON(&tokenData)
	token := tokenData["access_token"]

	// Initiate B2C
	b2cResp, err := h.Do(Request{
		Method: "POST",
		URL:    h.Config().MPesaURL + "/mpesa/b2c/v1/paymentrequest",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		Body: map[string]any{
			"InitiatorName":      "testapi",
			"SecurityCredential": "test",
			"CommandID":          "BusinessPayment",
			"Amount":             500,
			"PartyA":             "600000",
			"PartyB":             "254700000000",
			"Remarks":            "Withdrawal",
			"QueueTimeOutURL":    "http://localhost:8000/timeout",
			"ResultURL":          "http://localhost:8000/result",
			"Occasion":           "WD001",
		},
	})
	if err != nil {
		t.Fatalf("B2C request failed: %v", err)
	}
	h.AssertStatus(b2cResp, 200)

	var b2cData map[string]any
	if err := b2cResp.JSON(&b2cData); err != nil {
		t.Fatalf("Failed to parse B2C response: %v", err)
	}

	if b2cData["ResponseCode"] != "0" {
		t.Errorf("Expected ResponseCode 0, got %v", b2cData["ResponseCode"])
	}

	if _, ok := b2cData["ConversationID"].(string); !ok {
		t.Error("No ConversationID in response")
	}
}

func TestMPesaInvalidToken(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	h := NewHarness(t)

	if err := h.WaitForMPesa(30 * time.Second); err != nil {
		t.Skipf("M-Pesa mock not available: %v", err)
	}

	// Try STK push without valid token
	resp, err := h.Do(Request{
		Method: "POST",
		URL:    h.Config().MPesaURL + "/mpesa/stkpush/v1/processrequest",
		Headers: map[string]string{
			"Authorization": "Bearer invalid-token",
		},
		Body: map[string]any{
			"BusinessShortCode": "174379",
		},
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	h.AssertStatus(resp, 401)
}
