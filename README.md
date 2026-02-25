# backend-payment-go

Internal Payment Service for Centra.

## Overview
This service acts as a standalone, security-focused gateway between Centra's various platforms (like the OBE) and the Vaultera PCI proxy. It is designed to be payment-gateway agnostic, supporting the "Bring Your Own Keys" (BYOK) model for hotels.

## Tech Stack
- **Language:** Go
- **Framework:** Fiber
- **Vaulting:** Vaultera PCI Proxy

## Getting Started
### Local Development
```bash
go run main.go
```

### Docker
```bash
docker compose up --build
```
