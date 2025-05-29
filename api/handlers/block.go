package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/services"
)

type BlockHandler struct {
	service services.BlockService
}

func NewBlockHandler(service services.BlockService) *BlockHandler {
	return &BlockHandler{
		service: service,
	}
}

// GetBlockHeightLatest godoc
// @Summary Get latest block height
// @Tags Block
// @Success 200 {integer} int64 "Latest block height"
// @Failure 500 {object} dto.ErrorResponse
// @Router /indexer/block/v1/latest_block_height [get]
func (h *BlockHandler) GetBlockHeightLatest(c *fiber.Ctx) error {
	response, err := h.service.GetBlockHeightLatest()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error: "Failed to get block height latest",
			Code:  fiber.StatusInternalServerError,
		})
	}

	return c.JSON(response)
}

// GetBlockTimeAverage godoc
// @Summary Get average block time
// @Tags Block
// @Success 200 {number} float64 "Average block time in seconds"
// @Failure 500 {object} dto.ErrorResponse
// @Router /indexer/block/v1/avg_blocktime [get]
func (h *BlockHandler) GetBlockTimeAverage(c *fiber.Ctx) error {
	response, err := h.service.GetBlockTimeAverage()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error: "Failed to get block time average",
			Code:  fiber.StatusInternalServerError,
		})
	}

	return c.JSON(response)
}
