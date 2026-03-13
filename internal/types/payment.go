package types

import (
	"encoding/json"
	"time"
)

// ---------------------------------------------------------------------------
// Enum types
// ---------------------------------------------------------------------------

// PaymentGatewayProvider identifies the payment provider.
type PaymentGatewayProvider string

const (
	GatewayProviderPayzone PaymentGatewayProvider = "payzone"
	GatewayProviderStripe  PaymentGatewayProvider = "stripe"
	GatewayProviderCustom  PaymentGatewayProvider = "custom"
)

// PaymentGatewayMode distinguishes test from live credentials.
type PaymentGatewayMode string

const (
	GatewayModeTest PaymentGatewayMode = "test"
	GatewayModeLive PaymentGatewayMode = "live"
)

// PaymentGatewayStatus reflects the operational state of a gateway.
type PaymentGatewayStatus string

const (
	GatewayStatusActive   PaymentGatewayStatus = "active"
	GatewayStatusInactive PaymentGatewayStatus = "inactive"
)

// CardTokenScope controls single- vs. multi-use token semantics.
type CardTokenScope string

const (
	TokenScopeSingleUse CardTokenScope = "single_use"
	TokenScopeMultiUse  CardTokenScope = "multi_use"
)

// CardTokenStatus tracks the lifecycle of a stored card token.
type CardTokenStatus string

const (
	TokenStatusActive   CardTokenStatus = "active"
	TokenStatusConsumed CardTokenStatus = "consumed"
	TokenStatusRevoked  CardTokenStatus = "revoked"
	TokenStatusExpired  CardTokenStatus = "expired"
)

// TransactionOperation describes the type of payment action.
type TransactionOperation string

const (
	TxOpAuthorize TransactionOperation = "authorize"
	TxOpCapture   TransactionOperation = "capture"
	TxOpCharge    TransactionOperation = "charge"
	TxOpRefund    TransactionOperation = "refund"
	TxOpVoid      TransactionOperation = "void"
)

// TransactionStatus tracks the lifecycle of a payment transaction.
type TransactionStatus string

const (
	TxStatusPending    TransactionStatus = "pending"
	TxStatusProcessing TransactionStatus = "processing"
	TxStatusSucceeded  TransactionStatus = "succeeded"
	TxStatusFailed     TransactionStatus = "failed"
	TxStatusCancelled  TransactionStatus = "cancelled"
)

// ---------------------------------------------------------------------------
// Shared payment structs
// ---------------------------------------------------------------------------

// PaymentGateway represents a row in the payment_gateways table.
type PaymentGateway struct {
	ID            string                 `json:"id"`
	OrgID         string                 `json:"org_id"`
	HotelID       string                 `json:"hotel_id"`
	Provider      PaymentGatewayProvider `json:"provider"`
	Mode          PaymentGatewayMode     `json:"mode"`
	Status        PaymentGatewayStatus   `json:"status"`
	IsDefault     bool                   `json:"is_default"`
	PublicConfig  json.RawMessage        `json:"public_config,omitempty"`
	SecretRefs    json.RawMessage        `json:"secret_refs,omitempty"`
	RoutingConfig json.RawMessage        `json:"routing_config,omitempty"`
	WebhookConfig json.RawMessage        `json:"webhook_config,omitempty"`
	Capabilities  json.RawMessage        `json:"capabilities,omitempty"`
	Metadata      json.RawMessage        `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// CardToken represents a row in the payment_card_tokens table.
type CardToken struct {
	ID                     string          `json:"id"`
	OrgID                  string          `json:"org_id"`
	HotelID                string          `json:"hotel_id"`
	ReservationID          *string         `json:"reservation_id,omitempty"`
	PersonID               *string         `json:"person_id,omitempty"`
	CustomerID             *string         `json:"customer_id,omitempty"`
	GatewayID              string          `json:"gateway_id"`
	VaultProvider          string          `json:"vault_provider"`
	VaultToken             string          `json:"vault_token"`
	TokenScope             CardTokenScope  `json:"token_scope"`
	Status                 CardTokenStatus `json:"status"`
	Brand                  *string         `json:"brand,omitempty"`
	Last4                  *string         `json:"last4,omitempty"`
	ExpMonth               *string         `json:"exp_month,omitempty"`
	ExpYear                *string         `json:"exp_year,omitempty"`
	Fingerprint            *string         `json:"fingerprint_hash,omitempty"`
	ExpiresAt              *time.Time      `json:"expires_at,omitempty"`
	ConsumedAt             *time.Time      `json:"consumed_at,omitempty"`
	RevokedAt              *time.Time      `json:"revoked_at,omitempty"`
	Metadata               json.RawMessage `json:"metadata,omitempty"`
	CreatedBy              *string         `json:"created_by,omitempty"`
	CreatedByPrincipalType *string         `json:"created_by_principal_type,omitempty"`
	UpdatedBy              *string         `json:"updated_by,omitempty"`
	UpdatedByPrincipalType *string         `json:"updated_by_principal_type,omitempty"`
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
}

// Transaction represents a row in the payment_transactions table.
type Transaction struct {
	ID                     string               `json:"id"`
	OrgID                  string               `json:"org_id"`
	HotelID                string               `json:"hotel_id"`
	ReservationID          *string              `json:"reservation_id,omitempty"`
	PersonID               *string              `json:"person_id,omitempty"`
	CustomerID             *string              `json:"customer_id,omitempty"`
	GatewayID              string               `json:"gateway_id"`
	CardTokenID            *string              `json:"card_token_id,omitempty"`
	ScheduledTxID          *string              `json:"scheduled_transaction_id,omitempty"`
	Operation              TransactionOperation `json:"operation"`
	Status                 TransactionStatus    `json:"status"`
	Amount                 int64                `json:"amount"`
	Currency               string               `json:"currency"`
	ParentTxID             *string              `json:"parent_transaction_id,omitempty"`
	IdempotencyKey         *string              `json:"idempotency_key,omitempty"`
	ProviderTransactionID  *string              `json:"provider_transaction_id,omitempty"`
	ProviderChargeID       *string              `json:"provider_charge_id,omitempty"`
	VaulteraRequestID      *string              `json:"vaultera_request_id,omitempty"`
	RequestPayload         json.RawMessage      `json:"request_payload,omitempty"`
	ResponsePayload        json.RawMessage      `json:"response_payload,omitempty"`
	FailureCode            *string              `json:"failure_code,omitempty"`
	FailureMessage         *string              `json:"failure_message,omitempty"`
	ProcessedAt            *time.Time           `json:"processed_at,omitempty"`
	FailedAt               *time.Time           `json:"failed_at,omitempty"`
	Metadata               json.RawMessage      `json:"metadata,omitempty"`
	CreatedBy              *string              `json:"created_by,omitempty"`
	CreatedByPrincipalType *string              `json:"created_by_principal_type,omitempty"`
	CreatedAt              time.Time            `json:"created_at"`
}

// TransactionStatusUpdate carries the fields written when a transaction
// resolves (succeeds or fails) after the provider call.
type TransactionStatusUpdate struct {
	Status                TransactionStatus
	ProviderTransactionID *string
	ProviderChargeID      *string
	VaulteraRequestID     *string
	ResponsePayload       json.RawMessage
	FailureCode           *string
	FailureMessage        *string
}
