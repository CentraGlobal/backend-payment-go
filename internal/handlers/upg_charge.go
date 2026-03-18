package handlers

import (
	"github.com/CentraGlobal/backend-payment-go/internal/processor"
	"github.com/gofiber/fiber/v2"
)

type upgChargeRequest struct {
	CardToken     string  `json:"card_token"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	GatewayName   string  `json:"gateway_name"`
	CredentialsID string  `json:"credentials_id"`
}

type upgChargeResponse struct {
	Status        string      `json:"status"`
	TransactionID string      `json:"transaction_id"`
	Message       string      `json:"message"`
	RawResponse   interface{} `json:"raw_response,omitempty"`
}

// ChargeUPG handles POST /v1/payments/charge/upg.
// It processes a charge via the Universal Payment Gateway (UPG) using a stored
// card token and credentials previously stored in the PCI Booking vault.
func (h *PaymentHandler) ChargeUPG(c *fiber.Ctx) error {
	var req upgChargeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.CardToken == "" || req.Currency == "" || req.GatewayName == "" || req.CredentialsID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "card_token, currency, gateway_name, and credentials_id are required",
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
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(upgChargeResponse{
		Status:        resp.Status,
		TransactionID: resp.TransactionID,
		Message:       resp.Message,
		RawResponse:   resp.Raw,
	})
}
