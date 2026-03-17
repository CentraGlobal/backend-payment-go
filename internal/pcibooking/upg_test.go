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

func TestClient_GetPaymentGateways(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/api/paymentGateway" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("api_key") != "test-api-key" {
			t.Errorf("expected api_key query param")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"name":              "Stripe",
				"credential_fields": []string{"api_key", "secret_key"},
			},
			{
				"name":              "Adyen",
				"credential_fields": []string{"merchant_account", "api_key"},
			},
		})
	}))
	defer mockServer.Close()

	client := newTestClient(mockServer.URL)

	gateways, err := client.GetPaymentGateways(context.Background())
	if err != nil {
		t.Fatalf("GetPaymentGateways failed: %v", err)
	}

	if len(gateways) != 2 {
		t.Fatalf("expected 2 gateways, got %d", len(gateways))
	}
	if gateways[0].Name != "Stripe" {
		t.Errorf("expected first gateway name Stripe, got %s", gateways[0].Name)
	}
	if len(gateways[0].CredentialFields) != 2 {
		t.Errorf("expected 2 credential fields, got %d", len(gateways[0].CredentialFields))
	}
}

func TestClient_GetCredentialsStructure(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/api/credentials/Stripe/structure" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"api_key": map[string]any{
				"type":     "string",
				"required": true,
			},
			"secret_key": map[string]any{
				"type":     "string",
				"required": true,
			},
		})
	}))
	defer mockServer.Close()

	client := newTestClient(mockServer.URL)

	structure, err := client.GetCredentialsStructure(context.Background(), "Stripe")
	if err != nil {
		t.Fatalf("GetCredentialsStructure failed: %v", err)
	}

	if _, ok := structure["api_key"]; !ok {
		t.Error("expected api_key in structure")
	}
	if _, ok := structure["secret_key"]; !ok {
		t.Error("expected secret_key in structure")
	}
}

func TestClient_ChargeUPG(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/api/paymentGateway" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["Operation"] != "Charge" {
			t.Errorf("expected Operation=Charge, got %v", body["Operation"])
		}
		if body["CardToken"] != "tok_test123" {
			t.Errorf("expected CardToken=tok_test123, got %v", body["CardToken"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"Status":        "Success",
			"TransactionID": "txn_abc123",
			"Message":       "Payment processed successfully",
		})
	}))
	defer mockServer.Close()

	client := newTestClient(mockServer.URL)

	req := processor.UPGChargeRequest{
		CardToken:     "tok_test123",
		Amount:        150.00,
		Currency:      "USD",
		GatewayName:   "Stripe",
		CredentialsID: "hotel-123-stripe-creds",
	}

	resp, err := client.ChargeUPG(context.Background(), req)
	if err != nil {
		t.Fatalf("ChargeUPG failed: %v", err)
	}

	if resp.Status != "Success" {
		t.Errorf("expected status Success, got %s", resp.Status)
	}
	if resp.TransactionID != "txn_abc123" {
		t.Errorf("expected transaction_id txn_abc123, got %s", resp.TransactionID)
	}
}

func TestClient_ChargeUPG_APIError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid credentials"}`))
	}))
	defer mockServer.Close()

	client := newTestClient(mockServer.URL)

	_, err := client.ChargeUPG(context.Background(), processor.UPGChargeRequest{
		CardToken:     "tok_test",
		Currency:      "USD",
		GatewayName:   "Stripe",
		CredentialsID: "creds-123",
	})
	if err == nil {
		t.Error("expected error on API failure")
	}
}

func TestClient_GetPaymentGateways_APIError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer mockServer.Close()

	client := pcibooking.NewClient("bad-key", mockServer.URL)

	_, err := client.GetPaymentGateways(context.Background())
	if err == nil {
		t.Error("expected error on API failure")
	}
}
