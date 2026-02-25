# Project Discussion: backend-payment-go
**Date:** 2026-02-25
**Participants:** User & Centra Agent 1

## 1. Architectural Vision
The goal is to introduce a dedicated, security-focused payment microservice (**backend-payment-go**) that is decoupled from the main booking API. This service serves as the universal adapter for Centra's Online Booking Engine (OBE).

## 2. Core Technology Stack
- **Language:** Go (Golang) - chosen for strict typing, concurrency performance, and security.
- **Framework:** Fiber v2 (Express-like, high performance).
- **Vaulting:** Vaultera PCI Proxy.
- **Infrastructure:** Docker (multi-stage builds) for minimal footprint.

## 3. Integration Strategy: Vaultera PCI Proxy
To maintain zero PCI scope for Centra's servers, the service uses Vaultera's detokenization proxy:
- **Inbound:** OBE Frontend captures cards via Vaultera IFrame $\rightarrow$ returns a `card_token`.
- **Outbound:** `backend-payment-go` sends a "Payload Template" to Vaultera with placeholders (e.g., `%CARD_NUMBER%`).
- **Proxying:** Vaultera replaces placeholders with raw data and forwards the request to the target gateway (Stripe, Payzone, etc.).
- **Agility:** This allows Centra to switch gateways or support multiple gateways simultaneously without migrating sensitive data.

## 4. Business Model: Bring Your Own Keys (BYOK)
- Initial rollout will focus on hotels providing their own payment gateway API keys (Stripe, Payzone, etc.).
- **Flow:** Payment Service retrieves the hotel's specific keys from the database (encrypted) and uses them for the specific transaction.
- **Benefits:** Reduces Centra's financial risk, simplifies onboarding, and allows hotels to use their preferred bank rates.

## 5. Proposed API Flow
1. **Frontend Init:** OBE hits `GET /v1/session` to get Vaultera IFrame credentials.
2. **Payment Execution:** OBE hits `POST /v1/charge` with `card_token` and `amount`.
3. **Verification:** Main API (`centra-backend-api-nodejs`) verifies the transaction status with the Payment Service before confirming the booking.

## 6. Next Steps
- Implement `/v1/session` and `/v1/charge` endpoints.
- Establish encrypted storage for Hotel API keys.
- Develop a "Test Connection" feature for hotel onboarding.
