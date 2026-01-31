package mpesa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	t.Run("sandbox URL", func(t *testing.T) {
		client := NewClient(&Config{
			ConsumerKey:    "key",
			ConsumerSecret: "secret",
			Sandbox:        true,
		})

		if client.baseURL != "https://sandbox.safaricom.co.ke" {
			t.Errorf("baseURL = %s, want sandbox URL", client.baseURL)
		}
	})

	t.Run("production URL", func(t *testing.T) {
		client := NewClient(&Config{
			ConsumerKey:    "key",
			ConsumerSecret: "secret",
			Sandbox:        false,
		})

		if client.baseURL != "https://api.safaricom.co.ke" {
			t.Errorf("baseURL = %s, want production URL", client.baseURL)
		}
	})
}

func TestParseCallback(t *testing.T) {
	t.Run("successful callback", func(t *testing.T) {
		callback := &STKCallback{}
		callback.Body.StkCallback.MerchantRequestID = "merchant-123"
		callback.Body.StkCallback.CheckoutRequestID = "checkout-456"
		callback.Body.StkCallback.ResultCode = 0
		callback.Body.StkCallback.ResultDesc = "Success"
		callback.Body.StkCallback.CallbackMetadata = &struct {
			Item []CallbackItem `json:"Item"`
		}{
			Item: []CallbackItem{
				{Name: "Amount", Value: 100.0},
				{Name: "MpesaReceiptNumber", Value: "PQ12345678"},
				{Name: "TransactionDate", Value: 20240131120000.0},
				{Name: "PhoneNumber", Value: 254712345678.0},
			},
		}

		data := ParseCallback(callback)

		if data.MerchantRequestID != "merchant-123" {
			t.Errorf("MerchantRequestID = %s, want merchant-123", data.MerchantRequestID)
		}
		if data.Amount != 100.0 {
			t.Errorf("Amount = %f, want 100.0", data.Amount)
		}
		if data.MpesaReceiptNo != "PQ12345678" {
			t.Errorf("MpesaReceiptNo = %s, want PQ12345678", data.MpesaReceiptNo)
		}
		if data.ResultCode != 0 {
			t.Errorf("ResultCode = %d, want 0", data.ResultCode)
		}
	})

	t.Run("failed callback (no metadata)", func(t *testing.T) {
		callback := &STKCallback{}
		callback.Body.StkCallback.MerchantRequestID = "merchant-123"
		callback.Body.StkCallback.CheckoutRequestID = "checkout-456"
		callback.Body.StkCallback.ResultCode = 1032
		callback.Body.StkCallback.ResultDesc = "Request cancelled by user"

		data := ParseCallback(callback)

		if data.ResultCode != 1032 {
			t.Errorf("ResultCode = %d, want 1032", data.ResultCode)
		}
		if data.Amount != 0 {
			t.Error("Amount should be 0 for failed callback")
		}
	})
}

func TestParseB2CCallback(t *testing.T) {
	t.Run("successful B2C callback", func(t *testing.T) {
		callback := &B2CCallback{}
		callback.Result.ResultCode = 0
		callback.Result.ResultDesc = "Success"
		callback.Result.OriginatorConversationID = "orig-123"
		callback.Result.ConversationID = "conv-456"
		callback.Result.TransactionID = "PQ87654321"
		callback.Result.ResultParameters = &struct {
			ResultParameter []ResultParameter `json:"ResultParameter"`
		}{
			ResultParameter: []ResultParameter{
				{Key: "TransactionAmount", Value: 500.0},
				{Key: "TransactionReceipt", Value: "PQ87654321"},
				{Key: "ReceiverPartyPublicName", Value: "254712345678 - JOHN DOE"},
				{Key: "TransactionCompletedDateTime", Value: "31.01.2024 12:00:00"},
				{Key: "B2CUtilityAccountAvailableFunds", Value: 10000.0},
				{Key: "B2CWorkingAccountAvailableFunds", Value: 5000.0},
			},
		}

		data := ParseB2CCallback(callback)

		if !data.IsSuccess {
			t.Error("IsSuccess should be true for ResultCode 0")
		}
		if data.TransactionAmount != 500.0 {
			t.Errorf("TransactionAmount = %f, want 500.0", data.TransactionAmount)
		}
		if data.TransactionReceipt != "PQ87654321" {
			t.Errorf("TransactionReceipt = %s, want PQ87654321", data.TransactionReceipt)
		}
		if data.B2CUtilityAccountBalance != 10000.0 {
			t.Errorf("B2CUtilityAccountBalance = %f, want 10000.0", data.B2CUtilityAccountBalance)
		}
	})

	t.Run("failed B2C callback", func(t *testing.T) {
		callback := &B2CCallback{}
		callback.Result.ResultCode = 2001
		callback.Result.ResultDesc = "The initiator information is invalid."
		callback.Result.OriginatorConversationID = "orig-123"
		callback.Result.ConversationID = "conv-456"

		data := ParseB2CCallback(callback)

		if data.IsSuccess {
			t.Error("IsSuccess should be false for non-zero ResultCode")
		}
		if data.ResultCode != 2001 {
			t.Errorf("ResultCode = %d, want 2001", data.ResultCode)
		}
	})
}

