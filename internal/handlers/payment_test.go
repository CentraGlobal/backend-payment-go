package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CentraGlobal/backend-payment-go/internal/domain"
	"github.com/CentraGlobal/backend-payment-go/internal/handlers"
	"github.com/CentraGlobal/backend-payment-go/internal/vaultera"
	"github.com/gofiber/fiber/v2"
)

// ---------------------------------------------------------------------------
// Mock service implementations
// ---------------------------------------------------------------------------

type mockCardTokenService struct {
	created *domain.CardToken
	err     error
}

func (m *mockCardTokenService) Create(_ context.Context, token *domain.CardToken) (*domain.CardToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	token.ID = "ct_db_001"
	m.created = token
	return token, nil
}
func (m *mockCardTokenService) GetByID(_ context.Context, _ string) (*domain.CardToken, error) {
	return nil, nil
}
func (m *mockCardTokenService) GetByVaultToken(_ context.Context, _, _ string) (*domain.CardToken, error) {
	return nil, nil
}
func (m *mockCardTokenService) UpdateStatus(_ context.Context, _ string, _ domain.CardTokenStatus) error {
	return nil
}

type mockTransactionService struct {
	created *domain.Transaction
	updated *domain.TransactionStatusUpdate
	err     error
}

func (m *mockTransactionService) CreatePending(_ context.Context, tx *domain.Transaction) (*domain.Transaction, error) {
	if m.err != nil {
		return nil, m.err
	}
	tx.ID = "tx_db_001"
	m.created = tx
	return tx, nil
}
func (m *mockTransactionService) UpdateStatus(_ context.Context, _ string, u domain.TransactionStatusUpdate) error {
	m.updated = &u
	return nil
}
func (m *mockTransactionService) GetByID(_ context.Context, _ string) (*domain.Transaction, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// setupPaymentApp builds a Fiber app with a mock Vaultera server and optional
// mock services injected.
func setupPaymentApp(vaulteraHandler http.Handler, svcs handlers.Services) (*fiber.App, *httptest.Server) {
	vSrv := httptest.NewServer(vaulteraHandler)
	client := vaultera.NewClient("test-key", vSrv.URL)
	ph := handlers.NewPaymentHandler(client, svcs)

	app := fiber.New()
	v1 := app.Group("/v1")
	v1.Get("/session", ph.GetSession)
	payments := v1.Group("/payments")
	payments.Post("/tokenize", ph.Tokenize)
	payments.Post("/charge", ph.Charge)
	payments.Get("/cards/:token", ph.GetCard)
	payments.Delete("/cards/:token", ph.DeleteCard)

	return app, vSrv
}

// ---------------------------------------------------------------------------
// Session
// ---------------------------------------------------------------------------

func TestGetSession(t *testing.T) {
	mockVaultera := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/session_tokens" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"session_token": map[string]string{"token": "st_test", "scope": "card"},
		})
	})

	app, srv := setupPaymentApp(mockVaultera, handlers.Services{})
	defer srv.Close()

	req := httptest.NewRequest(http.MethodGet, "/v1/session", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	json.Unmarshal(body, &result)
	if result["token"] != "st_test" {
		t.Errorf("expected token st_test, got %q", result["token"])
	}
}

// ---------------------------------------------------------------------------
// Tokenize
// ---------------------------------------------------------------------------

