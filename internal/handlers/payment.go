package handlers

import (
	"github.com/CentraGlobal/backend-payment-go/internal/vaultera"
	"github.com/gofiber/fiber/v2"
)

// PaymentHandler wraps the Vaultera client and exposes payment-related routes.
type PaymentHandler struct {
	vaultera *vaultera.Client
}

// NewPaymentHandler creates a new PaymentHandler.
func NewPaymentHandler(v *vaultera.Client) *PaymentHandler {
	return &PaymentHandler{vaultera: v}
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
type tokenizeRequest struct {
	Card vaultera.Card `json:"card"`
}

// Tokenize stores a card in Vaultera and returns its token.
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
	return c.Status(fiber.StatusCreated).JSON(card)
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
type chargeRequest struct {
	CardToken string            `json:"card_token"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body"`
}

// Charge detokenizes a stored card and forwards the request to the downstream
// payment gateway via Vaultera.
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

	sendReq := vaultera.SendRequest{
		Method:  req.Method,
		URL:     req.URL,
		Headers: req.Headers,
		Body:    req.Body,
	}

	resp, err := h.vaultera.SendCard(c.Context(), req.CardToken, sendReq)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(resp)
}
