package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/services"
)

type AccountHandler struct {
	service services.AccountService
}

func NewAccountHandler(service services.AccountService) *AccountHandler {
	return &AccountHandler{
		service: service,
	}
}

// GetAccountByAccountAddress godoc
// @Summary Get account by address
// @Description Retrieve account details by account address
// @Tags Account
// @Accept json
// @Produce json
// @Param accountAddress path string true "Account address"
// @Success 200 {object} db.Account
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/account/v1/{accountAddress} [get]
func (h *AccountHandler) GetAccountByAccountAddress(c *fiber.Ctx) error {
	accountAddress := strings.ToLower(c.Params("accountAddress"))

	response, err := h.service.GetAccountByAccountAddress(accountAddress)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetAccountProposals godoc
// @Summary Get account proposals
// @Description Retrieve proposals associated with an account
// @Tags Account
// @Accept json
// @Produce json
// @Param accountAddress path string true "Account address"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Whether to count total Nfts" default(false)
// @Param pagination.reverse query boolean false "Whether to reverse the order of transactions" default(true)
// @Success 200 {object} dto.AccountProposalsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/account/v1/{accountAddress}/proposals [get]
func (h *AccountHandler) GetAccountProposals(c *fiber.Ctx) error {
	accountAddress := strings.ToLower(c.Params("accountAddress"))

	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetAccountProposals(*pagination, accountAddress)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetAccountTxs godoc
// @Summary Get account transactions
// @Description Retrieve transactions associated with an account
// @Tags Account
// @Accept json
// @Produce json
// @Param accountAddress path string true "Account address"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Whether to count total transactions" default(false)
// @Param pagination.reverse query boolean false "Whether to reverse the order of transactions" default(true)
// @Param search query string false "Search term for transactions"
// @Param is_send query boolean false "Filter by sent transactions" default(false)
// @Param is_ibc query boolean false "Filter by IBC transactions" default(false)
// @Param is_opinit query boolean false "Filter by OPINIT transactions" default(false)
// @Param is_move_publish query boolean false "Filter by Move publish transactions" default(false)
// @Param is_move_upgrade query boolean false "Filter by Move upgrade transactions" default(false)
// @Param is_move_execute query boolean false "Filter by Move execute transactions" default(false)
// @Param is_move_script query boolean false "Filter by Move script transactions" default(false)
// @Param is_signer query boolean false "Filter by transactions where the account is a signer"
// @Success 200 {array} dto.AccountTxsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/account/v1/{accountAddress}/txs [get]
func (h *AccountHandler) GetAccountTxs(c *fiber.Ctx) error {
	accountAddress := strings.ToLower(c.Params("accountAddress"))

	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	search := c.Query("search")
	isSend := c.Query("is_send") == "true"
	isIbc := c.Query("is_ibc") == "true"
	isOpinit := c.Query("is_opinit") == "true"
	isMovePublish := c.Query("is_move_publish") == "true"
	isMoveUpgrade := c.Query("is_move_upgrade") == "true"
	isMoveExecute := c.Query("is_move_execute") == "true"
	isMoveScript := c.Query("is_move_script") == "true"
	isSignerStr := c.Query("is_signer")

	var isSigner *bool

	if isSignerStr != "" {
		isSignerValue := isSignerStr == "true"
		isSigner = &isSignerValue
	}

	response, err := h.service.GetAccountTxs(
		*pagination,
		accountAddress,
		search,
		isSend,
		isIbc,
		isOpinit,
		isMovePublish,
		isMoveUpgrade,
		isMoveExecute,
		isMoveScript,
		isSigner,
	)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}
