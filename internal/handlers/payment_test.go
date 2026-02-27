package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CentraGlobal/backend-payment-go/internal/handlers"
	"github.com/CentraGlobal/backend-payment-go/internal/vaultera"
	"github.com/gofiber/fiber/v2"
)

// setupPaymentApp builds a Fiber app with a mock Vaultera server injected.
func setupPaymentApp(vaulteraHandler http.Handler) (*fiber.App, *httptest.Server) {
	vSrv := httptest.NewServer(vaulteraHandler)
	client := vaultera.NewClient("test-key", vSrv.URL)
	ph := handlers.NewPaymentHandler(client)

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

func TestGetSession(t *testing.T) {
	mockVaultera := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/session_tokens" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"session_token": map[string]string{"token": "st_test", "scope": "card"},
		})
	})

	app, srv := setupPaymentApp(mockVaultera)
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

	app, srv := setupPaymentApp(mockVaultera)
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

func TestTokenize_BadBody(t *testing.T) {
	app, srv := setupPaymentApp(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/payments/tokenize", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCharge_MissingFields(t *testing.T) {
	app, srv := setupPaymentApp(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
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

func TestDeleteCard(t *testing.T) {
	called := false
	mockVaultera := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	app, srv := setupPaymentApp(mockVaultera)
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
