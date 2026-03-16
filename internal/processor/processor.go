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

type Processor interface {
	CreateCard(ctx context.Context, card Card) (*CardResponse, error)
	GetCard(ctx context.Context, cardToken string) (*CardResponse, error)
	DeleteCard(ctx context.Context, cardToken string) error
	SendCard(ctx context.Context, cardToken string, req SendRequest) (*SendResponse, error)
	CreateSessionToken(ctx context.Context, scope string) (*SessionTokenResponse, error)
	CaptureFormURL(sessionToken string) string
	Name() string
}
