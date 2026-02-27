package vaultera_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CentraGlobal/backend-payment-go/internal/vaultera"
)

// newTestClient creates a Vaultera client pointed at the given test server URL.
func newTestClient(serverURL string) *vaultera.Client {
	return vaultera.NewClient("test-api-key", serverURL)
}

func TestCreateCard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/cards" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("api_key") != "test-api-key" {
			t.Errorf("missing or wrong api_key")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"card": map[string]string{
				"card_token":       "tok_abc123",
				"card_number_mask": "411111******1111",
				"card_type":        "visa",
				"cardholder_name":  "JOHN DOE",
				"expiration_month": "12",
				"expiration_year":  "2030",
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	card, err := client.CreateCard(context.Background(), vaultera.Card{
		CardNumber:      "4111111111111111",
		CardType:        "visa",
		CardholderName:  "JOHN DOE",
		ExpirationMonth: "12",
		ExpirationYear:  "2030",
	})
	if err != nil {
		t.Fatalf("CreateCard error: %v", err)
	}
	if card.CardToken != "tok_abc123" {
		t.Errorf("expected card_token tok_abc123, got %q", card.CardToken)
	}
	if card.CardNumberMask != "411111******1111" {
		t.Errorf("unexpected card_number_mask %q", card.CardNumberMask)
	}
}

func TestGetCard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/cards/tok_abc123" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"card": map[string]string{
				"card_token":       "tok_abc123",
				"card_number_mask": "411111******1111",
				"card_type":        "visa",
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	card, err := client.GetCard(context.Background(), "tok_abc123")
	if err != nil {
		t.Fatalf("GetCard error: %v", err)
	}
	if card.CardToken != "tok_abc123" {
		t.Errorf("expected card_token tok_abc123, got %q", card.CardToken)
	}
}

func TestDeleteCard(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/cards/tok_abc123" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	if err := client.DeleteCard(context.Background(), "tok_abc123"); err != nil {
		t.Fatalf("DeleteCard error: %v", err)
	}
	if !called {
		t.Error("expected server to be called")
	}
}

func TestSendCard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/cards/tok_abc123/send" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(vaultera.SendResponse{StatusCode: 200})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	resp, err := client.SendCard(context.Background(), "tok_abc123", vaultera.SendRequest{
		Method: "POST",
		URL:    "https://api.stripe.com/v1/charges",
		Body:   `{"amount":1000,"currency":"usd","source":"%CARD_NUMBER%"}`,
	})
	if err != nil {
		t.Fatalf("SendCard error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status_code 200, got %d", resp.StatusCode)
	}
}

func TestCreateSessionToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/session_tokens" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"session_token": map[string]string{
				"token": "st_xyz",
				"scope": "card",
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	st, err := client.CreateSessionToken(context.Background(), "card")
	if err != nil {
		t.Fatalf("CreateSessionToken error: %v", err)
	}
	if st.Token != "st_xyz" {
		t.Errorf("expected token st_xyz, got %q", st.Token)
	}
	if st.Scope != "card" {
		t.Errorf("expected scope card, got %q", st.Scope)
	}
}

func TestCaptureFormURL(t *testing.T) {
	client := newTestClient("https://pci.vaultera.co/api/v1")
	u := client.CaptureFormURL("st_xyz")
	expected := "https://pci.vaultera.co/api/v1/capture_form?session_token=st_xyz"
	if u != expected {
		t.Errorf("expected %q, got %q", expected, u)
	}
}

func TestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	_, err := client.GetCard(context.Background(), "bad_token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
