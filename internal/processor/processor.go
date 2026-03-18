package processor

import (
	"context"
	"encoding/json"
)

type Card struct {
	CardNumber      string `json:"card_number"`
	CardType        string `json:"card_type,omitempty"`
	CardholderName  string `json:"cardholder_name,omitempty"`
	ServiceCode     string `json:"service_code,omitempty"`
	ExpirationMonth string `json:"expiration_month"`
	ExpirationYear  string `json:"expiration_year"`
}

type CardResponse struct {
	CardToken       string `json:"card_token"`
	CardMask        string `json:"card_number_mask"`
	CardType        string `json:"card_type"`
	CardholderName  string `json:"cardholder_name"`
	ExpirationMonth string `json:"expiration_month"`
	ExpirationYear  string `json:"expiration_year"`
}

type SendRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

type SendResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       json.RawMessage   `json:"body"`
}

type SessionTokenResponse struct {
	Token string `json:"token"`
	Scope string `json:"scope"`
}

// GatewayInfo describes a payment gateway supported by the UPG provider.
type GatewayInfo struct {
	Name             string   `json:"name"`
	CredentialFields []string `json:"credential_fields"`
}

// UPGChargeRequest holds the parameters for a UPG charge operation.
type UPGChargeRequest struct {
	CardToken     string  `json:"card_token"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	GatewayName   string  `json:"gateway_name"`
	CredentialsID string  `json:"credentials_id"`
}

// UPGChargeResponse holds the result of a UPG charge operation.
// Status values: Accepted, Success, Rejected, TemporaryFailure, FatalFailure.
type UPGChargeResponse struct {
	Status        string          `json:"status"`
	TransactionID string          `json:"transaction_id"`
	Message       string          `json:"message"`
	Raw           json.RawMessage `json:"raw,omitempty"`
}

type Processor interface {
	CreateCard(ctx context.Context, card Card) (*CardResponse, error)
	GetCard(ctx context.Context, cardToken string) (*CardResponse, error)
	DeleteCard(ctx context.Context, cardToken string) error
	SendCard(ctx context.Context, cardToken string, req SendRequest) (*SendResponse, error)
	CreateSessionToken(ctx context.Context, scope string) (*SessionTokenResponse, error)
	CaptureFormURL(sessionToken string) string
	Name() string

	// UPG (Universal Payment Gateway) methods.
	// These are only fully implemented by the pci_booking provider.
	// Other providers return an error indicating UPG is unsupported.
	GetPaymentGateways(ctx context.Context) ([]GatewayInfo, error)
	GetCredentialsStructure(ctx context.Context, gatewayName string) (map[string]any, error)
	ChargeUPG(ctx context.Context, req UPGChargeRequest) (*UPGChargeResponse, error)
}
