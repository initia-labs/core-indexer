package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/apperror"
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
// @Description Retrieve a list of NFT collections with optional search and pagination
// @Tags NFT
// @Accept json
// @Produce json
// @Param search query string false "Search term for filtering collections"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Success 200 {object} dto.NFTCollectionsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /nft/v1/collections [get]
func (h *NFTHandler) GetCollections(c *fiber.Ctx) error {
	// Parse pagination parameters manually
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	// Get search parameter
	search := c.Query("search")

	// Get collections from service
	response, err := h.service.GetCollections(*pagination, search)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}
