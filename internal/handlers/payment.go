package handlers

import (
	"encoding/json"
	"log"

	"github.com/CentraGlobal/backend-payment-go/internal/services"
	"github.com/CentraGlobal/backend-payment-go/internal/types"
	"github.com/CentraGlobal/backend-payment-go/internal/vaultera"
	"github.com/gofiber/fiber/v2"
)

// Services groups the optional domain service dependencies for PaymentHandler.
// Any nil service is silently skipped — the handler degrades gracefully to
// Vaultera-only mode without persistence.
type Services struct {
	Gateway     services.PaymentGatewayService
	CardToken   services.CardTokenService
	Transaction services.TransactionService
}

// PaymentHandler wraps the Vaultera client and domain services to expose
// payment-related HTTP routes.
type PaymentHandler struct {
	vaultera   *vaultera.Client
	gatewaySvc services.PaymentGatewayService
	cardSvc    services.CardTokenService
	txSvc      services.TransactionService
}

// NewPaymentHandler creates a new PaymentHandler.
// svcs is optional; pass an empty Services{} when no DB-backed services are
// needed (e.g. in tests).
func NewPaymentHandler(v *vaultera.Client, svcs Services) *PaymentHandler {
	return &PaymentHandler{
		vaultera:   v,
		gatewaySvc: svcs.Gateway,
		cardSvc:    svcs.CardToken,
		txSvc:      svcs.Transaction,
	}
}

// ---------------------------------------------------------------------------
// Session
// ---------------------------------------------------------------------------

// GetSession returns a Vaultera session token for use in the card-capture iframe.
// GET /v1/session
func (h *PaymentHandler) GetSession(c *fiber.Ctx) error {
	scope := c.Query("scope", "card")
	token, err := h.vaultera.CreateSessionToken(c.Context(), scope)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(token)
}

// ---------------------------------------------------------------------------
// Cards / Tokenization
// ---------------------------------------------------------------------------

// tokenizeRequest is the request body for the tokenize endpoint.
// The card field is forwarded to Vaultera; the remaining fields are used to
// persist a card token record in the shared schema when a CardTokenService is
// wired (org_id, hotel_id, and gateway_id are required for persistence).
type tokenizeRequest struct {
	Card      vaultera.Card `json:"card"`
	OrgID     string        `json:"org_id,omitempty"`
	HotelID   string        `json:"hotel_id,omitempty"`
	GatewayID string        `json:"gateway_id,omitempty"`
	// Optional association fields
	ReservationID string `json:"reservation_id,omitempty"`
	PersonID      string `json:"person_id,omitempty"`
	CustomerID    string `json:"customer_id,omitempty"`
	// Token behaviour
	TokenScope string `json:"token_scope,omitempty"`
}

// tokenizeResponse extends the Vaultera card response with the optional DB record id.
type tokenizeResponse struct {
	*vaultera.CardResponse
	CardTokenRecordID string `json:"card_token_record_id,omitempty"`
}

