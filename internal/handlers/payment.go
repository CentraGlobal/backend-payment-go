package handlers

import (
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
	CardToken string            `json:"card_token"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body"`
}

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
