package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// GetGateways handles GET /v1/upg/gateways.
// UPG-only: returns the list of payment gateways available through the UPG-capable
// processor (pci_booking_upg). Returns 503 UPG_NOT_AVAILABLE when the service is
// configured with a non-UPG processor (e.g. vaultera or pcibooking).
func (h *PaymentHandler) GetGateways(c *fiber.Ctx) error {
	gateways, err := h.processor.GetPaymentGateways(c.Context())
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "UPG is not supported") {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":   "UPG_NOT_AVAILABLE",
				"message": "This endpoint is only available when the payment service is configured with the pci_booking_upg processor. Please contact support.",
			})
		}
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(gateways)
}

// GetGatewayStructure handles GET /v1/upg/gateways/:name/structure.
// UPG-only: returns the required credential fields for the named payment gateway via UPG.
// Returns 503 UPG_NOT_AVAILABLE when the service is configured with a non-UPG processor
// (e.g. vaultera or pcibooking).
func (h *PaymentHandler) GetGatewayStructure(c *fiber.Ctx) error {
	name := c.Params("name")
	structure, err := h.processor.GetCredentialsStructure(c.Context(), name)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "UPG is not supported") {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":   "UPG_NOT_AVAILABLE",
				"message": "This endpoint is only available when the payment service is configured with the pci_booking_upg processor. Please contact support.",
			})
		}
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(structure)
}
