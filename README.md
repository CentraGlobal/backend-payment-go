# backend-payment-go

Internal Payment Service for Centra.

## Overview
This service acts as a standalone, security-focused gateway between Centra's various platforms (like the OBE) and the Vaultera PCI proxy. It is designed to be payment-gateway agnostic, supporting the "Bring Your Own Keys" (BYOK) model for hotels.

## Tech Stack
- **Language:** Go
- **Framework:** Fiber v2
- **Vaulting:** Vaultera PCI Proxy (`pci.vaultera.co`)
- **Database:** PostgreSQL via `pgx/v5` (primary DB + ARI DB)
- **Cache / Queue:** Redis via `go-redis/v9`
- **Config:** `envconfig` (environment variables with prefixes)

## Environment Variables

| Variable | Prefix | Description | Default |
|---|---|---|---|
| `APP_PORT` | `APP` | HTTP listen port | `3000` |
| `APP_ENV` | `APP` | Runtime environment | `development` |
| `DATABASE_HOST` | `DATABASE` | Primary DB host | `localhost` |
| `DATABASE_PORT` | `DATABASE` | Primary DB port | `5432` |
| `DATABASE_NAME` | `DATABASE` | Primary DB name | `payment` |
| `DATABASE_USER` | `DATABASE` | Primary DB user | `postgres` |
| `DATABASE_PASSWORD` | `DATABASE` | Primary DB password | _(empty)_ |
| `DATABASE_SSLMODE` | `DATABASE` | Primary DB SSL mode | `disable` |
| `ARI_DB_HOST` | `ARI_DB` | ARI DB host | `localhost` |
| `ARI_DB_PORT` | `ARI_DB` | ARI DB port | `5432` |
| `ARI_DB_NAME` | `ARI_DB` | ARI DB name | `ari` |
| `ARI_DB_USER` | `ARI_DB` | ARI DB user | `postgres` |
| `ARI_DB_PASSWORD` | `ARI_DB` | ARI DB password | _(empty)_ |
| `ARI_DB_SSLMODE` | `ARI_DB` | ARI DB SSL mode | `disable` |
| `REDIS_HOST` | `REDIS` | Redis host | `localhost` |
| `REDIS_PORT` | `REDIS` | Redis port | `6379` |
| `REDIS_PASSWORD` | `REDIS` | Redis password | _(empty)_ |
| `REDIS_DB` | `REDIS` | Redis logical DB | `0` |
| `VAULTERA_API_KEY` | `VAULTERA` | Vaultera API key | _(required)_ |
| `VAULTERA_BASE_URL` | `VAULTERA` | Vaultera API base URL | `https://pci.vaultera.co/api/v1` |

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check (includes DB / Redis status) |
| `GET` | `/v1/session` | Create a Vaultera session token for an iframe |
| `POST` | `/v1/payments/tokenize` | Tokenize a credit card |
| `GET` | `/v1/payments/cards/:token` | Get masked card info |
| `DELETE` | `/v1/payments/cards/:token` | Delete a stored card token |
| `POST` | `/v1/payments/charge` | Detokenize and forward a charge to a gateway |

### Example: Tokenize a card
```bash
curl -X POST http://localhost:3000/v1/payments/tokenize \
  -H 'Content-Type: application/json' \
  -d '{
    "card": {
      "card_number": "4111111111111111",
      "card_type": "visa",
      "cardholder_name": "JOHN DOE",
      "service_code": "123",
      "expiration_month": "12",
      "expiration_year": "2030"
    }
  }'
```

### Example: Get a session token for the iframe
```bash
curl "http://localhost:3000/v1/session?scope=card"
```

### Example: Charge via a gateway
```bash
curl -X POST http://localhost:3000/v1/payments/charge \
  -H 'Content-Type: application/json' \
  -d '{
    "card_token": "tok_abc123",
    "method": "POST",
    "url": "https://api.stripe.com/v1/charges",
    "headers": {"Authorization": "Bearer sk_live_..."},
    "body": "{\"amount\":1000,\"currency\":\"usd\",\"source\":\"%CARD_NUMBER%\"}"
  }'
```

## Getting Started

### Local Development
```bash
export VAULTERA_API_KEY=your_key_here
go run main.go
```

### Docker
```bash
docker compose up --build
```

### Tests
```bash
go test ./...
```

