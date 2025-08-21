package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/initia-labs/core-indexer/api/apperror"
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
// @Description Retrieve the latest block height
// @Tags Block
// @Produce json
// @Success 200 {object} dto.BlockHeightLatestResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/block/v1/latest_block_height [get]
func (h *BlockHandler) GetBlockHeightLatest(c *fiber.Ctx) error {
	response, err := h.service.GetBlockHeightLatest()
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetBlockHeightInformativeLatest godoc
// @Summary Get latest informative block height
// @Description Retrieve the latest informative block height
// @Tags Block
// @Produce json
// @Success 200 {object} dto.BlockHeightInformativeLatestResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/block/v1/latest_informative_block_height [get]
func (h *BlockHandler) GetBlockHeightInformativeLatest(c *fiber.Ctx) error {
	response, err := h.service.GetBlockHeightInformativeLatest()
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetBlockTimeAverage godoc
// @Summary Get average block time
// @Description Retrieve the average time taken to mine a block
// @Tags Block
// @Produce json
// @Success 200 {object} dto.BlockTimeAverageResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/block/v1/avg_blocktime [get]
func (h *BlockHandler) GetBlockTimeAverage(c *fiber.Ctx) error {
	response, err := h.service.GetBlockTimeAverage()
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetBlocks godoc
// @Summary Get blocks
// @Description Retrieve a list of blocks with pagination
// @Tags Block
// @Accept json
// @Produce json
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Count total number of transactions" default(false)
// @Success 200 {object} dto.BlocksResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/block/v1/blocks [get]
func (h *BlockHandler) GetBlocks(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetBlocks(*pagination)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetBlockInfo godoc
// @Summary Get block info by height
// @Description Retrieve detailed information about a block by its height
// @Tags Block
// @Accept json
// @Produce json
// @Param height path int true "Block height"
// @Success 200 {object} dto.BlockInfoResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/block/v1/blocks/{height}/info [get]
func (h *BlockHandler) GetBlockInfo(c *fiber.Ctx) error {
	heightStr := c.Params("height")

	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		return apperror.HandleErrorResponse(c, apperror.NewHeightInteger())
	}

	response, err := h.service.GetBlockInfo(height)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetBlockTxs godoc
// @Summary Get transactions in a block by height
// @Description Retrieve transactions in a block by its height with pagination
// @Tags Block
// @Accept json
// @Produce json
// @Param height path int true "Block height"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Count total number of transactions" default(false)
// @Success 200 {object} dto.BlockTxsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/block/v1/blocks/{height}/txs [get]
func (h *BlockHandler) GetBlockTxs(c *fiber.Ctx) error {
	heightStr := c.Params("height")

	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		return apperror.HandleErrorResponse(c, apperror.NewHeightInteger())
	}

	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetBlockTxs(*pagination, height)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}
