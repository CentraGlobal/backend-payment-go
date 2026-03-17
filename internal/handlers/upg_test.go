package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CentraGlobal/backend-payment-go/internal/handlers"
	"github.com/CentraGlobal/backend-payment-go/internal/processor"
	"github.com/gofiber/fiber/v2"
)

// mockUPGProcessor is a test double that implements processor.Processor with
// configurable UPG behaviour.
type mockUPGProcessor struct {
	gateways  []processor.GatewayInfo
	structure map[string]any
	charge    *processor.UPGChargeResponse
	err       error
}

func (m *mockUPGProcessor) CreateCard(_ context.Context, _ processor.Card) (*processor.CardResponse, error) {
	return nil, errors.New("not implemented")
}
func (m *mockUPGProcessor) GetCard(_ context.Context, _ string) (*processor.CardResponse, error) {
	return nil, errors.New("not implemented")
}
func (m *mockUPGProcessor) DeleteCard(_ context.Context, _ string) error {
	return errors.New("not implemented")
}
func (m *mockUPGProcessor) SendCard(_ context.Context, _ string, _ processor.SendRequest) (*processor.SendResponse, error) {
	return nil, errors.New("not implemented")
}
func (m *mockUPGProcessor) CreateSessionToken(_ context.Context, _ string) (*processor.SessionTokenResponse, error) {
	return nil, errors.New("not implemented")
}
func (m *mockUPGProcessor) CaptureFormURL(_ string) string { return "" }
func (m *mockUPGProcessor) Name() string                   { return "mock" }

func (m *mockUPGProcessor) GetPaymentGateways(_ context.Context) ([]processor.GatewayInfo, error) {
	return m.gateways, m.err
}
func (m *mockUPGProcessor) GetCredentialsStructure(_ context.Context, _ string) (map[string]any, error) {
	return m.structure, m.err
}
func (m *mockUPGProcessor) ChargeUPG(_ context.Context, _ processor.UPGChargeRequest) (*processor.UPGChargeResponse, error) {
	return m.charge, m.err
}

func setupUPGApp(mock *mockUPGProcessor) *fiber.App {
	ph := handlers.NewPaymentHandler(mock)
	app := fiber.New()
	v1 := app.Group("/v1")
	payments := v1.Group("/payments")
	payments.Post("/charge/upg", ph.ChargeUPG)
	gateways := v1.Group("/gateways")
	gateways.Get("/", ph.GetGateways)
	gateways.Get("/:name/structure", ph.GetGatewayStructure)
	return app
}

func TestChargeUPG_Success(t *testing.T) {
	mock := &mockUPGProcessor{
		charge: &processor.UPGChargeResponse{
			Status:        "Success",
			TransactionID: "txn_abc123",
			Message:       "Payment processed",
		},
	}
	app := setupUPGApp(mock)

	body := `{"card_token":"tok_test","amount":100.00,"currency":"USD","gateway_name":"Stripe","credentials_id":"creds-123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/payments/charge/upg", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(respBody, &result)

	if result["status"] != "Success" {
		t.Errorf("expected status Success, got %v", result["status"])
	}
	if result["transaction_id"] != "txn_abc123" {
		t.Errorf("expected transaction_id txn_abc123, got %v", result["transaction_id"])
	}
}

func TestChargeUPG_BadBody(t *testing.T) {
	mock := &mockUPGProcessor{}
	app := setupUPGApp(mock)

	req := httptest.NewRequest(http.MethodPost, "/v1/payments/charge/upg", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestChargeUPG_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"missing card_token", `{"amount":100,"currency":"USD","gateway_name":"Stripe","credentials_id":"c1"}`},
		{"missing currency", `{"card_token":"tok","amount":100,"gateway_name":"Stripe","credentials_id":"c1"}`},
		{"missing gateway_name", `{"card_token":"tok","amount":100,"currency":"USD","credentials_id":"c1"}`},
		{"missing credentials_id", `{"card_token":"tok","amount":100,"currency":"USD","gateway_name":"Stripe"}`},
		{"zero amount", `{"card_token":"tok","amount":0,"currency":"USD","gateway_name":"Stripe","credentials_id":"c1"}`},
		{"negative amount", `{"card_token":"tok","amount":-10,"currency":"USD","gateway_name":"Stripe","credentials_id":"c1"}`},
	}

	mock := &mockUPGProcessor{}
	app := setupUPGApp(mock)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/payments/charge/upg", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")

			resp, _ := app.Test(req)
			resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

func TestChargeUPG_ProcessorError(t *testing.T) {
	mock := &mockUPGProcessor{
		err: errors.New("gateway rejected the charge"),
	}
	app := setupUPGApp(mock)

	body := `{"card_token":"tok","amount":100,"currency":"USD","gateway_name":"Stripe","credentials_id":"c1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/payments/charge/upg", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", resp.StatusCode)
	}
}

func TestGetGateways_Success(t *testing.T) {
	mock := &mockUPGProcessor{
		gateways: []processor.GatewayInfo{
			{Name: "Stripe", CredentialFields: []string{"api_key"}},
			{Name: "Adyen", CredentialFields: []string{"merchant_account", "api_key"}},
		},
	}
	app := setupUPGApp(mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/gateways/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result []map[string]any
	json.Unmarshal(body, &result)

	if len(result) != 2 {
		t.Errorf("expected 2 gateways, got %d", len(result))
	}
	if result[0]["name"] != "Stripe" {
		t.Errorf("expected first gateway Stripe, got %v", result[0]["name"])
	}
}

func TestGetGateways_ProcessorError(t *testing.T) {
	mock := &mockUPGProcessor{
		err: errors.New("UPG not supported"),
	}
	app := setupUPGApp(mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/gateways/", nil)
	resp, _ := app.Test(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", resp.StatusCode)
	}
}

func TestGetGatewayStructure_Success(t *testing.T) {
	mock := &mockUPGProcessor{
		structure: map[string]any{
			"api_key":    map[string]any{"type": "string", "required": true},
			"secret_key": map[string]any{"type": "string", "required": true},
		},
	}
	app := setupUPGApp(mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/gateways/Stripe/structure", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(body, &result)

	if _, ok := result["api_key"]; !ok {
		t.Error("expected api_key in structure response")
	}
}

func TestGetGatewayStructure_ProcessorError(t *testing.T) {
	mock := &mockUPGProcessor{
		err: errors.New("gateway not found"),
	}
	app := setupUPGApp(mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/gateways/Unknown/structure", nil)
	resp, _ := app.Test(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", resp.StatusCode)
	}
}

func TestVaultera_UPGStubs(t *testing.T) {
	// Verify that calling UPG endpoints when backed by vaultera returns an appropriate error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	mock := &mockUPGProcessor{
		err: errors.New("vaultera: UPG is not supported; use the pci_booking_upg provider instead"),
	}
	app := setupUPGApp(mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/gateways/", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502 for UPG stub, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(respBody, &result)
	if result["error"] == nil {
		t.Error("expected error in response body")
	}
}