// Tokenize stores a card in Vaultera and, when persistence context is provided,
// writes a card token record to the shared payment_card_tokens table.
// POST /v1/payments/tokenize
func (h *PaymentHandler) Tokenize(c *fiber.Ctx) error {
	var req tokenizeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	card, err := h.vaultera.CreateCard(c.Context(), req.Card)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	resp := &tokenizeResponse{CardResponse: card}

	// Persist the card token record when all required context is available.
	if h.cardSvc != nil && req.OrgID != "" && req.HotelID != "" && req.GatewayID != "" {
		scope := types.CardTokenScope(req.TokenScope)
		if scope == "" {
			scope = types.TokenScopeSingleUse
		}

		token := &types.CardToken{
			OrgID:         req.OrgID,
			HotelID:       req.HotelID,
			GatewayID:     req.GatewayID,
			VaultProvider: "vaultera",
			VaultToken:    card.CardToken,
			TokenScope:    scope,
			Status:        types.TokenStatusActive,
		}
		if req.ReservationID != "" {
			token.ReservationID = &req.ReservationID
		}
		if req.PersonID != "" {
			token.PersonID = &req.PersonID
		}
		if req.CustomerID != "" {
			token.CustomerID = &req.CustomerID
		}
		if card.CardType != "" {
			token.Brand = &card.CardType
		}
		// Extract last4 from the mask (e.g. "411111******1111" → "1111").
		if mask := card.CardNumberMask; len(mask) >= 4 {
			last4 := mask[len(mask)-4:]
			token.Last4 = &last4
		}
		if card.ExpirationMonth != "" {
			token.ExpMonth = &card.ExpirationMonth
		}
		if card.ExpirationYear != "" {
			token.ExpYear = &card.ExpirationYear
		}

		saved, saveErr := h.cardSvc.Create(c.Context(), token)
		if saveErr != nil {
			// Log but do not fail the request — Vaultera tokenisation succeeded.
			log.Printf("tokenize: persist card token: %v", saveErr)
		} else {
			resp.CardTokenRecordID = saved.ID
		}
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// GetCard returns masked card info for a given token.
// GET /v1/payments/cards/:token
func (h *PaymentHandler) GetCard(c *fiber.Ctx) error {
	token := c.Params("token")
	card, err := h.vaultera.GetCard(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(card)
}

// DeleteCard removes a stored card token from Vaultera.
// DELETE /v1/payments/cards/:token
func (h *PaymentHandler) DeleteCard(c *fiber.Ctx) error {
	token := c.Params("token")
	if err := h.vaultera.DeleteCard(c.Context(), token); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Charge / Authorize
// ---------------------------------------------------------------------------

// chargeRequest is the request body for the charge endpoint.
// The card_token, method, url, headers, and body fields drive the Vaultera
// detokenise-and-forward call.  The remaining fields are used to persist a
// transaction record in the shared payment_transactions table when a
// TransactionService is wired (org_id, hotel_id, and gateway_id are required).
type chargeRequest struct {
	CardToken string            `json:"card_token"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body"`
	// Persistence context (optional)
	OrgID          string `json:"org_id,omitempty"`
	HotelID        string `json:"hotel_id,omitempty"`
	GatewayID      string `json:"gateway_id,omitempty"`
	CardTokenID    string `json:"card_token_id,omitempty"`
	ReservationID  string `json:"reservation_id,omitempty"`
	PersonID       string `json:"person_id,omitempty"`
	CustomerID     string `json:"customer_id,omitempty"`
	Amount         int64  `json:"amount,omitempty"`
	Currency       string `json:"currency,omitempty"`
	Operation      string `json:"operation,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// chargeResponse extends the Vaultera send response with an optional DB record id.
type chargeResponse struct {
	*vaultera.SendResponse
	TransactionID string `json:"transaction_id,omitempty"`
}

// Charge detokenizes a stored card and forwards the request to the downstream
// payment gateway via Vaultera.  When persistence context is provided, a
// transaction record is created before the provider call and updated with the
// final status afterwards.
// POST /v1/payments/charge
func (h *PaymentHandler) Charge(c *fiber.Ctx) error {
	var req chargeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.CardToken == "" || req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "card_token and url are required",
		})
	}

	// Create a pending transaction record before the provider call.
	var txID string
	if h.txSvc != nil && req.OrgID != "" && req.HotelID != "" && req.GatewayID != "" {
		op := types.TransactionOperation(req.Operation)
		if op == "" {
			op = types.TxOpCharge
		}

		pendingTx := &types.Transaction{
			OrgID:     req.OrgID,
			HotelID:   req.HotelID,
			GatewayID: req.GatewayID,
			Operation: op,
			Amount:    req.Amount,
			Currency:  req.Currency,
		}
		if req.CardTokenID != "" {
			pendingTx.CardTokenID = &req.CardTokenID
		}
		if req.ReservationID != "" {
			pendingTx.ReservationID = &req.ReservationID
		}
		if req.PersonID != "" {
			pendingTx.PersonID = &req.PersonID
		}
		if req.CustomerID != "" {
			pendingTx.CustomerID = &req.CustomerID
		}
		if req.IdempotencyKey != "" {
			pendingTx.IdempotencyKey = &req.IdempotencyKey
		}

		// Capture the request payload for audit.
		if reqJSON, err := json.Marshal(req); err == nil {
			pendingTx.RequestPayload = reqJSON
		}

		saved, createErr := h.txSvc.CreatePending(c.Context(), pendingTx)
		if createErr != nil {
			log.Printf("charge: create pending transaction: %v", createErr)
		} else {
			txID = saved.ID
		}
	}

	sendReq := vaultera.SendRequest{
		Method:  req.Method,
		URL:     req.URL,
		Headers: req.Headers,
		Body:    req.Body,
	}

	providerResp, providerErr := h.vaultera.SendCard(c.Context(), req.CardToken, sendReq)

	// Update the transaction status based on the provider outcome.
	if h.txSvc != nil && txID != "" {
		update := buildTransactionUpdate(providerResp, providerErr)
		if updateErr := h.txSvc.UpdateStatus(c.Context(), txID, update); updateErr != nil {
			log.Printf("charge: update transaction status: %v", updateErr)
		}
	}

	if providerErr != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": providerErr.Error(),
		})
	}

	resp := &chargeResponse{
		SendResponse:  providerResp,
		TransactionID: txID,
	}
	return c.JSON(resp)
}

// buildTransactionUpdate constructs the status update from the provider response.
func buildTransactionUpdate(resp *vaultera.SendResponse, providerErr error) types.TransactionStatusUpdate {
	if providerErr != nil {
		msg := providerErr.Error()
		return types.TransactionStatusUpdate{
			Status:         types.TxStatusFailed,
			FailureMessage: &msg,
		}
	}

	update := types.TransactionStatusUpdate{
		Status: types.TxStatusSucceeded,
	}
	if resp != nil {
		if b, err := json.Marshal(resp); err == nil {
			update.ResponsePayload = b
		}
	}
	return update
}