func TestIsStatusTransitionValid(t *testing.T) {
	tests := []struct {
		from  WithdrawalStatus
		to    WithdrawalStatus
		valid bool
	}{
		{WithdrawalPending, WithdrawalProcessing, true},
		{WithdrawalPending, WithdrawalFailed, true},
		{WithdrawalProcessing, WithdrawalSucceeded, true},
		{WithdrawalProcessing, WithdrawalFailed, true},
		{WithdrawalSucceeded, WithdrawalReversed, true},
		{WithdrawalFailed, WithdrawalPending, true}, // Retry
		{WithdrawalPending, WithdrawalSucceeded, false},
		{WithdrawalSucceeded, WithdrawalPending, false},
		{WithdrawalReversed, WithdrawalPending, false},
	}

	for _, tt := range tests {
		name := string(tt.from) + "->" + string(tt.to)
		t.Run(name, func(t *testing.T) {
			result := IsStatusTransitionValid(tt.from, tt.to)
			if result != tt.valid {
				t.Errorf("IsStatusTransitionValid(%s, %s) = %v, want %v",
					tt.from, tt.to, result, tt.valid)
			}
		})
	}
}

func TestMockClient_STKPush(t *testing.T) {
	client := NewMockClient()

	resp, err := client.STKPush(context.Background(), "254712345678", 100, "TEST-001")
	if err != nil {
		t.Fatalf("STKPush() error = %v", err)
	}

	if resp.ResponseCode != "0" {
		t.Errorf("ResponseCode = %s, want 0", resp.ResponseCode)
	}

	if len(client.Requests) != 1 {
		t.Errorf("Requests count = %d, want 1", len(client.Requests))
	}

	if client.Requests[0].Phone != "254712345678" {
		t.Errorf("Request phone = %s, want 254712345678", client.Requests[0].Phone)
	}
}

func TestMockClient_B2C(t *testing.T) {
	client := NewMockClient()

	b2cConfig := &B2CConfig{
		InitiatorName:      "TestInitiator",
		SecurityCredential: "encrypted-cred",
		QueueTimeoutURL:    "https://example.com/timeout",
		ResultURL:          "https://example.com/result",
	}

	resp, err := client.B2C(context.Background(), "254712345678", 500, "WD-001", b2cConfig)
	if err != nil {
		t.Fatalf("B2C() error = %v", err)
	}

	if resp.ResponseCode != "0" {
		t.Errorf("ResponseCode = %s, want 0", resp.ResponseCode)
	}

	if len(client.B2CRequests) != 1 {
		t.Errorf("B2CRequests count = %d, want 1", len(client.B2CRequests))
	}

	if client.B2CRequests[0].Amount != 500 {
		t.Errorf("Request amount = %d, want 500", client.B2CRequests[0].Amount)
	}
}

func TestB2C_Integration(t *testing.T) {
	// Mock M-Pesa server
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/v1/generate" {
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-token",
				"expires_in":   "3599",
			})
			return
		}

		if r.URL.Path == "/mpesa/b2c/v1/paymentrequest" {
			// Verify authorization
			if r.Header.Get("Authorization") != "Bearer mock-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			json.NewEncoder(w).Encode(B2CResponse{
				ConversationID:           "conv-123",
				OriginatorConversationID: "orig-456",
				ResponseCode:             "0",
				ResponseDescription:      "Accept the service request successfully.",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer tokenServer.Close()

	client := &Client{
		config: &Config{
			ConsumerKey:    "test-key",
			ConsumerSecret: "test-secret",
			ShortCode:      "600123",
		},
		httpClient: &http.Client{},
		baseURL:    tokenServer.URL,
	}

	b2cConfig := &B2CConfig{
		InitiatorName:      "TestAPI",
		SecurityCredential: "encrypted-credential",
		QueueTimeoutURL:    "https://example.com/timeout",
		ResultURL:          "https://example.com/result",
	}

	resp, err := client.B2C(context.Background(), "254712345678", 1000, "WD-TEST-001", b2cConfig)
	if err != nil {
		t.Fatalf("B2C() error = %v", err)
	}

	if resp.ResponseCode != "0" {
		t.Errorf("ResponseCode = %s, want 0", resp.ResponseCode)
	}
	if resp.ConversationID != "conv-123" {
		t.Errorf("ConversationID = %s, want conv-123", resp.ConversationID)
	}
}
