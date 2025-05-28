package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/services"
)

// NFTHandler handles NFT-related HTTP requests
type NFTHandler struct {
	service services.NFTService
}

// NewNFTHandler creates a new instance of NFTHandler
func NewNFTHandler(service services.NFTService) *NFTHandler {
	return &NFTHandler{
		service: service,
	}
}

// GetCollections godoc
// @Summary Get NFT collections
// @Tags NFT
// @Success 200 {object} dto.NFTCollectionsResponse
// @Router /collections [get]
func (h *NFTHandler) GetCollections(c *fiber.Ctx) error {
	// Parse pagination parameters manually
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "Invalid pagination parameters",
			Code:  fiber.StatusBadRequest,
		})
	}

	// Get search parameter
	search := c.Query("search")

	// Get collections from service
	response, err := h.service.GetCollections(*pagination, search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error: "Failed to get collections",
			Code:  fiber.StatusInternalServerError,
		})
	}

	return c.JSON(response)
}
