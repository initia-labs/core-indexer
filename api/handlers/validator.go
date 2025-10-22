package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

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
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.reverse query boolean false "Reverse order for pagination" default(false)
// @Param pagination.count_total query boolean false "Count total number of transactions" default(false)
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
		return apperror.HandleErrorResponse(c, err)
	}

	isActive := true
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActive = isActiveStr == "true"
	}
	sortBy := c.Query("sort_by")
	search := c.Query("search")

	response, err := h.service.GetValidators(c.UserContext(), *pagination, isActive, sortBy, search)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
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
	val, err := h.service.GetValidatorInfo(c.UserContext(), addr)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
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
			return apperror.HandleErrorResponse(c, apperror.NewValidationError(apperror.ErrMsgBlocks))
		} else if block <= 0 {
			return apperror.HandleErrorResponse(c, apperror.NewValidationError(apperror.ErrMsgBlocksZero))
		} else {
			blocks = block
		}
	} else {
		return apperror.HandleErrorResponse(c, apperror.NewValidationError(apperror.ErrMsgBlocksRequired))
	}

	uptime, err := h.service.GetValidatorUptime(c.UserContext(), addr, blocks)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}
	return c.JSON(uptime)
}

// GetValidatorDelegationRelatedTxs godoc
// @Summary Get delegation transactions of a validator
// @Description Retrieves list of delegation related of a validator
// @Tags Validator
// @Produce json
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.reverse query boolean false "Reverse order for pagination" default(false)
// @Param pagination.count_total query boolean false "Count total number of transactions" default(false)
// @Param operatorAddr path string true "Validator operator address"
// @Success 200 {object} dto.ValidatorDelegationRelatedTxsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/validator/v1/validators/{operatorAddr}/delegation_related_txs [get]
func (h *ValidatorHandler) GetValidatorDelegationRelatedTxs(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	addr := c.Params("operatorAddr")
	response, err := h.service.GetValidatorDelegationTxs(c.UserContext(), *pagination, addr)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetValidatorProposedBlocks godoc
// @Summary Get validator proposed blocks
// @Description Retrieves list of proposed blocks from a validator
// @Tags Validator
// @Produce json
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.reverse query boolean false "Reverse order for pagination" default(false)
// @Param pagination.count_total query boolean false "Count total number of transactions" default(false)
// @Param operatorAddr path string true "Validator operator address"
// @Success 200 {object} dto.ValidatorProposedBlocksResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/validator/v1/validators/{operatorAddr}/proposed_blocks [get]
func (h *ValidatorHandler) GetValidatorProposedBlocks(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	addr := c.Params("operatorAddr")
	response, err := h.service.GetValidatorProposedBlocks(c.UserContext(), *pagination, addr)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
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
// @Router /indexer/validator/v1/validators/{operatorAddr}/historical_powers [get]
func (h *ValidatorHandler) GetValidatorHistoricalPowers(c *fiber.Ctx) error {
	addr := c.Params("operatorAddr")
	response, err := h.service.GetValidatorHistoricalPowers(c.UserContext(), addr)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetValidatorVotedProposals godoc
// @Summary Get validator voted proposals
// @Description Retrieves list of voted governance proposals from a validator
// @Tags Validator
// @Produce json
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.reverse query boolean false "Reverse order for pagination" default(false)
// @Param pagination.count_total query boolean false "Count total number of transactions" default(false)
// @Param operatorAddr path string true "Validator operator address"
// @Param search query string false "Search validators by moniker or exact operator address" default()
// @Param answer query string false "Filter by given answer" default()
// @Success 200 {object} dto.ValidatorVotedProposalsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/validator/v1/validators/{operatorAddr}/voted_proposals [get]
func (h *ValidatorHandler) GetValidatorVotedProposals(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	addr := c.Params("operatorAddr")
	search := c.Query("search")
	answer := c.Query("answer")

	proposals, err := h.service.GetValidatorVotedProposals(c.UserContext(), *pagination, addr, search, answer)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
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
// @Router /indexer/validator/v1/validators/{operatorAddr}/answer_counts [get]
func (h *ValidatorHandler) GetValidatorAnswerCounts(c *fiber.Ctx) error {
	addr := c.Params("operatorAddr")

	counts, err := h.service.GetValidatorAnswerCounts(c.UserContext(), addr)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}
	return c.JSON(counts)
}
