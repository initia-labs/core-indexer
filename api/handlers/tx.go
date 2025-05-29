package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/dto"
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
// @Tags Tx
// @Success 200 {integer} int64 "Transaction count"
// @Failure 500 {object} dto.ErrorResponse
// @Router /indexer/tx/v1/txs/count [get]
func (h *TxHandler) GetTxCount(c *fiber.Ctx) error {
	txCount, err := h.service.GetTxCount()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error: "Failed to get transaction count",
			Code:  fiber.StatusInternalServerError,
		})
	}

	return c.JSON(txCount)
}
