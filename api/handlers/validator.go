package handlers

import (
	"github.com/gofiber/fiber/v2"
	"strconv"

	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/services"
)

type ValidatorHandler struct {
	service services.ValidatorService
}

func NewValidatorHandler(service services.ValidatorService) *ValidatorHandler {
	return &ValidatorHandler{
		service: service,
	}
}

// GetValidators godoc
// @Summary Get list of validators
// @Description Retrieve the list of all validators
// @Tags Validator
// @Produce json
// @Param is_active query boolean false "Query for active validators" default(true)
// @Param sort_by query string false "Sort validators by field: 'uptime', 'commission', 'moniker' or empty for default (voting power)" default()
// @Param search query string false "Search validators by moniker or exact operator address" default()
// @Success 200 {object} dto.ValidatorsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/validator/v1/validators [get]
func (h *ValidatorHandler) GetValidators(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	isActive := true
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActive = isActiveStr == "true"
	}
	sortBy := c.Query("sort_by")
	search := c.Query("search")

	response, err := h.service.GetValidators(*pagination, isActive, sortBy, search)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}

// GetValidatorInfo godoc
// @Summary Get validator info
// @Description Get validator info from the operator address
// @Tags Validator
// @Produce json
// @Param operatorAddr path string true "Validator operator address"
// @Success 200 {object} dto.ValidatorInfoResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/validator/v1/validators/{operatorAddr}/info [get]
func (h *ValidatorHandler) GetValidatorInfo(c *fiber.Ctx) error {
	addr := c.Params("operatorAddr")
	val, err := h.service.GetValidatorInfo(addr)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}
	return c.JSON(val)
}

// GetValidatorUptime godoc
// @Summary Get validator uptime
// @Description Get validator uptime from the operator address
// @Tags Validator
// @Produce json
// @Param operatorAddr path string true "Validator operator address"
// @Param blocks query integer true "Blocks to be queried" default()
// @Success 200 {object} dto.ValidatorUptimeResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/validator/v1/validators/{operatorAddr}/uptime [get]
func (h *ValidatorHandler) GetValidatorUptime(c *fiber.Ctx) error {
	addr := c.Params("operatorAddr")
	blocks := 0
	if blocksStr := c.Query("blocks"); blocksStr != "" {
		if block, err := strconv.Atoi(blocksStr); err != nil {
			errResp := apperror.NewBadRequest("blocks argument is not a valid integer")
			return c.Status(errResp.Code).JSON(errResp)
		} else {
			blocks = block
		}
	} else {
		errResp := apperror.NewBadRequest("blocks parameter is required")
		return c.Status(errResp.Code).JSON(errResp)
	}

	uptime, err := h.service.GetValidatorUptime(addr, blocks)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}
	return c.JSON(uptime)
}

// GetValidatorDelegationRelatedTxs godoc
// @Summary Get delegation transactions of a validator
// @Description Retrieves list of delegation related of a validator
// @Tags Validator
// @Produce json
// @Param operatorAddr path string true "Validator operator address"
// @Success 200 {object} dto.ValidatorDelegationRelatedTxsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/validator/v1/validators/{operatorAddr}/delegation-related-txs [get]
func (h *ValidatorHandler) GetValidatorDelegationRelatedTxs(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	addr := c.Params("operatorAddr")
	response, err := h.service.GetValidatorDelegationTxs(*pagination, addr)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}

// GetValidatorProposedBlocks godoc
// @Summary Get validator proposed blocks
// @Description Retrieves list of proposed blocks from a validator
// @Tags Validator
// @Produce json
// @Param operatorAddr path string true "Validator operator address"
// @Success 200 {object} dto.ValidatorProposedBlocksResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/validator/v1/validators/{operatorAddr}/proposed-blocks [get]
func (h *ValidatorHandler) GetValidatorProposedBlocks(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	addr := c.Params("operatorAddr")
	response, err := h.service.GetValidatorProposedBlocks(*pagination, addr)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}

// GetValidatorHistoricalPowers godoc
// @Summary Get validator historical powers
// @Description Retrieves historical powers of a validator to be rendered
// @Tags Validator
// @Produce json
// @Param operatorAddr path string true "Validator operator address"
// @Success 200 {object} dto.ValidatorHistoricalPowersResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/validator/v1/validators/{operatorAddr}/historical-powers [get]
func (h *ValidatorHandler) GetValidatorHistoricalPowers(c *fiber.Ctx) error {
	addr := c.Params("operatorAddr")
	response, err := h.service.GetValidatorHistoricalPowers(addr)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}

// GetValidatorVotedProposals godoc
// @Summary Get validator voted proposals
// @Description Retrieves list of voted governance proposals from a validator
// @Tags Validator
// @Produce json
// @Param operatorAddr path string true "Validator operator address"
// @Param search query string false "Search validators by moniker or exact operator address" default()
// @Param answer query string false "Filter by given answer" default()
// @Success 200 {object} dto.ValidatorVotedProposalsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/validator/v1/validators/{operatorAddr}/voted-proposals [get]
func (h *ValidatorHandler) GetValidatorVotedProposals(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	addr := c.Params("operatorAddr")
	search := c.Query("search")
	answer := c.Query("answer")

	proposals, err := h.service.GetValidatorVotedProposals(*pagination, addr, search, answer)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}
	return c.JSON(proposals)
}

// GetValidatorAnswerCounts godoc
// @Summary Get validator proposal answers count
// @Description Get validator voted governance proposal answers count
// @Tags Validator
// @Produce json
// @Param operatorAddr path string true "Validator operator address"
// @Success 200 {object} dto.ValidatorAnswerCountsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/validator/v1/validators/{operatorAddr}/answer-counts [get]
func (h *ValidatorHandler) GetValidatorAnswerCounts(c *fiber.Ctx) error {
	addr := c.Params("operatorAddr")

	counts, err := h.service.GetValidatorAnswerCounts(addr)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}
	return c.JSON(counts)
}
