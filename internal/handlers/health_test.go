package handlers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CentraGlobal/backend-payment-go/internal/handlers"
	goredis "github.com/redis/go-redis/v9"
	"github.com/gofiber/fiber/v2"
)

func TestHealthHandler_NoInfra(t *testing.T) {
	app := fiber.New()
	// Pass nil pools and nil redis to simulate no infra configured.
	app.Get("/health", handlers.HealthHandler(nil, nil, nil))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if result["status"] != "healthy" {
		t.Errorf("expected status=healthy, got %q", result["status"])
	}
}

func TestHealthHandler_Degraded_Redis(t *testing.T) {
	app := fiber.New()
	// Point Redis at a port that is not listening so Ping fails immediately.
	badRedis := goredis.NewClient(&goredis.Options{
		Addr: "localhost:1", // port 1 is never open
	})
	app.Get("/health", handlers.HealthHandler(nil, nil, badRedis))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req, 5000) // 5 s timeout for the test
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if result["status"] != "degraded" {
		t.Errorf("expected status=degraded, got %q", result["status"])
	}
	if result["redis"] != "unhealthy" {
		t.Errorf("expected redis=unhealthy, got %q", result["redis"])
	}
}

