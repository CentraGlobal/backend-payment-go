package handlers

import (
	"github.com/gofiber/fiber/v2"
)

// GetGateways handles GET /v1/gateways.
// It returns a list of payment gateways supported by the configured UPG provider.
func (h *PaymentHandler) GetGateways(c *fiber.Ctx) error {
	gateways, err := h.processor.GetPaymentGateways(c.Context())
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(gateways)
}

// GetGatewayStructure handles GET /v1/gateways/:name/structure.
// It returns the required credential fields for the named payment gateway.
func (h *PaymentHandler) GetGatewayStructure(c *fiber.Ctx) error {
	name := c.Params("name")
	structure, err := h.processor.GetCredentialsStructure(c.Context(), name)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(structure)
}
