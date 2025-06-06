package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/initia-labs/core-indexer/api/services"
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
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(txCount)
}

// GetTxByHash godoc
// @Summary Get transaction by hash
// @Description Retrieve transaction details by its hash
// @Tags Transaction
// @Accept json
// @Produce json
// @Param tx_hash path string true "Transaction hash"
// @Success 200 {object} dto.TxResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/tx/v1/txs/{tx_hash} [get]
func (h *TxHandler) GetTxByHash(c *fiber.Ctx) error {
	hash := c.Params("tx_hash")
	tx, err := h.service.GetTxByHash(hash)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}
	return c.JSON(tx)
}
