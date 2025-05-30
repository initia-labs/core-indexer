package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/apperror"
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
// @Description Retrieve the latest block height
// @Tags Block
// @Produce json
// @Success 200 {object} dto.RestBlockHeightLatestResponse
// @Router /indexer/block/v1/latest_block_height [get]
func (h *BlockHandler) GetBlockHeightLatest(c *fiber.Ctx) error {
	response, err := h.service.GetBlockHeightLatest()
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}

// GetBlockTimeAverage godoc
// @Summary Get average block time
// @Description Retrieve the average time taken to mine a block
// @Tags Block
// @Produce json
// @Success 200 {object} dto.RestBlockTimeAverageResponse
// @Router /indexer/block/v1/avg_blocktime [get]
func (h *BlockHandler) GetBlockTimeAverage(c *fiber.Ctx) error {
	response, err := h.service.GetBlockTimeAverage()
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}
