package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

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
// @Summary Get list of proposals
// @Description Retrieve the list of all proposals
// @Tags Proposal
// @Produce json
// @Param proposer query string false "Filter proposals by proposer" default()
// @Param statuses query string false "Filter proposals by status(es)" default()
// @Param types query string false "Filter proposals by proposal type" default()
// @Param search query string false "Search proposals by title or exact proposal id" default()
// @Success 200 {object} dto.ProposalsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/proposal/v1/proposals [get]
func (h *ProposalHandler) GetProposals(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	proposer := c.Query("proposer")
	statuses := c.Query("statuses")
	types := c.Query("types")
	search := c.Query("search")

	proposals, err := h.service.GetProposals(*pagination, proposer, statuses, types, search)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(proposals)
}

// GetProposalsTypes godoc
// @Summary Get list of submitted proposal types
// @Description Retrieve all submitted proposal types
// @Tags Proposal
// @Produce json
// @Success 200 {object} dto.ProposalsTypesResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/proposal/v1/proposals/types [get]
func (h *ProposalHandler) GetProposalsTypes(c *fiber.Ctx) error {
	types, err := h.service.GetProposalsTypes()
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(types)
}

// GetProposalInfo godoc
// @Summary Get proposal info
// @Description Retrieve proposal details
// @Tags Proposal
// @Produce json
// @Param proposalId path string true "Proposal Id"
// @Success 200 {object} dto.ProposalInfoResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/proposal/v1/proposals/{proposalId}/info [get]
func (h *ProposalHandler) GetProposalInfo(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("proposalId"))
	if err != nil {
		errResp := apperror.NewBadRequest("proposal id is not a valid integer")
		return c.Status(errResp.Code).JSON(errResp)
	}

	proposal, err := h.service.GetProposalInfo(id)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(proposal)
}

// GetProposalVotes godoc
// @Summary Get all proposal votes
// @Description Retrieve list of all proposal votes
// @Tags Proposal
// @Produce json
// @Param proposalId path string true "Proposal Id"
// @Param search query string false "Search proposal vote by voter address" default()
// @Param answer query string false "Search proposal votes by vote option" default()
// @Success 200 {object} dto.ProposalVotesResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/proposal/v1/proposals/{proposalId}/votes [get]
func (h *ProposalHandler) GetProposalVotes(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	id, err := strconv.Atoi(c.Params("proposalId"))
	if err != nil {
		errResp := apperror.NewBadRequest("proposal id is not a valid integer")
		return c.Status(errResp.Code).JSON(errResp)
	}

	search := c.Query("search")
	answer := c.Query("answer")

	proposals, err := h.service.GetProposalVotes(*pagination, id, search, answer)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(proposals)
}

// GetProposalValidatorVotes godoc
// @Summary Get validator votes of a proposal
// @Description Retrieve list of all proposal votes by validators
// @Tags Proposal
// @Produce json
// @Param proposalId path string true "Proposal Id"
// @Param search query string false "Search proposal vote by validator moniker or address" default()
// @Param answer query string false "Search proposal votes by vote option" default()
// @Success 200 {object} dto.ProposalValidatorVotesResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/proposal/v1/proposals/{proposalId}/validator_votes [get]
func (h *ProposalHandler) GetProposalValidatorVotes(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	id, err := strconv.Atoi(c.Params("proposalId"))
	if err != nil {
		errResp := apperror.NewBadRequest("proposal id is not a valid integer")
		return c.Status(errResp.Code).JSON(errResp)
	}

	search := c.Query("search")
	answer := c.Query("answer")

	proposals, err := h.service.GetProposalValidatorVotes(*pagination, id, search, answer)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(proposals)
}

// GetProposalAnswerCounts godoc
// @Summary Get votes count of a proposal
// @Description Retrieve vote counts of a proposal by options
// @Tags Proposal
// @Produce json
// @Param proposalId path string true "Proposal Id"
// @Success 200 {object} dto.ProposalAnswerCountsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/proposal/v1/proposals/{proposalId}/answer_counts [get]
func (h *ProposalHandler) GetProposalAnswerCounts(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("proposalId"))
	if err != nil {
		errResp := apperror.NewBadRequest("proposal id is not a valid integer")
		return c.Status(errResp.Code).JSON(errResp)
	}

	counts, err := h.service.GetProposalAnswerCounts(id)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(counts)
}
