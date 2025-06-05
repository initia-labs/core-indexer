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
	return &TxHandler{service: service}
}

// GetTxByHash godoc
// @Summary Get transaction by hash
// @Description Retrieve transaction details by its hash
// @Tags Transaction
// @Accept json
// @Produce json
// @Param hash path string true "Transaction hash"
// @Success 200 {object} dto.TxResponse
// @Failure 400 {object} apperror.Response
// @Failure 404 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /tx/v1/txs/{hash} [get]
func (h *TxHandler) GetTxByHash(c *fiber.Ctx) error {
	hash := c.Params("hash")
	tx, err := h.service.GetTxByHash(hash)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}
	return c.JSON(tx)
}
