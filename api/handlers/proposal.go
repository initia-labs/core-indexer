package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/initia-labs/core-indexer/pkg/parser"

	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/services"
)

type ProposalHandler struct {
	service services.ProposalService
}

func NewProposalHandler(service services.ProposalService) *ProposalHandler {
	return &ProposalHandler{
		service: service,
	}
}

// GetProposals godoc
//
//	@Summary		Get list of proposals
//	@Description	Retrieve the list of all proposals
//	@Tags			Proposal
//	@Produce		json
//	@Param			pagination.offset		query		integer	false	"Offset for pagination"								default(0)
//	@Param			pagination.limit		query		integer	false	"Limit for pagination"								default(10)
//	@Param			pagination.reverse		query		boolean	false	"Reverse order for pagination"						default(false)
//	@Param			pagination.count_total	query		boolean	false	"Count total number of transactions"				default(false)
//	@Param			proposer				query		string	false	"Filter proposals by proposer"						default()
//	@Param			statuses				query		string	false	"Filter proposals by status(es)"					default()
//	@Param			types					query		string	false	"Filter proposals by proposal type"					default()
//	@Param			search					query		string	false	"Search proposals by title or exact proposal id"	default()
//	@Success		200						{object}	dto.ProposalsResponse
//	@Failure		400						{object}	apperror.Response
//	@Failure		500						{object}	apperror.Response
//	@Router			/indexer/proposal/v1/proposals [get]
func (h *ProposalHandler) GetProposals(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	proposerStr := c.Query("proposer")
	if proposerStr != "" {
		if proposer, err := parser.AccAddressFromString(proposerStr); err != nil {
			return apperror.HandleErrorResponse(c, err)
		} else {
			proposerStr = proposer.String()
		}
	}
	statuses := c.Query("statuses")
	types := c.Query("types")
	search := c.Query("search")

	proposals, err := h.service.GetProposals(*pagination, proposerStr, statuses, types, search)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(proposals)
}

// GetProposalsTypes godoc
//
//	@Summary		Get list of submitted proposal types
//	@Description	Retrieve all submitted proposal types
//	@Tags			Proposal
//	@Produce		json
//	@Success		200	{object}	dto.ProposalsTypesResponse
//	@Failure		400	{object}	apperror.Response
//	@Failure		500	{object}	apperror.Response
//	@Router			/indexer/proposal/v1/proposals/types [get]
func (h *ProposalHandler) GetProposalsTypes(c *fiber.Ctx) error {
	types, err := h.service.GetProposalsTypes()
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(types)
}

// GetProposalInfo godoc
//
//	@Summary		Get proposal info
//	@Description	Retrieve proposal details
//	@Tags			Proposal
//	@Produce		json
//	@Param			proposalId	path		string	true	"Proposal Id"
//	@Success		200			{object}	dto.ProposalInfoResponse
//	@Failure		400			{object}	apperror.Response
//	@Failure		500			{object}	apperror.Response
//	@Router			/indexer/proposal/v1/proposals/{proposalId}/info [get]
func (h *ProposalHandler) GetProposalInfo(c *fiber.Ctx) error {
	parsedId, err := strconv.ParseInt(c.Params("proposalId"), 10, 32)
	if err != nil {
		return apperror.HandleErrorResponse(c, apperror.NewValidationError(apperror.ErrMsgProposalId))
	}

	id := int(parsedId)
	proposal, err := h.service.GetProposalInfo(id)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(proposal)
}

// GetProposalVotes godoc
//
//	@Summary		Get all proposal votes
//	@Description	Retrieve list of all proposal votes
//	@Tags			Proposal
//	@Produce		json
//	@Param			pagination.offset		query		integer	false	"Offset for pagination"					default(0)
//	@Param			pagination.limit		query		integer	false	"Limit for pagination"					default(10)
//	@Param			pagination.reverse		query		boolean	false	"Reverse order for pagination"			default(false)
//	@Param			pagination.count_total	query		boolean	false	"Count total number of transactions"	default(false)
//	@Param			proposalId				path		string	true	"Proposal Id"
//	@Param			search					query		string	false	"Search proposal vote by voter address"	default()
//	@Param			answer					query		string	false	"Search proposal votes by vote option"	default()
//	@Success		200						{object}	dto.ProposalVotesResponse
//	@Failure		400						{object}	apperror.Response
//	@Failure		500						{object}	apperror.Response
//	@Router			/indexer/proposal/v1/proposals/{proposalId}/votes [get]
func (h *ProposalHandler) GetProposalVotes(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	parsedId, err := strconv.ParseInt(c.Params("proposalId"), 10, 32)
	if err != nil {
		return apperror.HandleErrorResponse(c, apperror.NewValidationError(apperror.ErrMsgProposalId))
	}

	search := c.Query("search")
	answer := c.Query("answer")

	id := int(parsedId)
	proposals, err := h.service.GetProposalVotes(*pagination, id, search, answer)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(proposals)
}

// GetProposalValidatorVotes godoc
//
//	@Summary		Get validator votes of a proposal
//	@Description	Retrieve list of all proposal votes by validators
//	@Tags			Proposal
//	@Produce		json
//	@Param			pagination.offset		query		integer	false	"Offset for pagination"					default(0)
//	@Param			pagination.limit		query		integer	false	"Limit for pagination"					default(10)
//	@Param			pagination.reverse		query		boolean	false	"Reverse order for pagination"			default(false)
//	@Param			pagination.count_total	query		boolean	false	"Count total number of transactions"	default(false)
//	@Param			proposalId				path		string	true	"Proposal Id"
//	@Param			search					query		string	false	"Search proposal vote by validator moniker or address"	default()
//	@Param			answer					query		string	false	"Search proposal votes by vote option"					default()
//	@Success		200						{object}	dto.ProposalValidatorVotesResponse
//	@Failure		400						{object}	apperror.Response
//	@Failure		500						{object}	apperror.Response
//	@Router			/indexer/proposal/v1/proposals/{proposalId}/validator_votes [get]
func (h *ProposalHandler) GetProposalValidatorVotes(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	parsedId, err := strconv.ParseInt(c.Params("proposalId"), 10, 32)
	if err != nil {
		return apperror.HandleErrorResponse(c, apperror.NewValidationError(apperror.ErrMsgProposalId))
	}

	search := c.Query("search")
	answer := c.Query("answer")

	id := int(parsedId)
	proposals, err := h.service.GetProposalValidatorVotes(*pagination, id, search, answer)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(proposals)
}

// GetProposalAnswerCounts godoc
//
//	@Summary		Get votes count of a proposal
//	@Description	Retrieve vote counts of a proposal by options
//	@Tags			Proposal
//	@Produce		json
//	@Param			proposalId	path		string	true	"Proposal Id"
//	@Success		200			{object}	dto.ProposalAnswerCountsResponse
//	@Failure		400			{object}	apperror.Response
//	@Failure		500			{object}	apperror.Response
//	@Router			/indexer/proposal/v1/proposals/{proposalId}/answer_counts [get]
func (h *ProposalHandler) GetProposalAnswerCounts(c *fiber.Ctx) error {
	parsedId, err := strconv.ParseInt(c.Params("proposalId"), 10, 32)
	if err != nil {
		return apperror.HandleErrorResponse(c, apperror.NewValidationError(apperror.ErrMsgProposalId))
	}

	id := int(parsedId)
	counts, err := h.service.GetProposalAnswerCounts(id)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(counts)
}