func TestTokenize(t *testing.T) {
	mockVaultera := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"card": map[string]string{
				"card_token":       "tok_new",
				"card_number_mask": "411111******1111",
				"card_type":        "visa",
				"expiration_month": "12",
				"expiration_year":  "2030",
			},
		})
	})

	app, srv := setupPaymentApp(mockVaultera, handlers.Services{})
	defer srv.Close()

	body := `{"card":{"card_number":"4111111111111111","expiration_month":"12","expiration_year":"2030"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/payments/tokenize", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	json.Unmarshal(respBody, &result)
	if result["card_token"] != "tok_new" {
		t.Errorf("expected card_token tok_new, got %q", result["card_token"])
	}
}

func TestTokenize_PersistsCardToken(t *testing.T) {
	mockVaultera := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"card": map[string]string{
				"card_token":       "tok_persist",
				"card_number_mask": "411111******1111",
				"card_type":        "visa",
				"expiration_month": "12",
				"expiration_year":  "2030",
			},
		})
	})

	cardSvc := &mockCardTokenService{}
	app, srv := setupPaymentApp(mockVaultera, handlers.Services{CardToken: cardSvc})
	defer srv.Close()

	body := `{
"card": {"card_number":"4111111111111111","expiration_month":"12","expiration_year":"2030"},
"org_id": "org_1", "hotel_id": "hotel_1", "gateway_id": "gw_1",
"token_scope": "single_use"
}`
	req := httptest.NewRequest(http.MethodPost, "/v1/payments/tokenize", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]string
	json.Unmarshal(respBody, &result)

	if result["card_token"] != "tok_persist" {
		t.Errorf("expected card_token tok_persist, got %q", result["card_token"])
	}
	if result["card_token_record_id"] != "ct_db_001" {
		t.Errorf("expected card_token_record_id ct_db_001, got %q", result["card_token_record_id"])
	}
	if cardSvc.created == nil {
		t.Fatal("expected card token to be persisted")
	}
	if cardSvc.created.VaultToken != "tok_persist" {
		t.Errorf("expected vault_token tok_persist, got %q", cardSvc.created.VaultToken)
	}
	if cardSvc.created.OrgID != "org_1" {
		t.Errorf("expected org_id org_1, got %q", cardSvc.created.OrgID)
	}
}

func TestTokenize_SkipsPersistWhenNoContext(t *testing.T) {
	mockVaultera := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"card": map[string]string{"card_token": "tok_no_persist"},
		})
	})

	cardSvc := &mockCardTokenService{}
	app, srv := setupPaymentApp(mockVaultera, handlers.Services{CardToken: cardSvc})
	defer srv.Close()

	// No org_id / hotel_id / gateway_id — persistence should be skipped.
	body := `{"card":{"card_number":"4111111111111111","expiration_month":"12","expiration_year":"2030"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/payments/tokenize", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
	if cardSvc.created != nil {
		t.Error("expected card token NOT to be persisted when context is missing")
	}
}

func TestTokenize_BadBody(t *testing.T) {
	app, srv := setupPaymentApp(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), handlers.Services{})
	defer srv.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/payments/tokenize", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Charge
// ---------------------------------------------------------------------------

func TestCharge_MissingFields(t *testing.T) {
	app, srv := setupPaymentApp(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), handlers.Services{})
	defer srv.Close()

	body := `{"card_token":""}` // missing url
	req := httptest.NewRequest(http.MethodPost, "/v1/payments/charge", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCharge_PersistsTransaction(t *testing.T) {
	mockVaultera := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"status_code": 200,
			"body":        json.RawMessage(`{"result":"ok"}`),
		})
	})

	txSvc := &mockTransactionService{}
	app, srv := setupPaymentApp(mockVaultera, handlers.Services{Transaction: txSvc})
	defer srv.Close()

	body := `{
"card_token": "tok_abc",
"method": "POST",
"url": "https://gateway.example.com/charge",
"body": "amount=100",
"org_id": "org_1", "hotel_id": "hotel_1", "gateway_id": "gw_1",
"amount": 1000, "currency": "USD"
}`
	req := httptest.NewRequest(http.MethodPost, "/v1/payments/charge", bytes.NewBufferString(body))
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

	if result["transaction_id"] != "tx_db_001" {
		t.Errorf("expected transaction_id tx_db_001, got %v", result["transaction_id"])
	}
	if txSvc.created == nil {
		t.Fatal("expected transaction to be created")
	}
	if txSvc.created.OrgID != "org_1" {
		t.Errorf("expected org_id org_1, got %q", txSvc.created.OrgID)
	}
	if txSvc.updated == nil {
		t.Fatal("expected transaction status to be updated")
	}
	if txSvc.updated.Status != domain.TxStatusSucceeded {
		t.Errorf("expected status succeeded, got %q", txSvc.updated.Status)
	}
}

func TestCharge_TransactionMarkedFailedOnProviderError(t *testing.T) {
	mockVaultera := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error":"gateway down"}`))
	})

	txSvc := &mockTransactionService{}
	app, srv := setupPaymentApp(mockVaultera, handlers.Services{Transaction: txSvc})
	defer srv.Close()

	body := `{
"card_token": "tok_abc",
"method": "POST",
"url": "https://gateway.example.com/charge",
"body": "",
"org_id": "org_1", "hotel_id": "hotel_1", "gateway_id": "gw_1"
}`
	req := httptest.NewRequest(http.MethodPost, "/v1/payments/charge", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", resp.StatusCode)
	}
	if txSvc.updated == nil {
		t.Fatal("expected transaction status to be updated on provider error")
	}
	if txSvc.updated.Status != domain.TxStatusFailed {
		t.Errorf("expected status failed, got %q", txSvc.updated.Status)
	}
}

// ---------------------------------------------------------------------------
// Cards
// ---------------------------------------------------------------------------

func TestDeleteCard(t *testing.T) {
	called := false
	mockVaultera := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	app, srv := setupPaymentApp(mockVaultera, handlers.Services{})
	defer srv.Close()

	req := httptest.NewRequest(http.MethodDelete, "/v1/payments/cards/tok_abc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
	if !called {
		t.Error("expected mock Vaultera server to be called")
	}
}
