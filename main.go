package main

import (
	"context"
	"log"
	"strings"

	"github.com/CentraGlobal/backend-payment-go/internal/config"
	"github.com/CentraGlobal/backend-payment-go/internal/db"
	"github.com/CentraGlobal/backend-payment-go/internal/handlers"
	"github.com/CentraGlobal/backend-payment-go/internal/infisical"
	"github.com/CentraGlobal/backend-payment-go/internal/middleware"
	"github.com/CentraGlobal/backend-payment-go/internal/pcibooking"
	"github.com/CentraGlobal/backend-payment-go/internal/processor"
	redisclient "github.com/CentraGlobal/backend-payment-go/internal/redis"
	"github.com/CentraGlobal/backend-payment-go/internal/vaultera"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	if err := infisical.LoadSecrets(ctx); err != nil {
		log.Fatalf("failed to load secrets from Infisical: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Validate auth configuration.
	if cfg.Auth.Require && cfg.Auth.SharedSecret == "" {
		log.Fatalf("AUTH_SHARED_SECRET is required when AUTH_REQUIRE=true")
	}
	if cfg.App.Env != "development" && !cfg.Auth.Require {
		log.Printf("warning: AUTH_REQUIRE=false in non-development environment")
	}

	// Database pools (best-effort; service starts without them if unavailable)
	var dbPool *pgxpool.Pool
	var ariPool *pgxpool.Pool

	dbPool, err = db.NewPool(ctx, cfg.Database)
	if err != nil {
		log.Printf("warning: failed to connect to database: %v", err)
	}

	ariPool, err = db.NewARIPool(ctx, cfg.ARIDB)
	if err != nil {
		log.Printf("warning: failed to connect to ARI database: %v", err)
	}

	// Redis client (best-effort)
	var rdb *goredis.Client
	rdb = redisclient.NewClient(cfg.Redis)
	if pingErr := rdb.Ping(ctx).Err(); pingErr != nil {
		log.Printf("warning: failed to connect to Redis: %v", pingErr)
	}

	// Processor selection
	var proc processor.Processor
	procName := strings.TrimSpace(strings.ToLower(cfg.Processor.Name))
	switch procName {
	case "pcibooking":
		if cfg.PCIBooking.APIKey == "" {
			log.Fatalf("PCIBOOKING_API_KEY (or cfg.PCIBooking.APIKey) must be set when PROCESSOR_NAME=pcibooking")
		}
		if cfg.PCIBooking.BaseURL == "" {
			log.Fatalf("PCIBOOKING_BASE_URL (or cfg.PCIBooking.BaseURL) must be set when PROCESSOR_NAME=pcibooking")
		}
		proc = pcibooking.NewClient(cfg.PCIBooking.APIKey, cfg.PCIBooking.BaseURL)
	case "vaultera":
		if cfg.Vaultera.APIKey == "" {
			log.Fatalf("VAULTERA_API_KEY (or cfg.Vaultera.APIKey) must be set when PROCESSOR_NAME=vaultera")
		}
		if cfg.Vaultera.BaseURL == "" {
			log.Fatalf("VAULTERA_BASE_URL (or cfg.Vaultera.BaseURL) must be set when PROCESSOR_NAME=vaultera")
		}
		proc = vaultera.NewClient(cfg.Vaultera.APIKey, cfg.Vaultera.BaseURL)
	default:
		log.Fatalf("unknown processor: %s (supported: vaultera, pcibooking)", cfg.Processor.Name)
	}
	log.Printf("using processor: %s", proc.Name())

	// HTTP handlers
	paymentHandler := handlers.NewPaymentHandler(proc)

	app := fiber.New()
	app.Use(logger.New())

	// Health
	app.Get("/health", handlers.HealthHandler(dbPool, ariPool, rdb))

	// All /v1 routes require shared secret auth
	v1 := app.Group("/v1", middleware.RequireSharedSecret(cfg.Auth))
	v1.Get("/session", paymentHandler.GetSession)

	// Payment routes
	payments := v1.Group("/payments")
	payments.Post("/tokenize", paymentHandler.Tokenize)
	payments.Post("/charge", paymentHandler.Charge)
	payments.Get("/cards/:token", paymentHandler.GetCard)
	payments.Delete("/cards/:token", paymentHandler.DeleteCard)

	log.Fatal(app.Listen(":" + cfg.App.Port))
}
