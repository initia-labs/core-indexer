package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/services"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

type TxHandler struct {
	service services.TxService
}

func NewTxHandler(service services.TxService) *TxHandler {
	return &TxHandler{
		service: service,
	}
}

// GetTxCount godoc
// @Summary  Get transaction count
// @Description Retrieve the total number of transactions
// @Tags Transaction
// @Produce json
// @Success 200 {integer} int64 "Transaction count"
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/tx/v1/txs/count [get]
func (h *TxHandler) GetTxCount(c *fiber.Ctx) error {
	txCount, err := h.service.GetTxCount()
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(txCount)
}

// GetTxsByAccountAddress godoc
// @Summary  Get transactions by account address
// @Description Retrieve transactions by account address
// @Tags Transaction
// @Produce json
// @Param accountAddress path string true "Account address"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.reverse query boolean false "Reverse order for pagination" default(false)
// @Param pagination.count_total query boolean false "Count total number of transactions" default(false)
// @Success 200 {object} dto.TxsResponse "OK"
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/tx/v1/txs/by_account/{accountAddress} [get]
func (h *TxHandler) GetTxsByAccountAddress(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	accountAddress, err := parser.AccAddressFromString(c.Params("accountAddress"))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetTxsByAccountAddress(c.UserContext(), *pagination, accountAddress.String())
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

func (h *TxHandler) GetTxsByBlockHeight(c *fiber.Ctx) error {
	return nil
}

// GetTxByHash godoc
// @Summary Get transaction by hash
// @Description Retrieve transaction details by its hash
// @Tags Transaction
// @Accept json
// @Produce json
// @Param tx_hash path string true "Transaction hash"
// @Success 200 {object} dto.TxByHashResponse "OK"
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/tx/v1/txs/{tx_hash} [get]
func (h *TxHandler) GetTxByHash(c *fiber.Ctx) error {
	hash := c.Params("tx_hash")
	tx, err := h.service.GetTxByHash(c.UserContext(), hash)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}
	return c.JSON(tx)
}

// GetTxs godoc
// @Summary Get transactions
// @Description Retrieve a list of transactions with pagination
// @Tags Transaction
// @Accept json
// @Produce json
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.reverse query boolean false "Reverse order for pagination" default(false)
// @Param pagination.count_total query boolean false "Count total number of transactions" default(false)
// @Success 200 {object} dto.TxsModelResponse "OK"
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/tx/v1/txs [get]
func (h *TxHandler) GetTxs(c *fiber.Ctx) error {
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetTxs(pagination)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}
