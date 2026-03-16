package pcibooking_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CentraGlobal/backend-payment-go/internal/pcibooking"
	"github.com/CentraGlobal/backend-payment-go/internal/processor"
)

func newTestClient(serverURL string) *pcibooking.Client {
	return pcibooking.NewClient("test-api-key", serverURL)
}

func TestClient_Name(t *testing.T) {
	client := pcibooking.NewClient("key", "https://example.com")
	if client.Name() != "pcibooking" {
		t.Errorf("expected name pcibooking, got %s", client.Name())
	}
}

func TestClient_CreateCard(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/payments/paycard/capture" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("api_key") != "test-api-key" {
			t.Errorf("expected api_key query param")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"paycard": map[string]string{
				"Token":          "pcitok_test123",
				"CardNumberMask": "411111******1111",
				"CardType":       "Visa",
				"CardholderName": "Test User",
				"ExpirationMM":   "12",
				"ExpirationYYYY": "2030",
			},
		})
	}))
	defer mockServer.Close()

	client := newTestClient(mockServer.URL)

	card := processor.Card{
		CardNumber:      "4111111111111111",
		CardholderName:  "Test User",
		ExpirationMonth: "12",
		ExpirationYear:  "2030",
	}

	resp, err := client.CreateCard(context.Background(), card)
	if err != nil {
		t.Fatalf("CreateCard failed: %v", err)
	}

	if resp.CardToken != "pcitok_test123" {
		t.Errorf("expected token pcitok_test123, got %s", resp.CardToken)
	}
	if resp.CardMask != "411111******1111" {
		t.Errorf("expected mask 411111******1111, got %s", resp.CardMask)
	}
}

func TestClient_GetCard(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/payments/paycard" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("token") != "pcitok_test123" {
			t.Errorf("expected token query param")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"paycard": map[string]string{
				"Token":          "pcitok_test123",
				"CardNumberMask": "411111******1111",
				"CardType":       "Visa",
				"CardholderName": "Test User",
				"ExpirationMM":   "12",
				"ExpirationYYYY": "2030",
			},
		})
	}))
	defer mockServer.Close()

	client := newTestClient(mockServer.URL)

	resp, err := client.GetCard(context.Background(), "pcitok_test123")
	if err != nil {
		t.Fatalf("GetCard failed: %v", err)
	}

	if resp.CardToken != "pcitok_test123" {
		t.Errorf("expected token pcitok_test123, got %s", resp.CardToken)
	}
}

func TestClient_DeleteCard(t *testing.T) {
	called := false
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/api/payments/paycard" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("token") != "pcitok_test123" {
			t.Errorf("expected token query param")
		}
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer mockServer.Close()

	client := newTestClient(mockServer.URL)

	err := client.DeleteCard(context.Background(), "pcitok_test123")
	if err != nil {
		t.Fatalf("DeleteCard failed: %v", err)
	}

	if !called {
		t.Error("expected mock server to be called")
	}
}

func TestClient_SendCard(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/payments/paycard/relay" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"statusCode": 200,
			"headers":    map[string]string{"Content-Type": "application/json"},
			"body":       `{"status":"success"}`,
		})
	}))
	defer mockServer.Close()

	client := newTestClient(mockServer.URL)

	req := processor.SendRequest{
		Method:  "POST",
		URL:     "https://example.com/charge",
		Headers: map[string]string{"Authorization": "Bearer test"},
		Body:    `{"amount":1000}`,
	}

	resp, err := client.SendCard(context.Background(), "pcitok_test123", req)
	if err != nil {
		t.Fatalf("SendCard failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestClient_CreateSessionToken(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/payments/session_tokens" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"session_token": map[string]string{
				"token": "st_test_session",
				"scope": "card",
			},
		})
	}))
	defer mockServer.Close()

	client := newTestClient(mockServer.URL)

	resp, err := client.CreateSessionToken(context.Background(), "card")
	if err != nil {
		t.Fatalf("CreateSessionToken failed: %v", err)
	}

	if resp.Token != "st_test_session" {
		t.Errorf("expected token st_test_session, got %s", resp.Token)
	}
}

func TestClient_CaptureFormURL(t *testing.T) {
	client := pcibooking.NewClient("test-key", "https://service.pcibooking.net")
	url := client.CaptureFormURL("st_test123")

	expected := "https://service.pcibooking.net/api/payments/paycard/ui?session_token=st_test123"
	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}

func TestClient_APIError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid request"}`))
	}))
	defer mockServer.Close()

	client := newTestClient(mockServer.URL)

	_, err := client.CreateCard(context.Background(), processor.Card{})
	if err == nil {
		t.Error("expected error on API failure")
	}
}
