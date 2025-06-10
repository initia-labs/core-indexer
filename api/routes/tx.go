package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/services"
)

// SetupTxRoutes sets up the Tx routes
func SetupTxRoutes(app *fiber.App, txRepo *repositories.TxRepository) {
	// Initialize services
	txService := services.NewTxService(txRepo)

	// Initialize handlers
	txHandler := handlers.NewTxHandler(txService)

	// Tx routes
	v1 := app.Group("/indexer/tx/v1")

	txs := v1.Group("/txs")

	txs.Get("/", txHandler.GetTxs)
	txs.Get("/count", txHandler.GetTxCount)
	txs.Get("/:tx_hash", txHandler.GetTxByHash)
}
