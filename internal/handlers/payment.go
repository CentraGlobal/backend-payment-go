package handlers

import (
	"strings"

	"github.com/CentraGlobal/backend-payment-go/internal/processor"
	"github.com/gofiber/fiber/v2"
)

type PaymentHandler struct {
	processor processor.Processor
}

func NewPaymentHandler(p processor.Processor) *PaymentHandler {
	return &PaymentHandler{processor: p}
}

func (h *PaymentHandler) GetSession(c *fiber.Ctx) error {
	scope := c.Query("scope", "card")
	token, err := h.processor.CreateSessionToken(c.Context(), scope)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(token)
}

type tokenizeRequest struct {
	Card processor.Card `json:"card"`
}

func (h *PaymentHandler) Tokenize(c *fiber.Ctx) error {
	var req tokenizeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	card, err := h.processor.CreateCard(c.Context(), req.Card)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusCreated).JSON(card)
}

func (h *PaymentHandler) GetCard(c *fiber.Ctx) error {
	token := c.Params("token")
	card, err := h.processor.GetCard(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(card)
}

func (h *PaymentHandler) DeleteCard(c *fiber.Ctx) error {
	token := c.Params("token")
	if err := h.processor.DeleteCard(c.Context(), token); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

type chargeRequest struct {
	CardToken string `json:"card_token"`

	// Relay mode fields
	Method  string            `json:"method,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`

	// UPG mode fields
	CredentialsID string  `json:"credentials_id,omitempty"`
	GatewayName   string  `json:"gateway_name,omitempty"`
	Amount        float64 `json:"amount,omitempty"`
	Currency      string  `json:"currency,omitempty"`
}

func (h *PaymentHandler) Charge(c *fiber.Ctx) error {
	var req chargeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.CardToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "card_token is required",
		})
	}

	// Auto-detect mode from request fields
	if req.CredentialsID != "" {
		return h.chargeViaUPG(c, req)
	} else if req.URL != "" {
		return h.chargeViaRelay(c, req)
	}
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "either credentials_id (UPG mode) or url (relay mode) is required",
	})
}

func (h *PaymentHandler) chargeViaUPG(c *fiber.Ctx, req chargeRequest) error {
	if req.GatewayName == "" || req.Currency == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "credentials_id, gateway_name, and currency are required for UPG mode",
		})
	}

	if req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "amount must be greater than zero",
		})
	}

	resp, err := h.processor.ChargeUPG(c.Context(), processor.UPGChargeRequest{
		CardToken:     req.CardToken,
		Amount:        req.Amount,
		Currency:      req.Currency,
		GatewayName:   req.GatewayName,
		CredentialsID: req.CredentialsID,
	})
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "UPG is not supported") {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":   "PROCESSOR_CONFIGURATION_MISMATCH",
				"message": "This hotel's payment gateway requires UPG support, but the payment service is not configured for UPG. Please contact support to resolve this configuration issue.",
			})
		}
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":         resp.Status,
		"transaction_id": resp.TransactionID,
		"message":        resp.Message,
		"raw_response":   resp.Raw,
	})
}

func (h *PaymentHandler) chargeViaRelay(c *fiber.Ctx, req chargeRequest) error {
	sendReq := processor.SendRequest{
		Method:  req.Method,
		URL:     req.URL,
		Headers: req.Headers,
		Body:    req.Body,
	}

	resp, err := h.processor.SendCard(c.Context(), req.CardToken, sendReq)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(resp)
}
