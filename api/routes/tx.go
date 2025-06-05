package routes

import (
	"github.com/gofiber/fiber/v2"
	"gocloud.dev/blob"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/services"
)

// SetupTxRoutes sets up the Tx routes
func SetupTxRoutes(app *fiber.App, dbClient *gorm.DB, bucket *blob.Bucket) {
	// Initialize repositories
	txRepo := repositories.NewTxRepository(dbClient, bucket)

	// Initialize services
	txService := services.NewTxService(txRepo)

	// Initialize handlers
	txHandler := handlers.NewTxHandler(txService)

	// Tx routes
	v1 := app.Group("/indexer/tx/v1")

	txs := v1.Group("/txs")

	txs.Get("/count", txHandler.GetTxCount)
	txs.Get("/:tx_hash", txHandler.GetTxByHash)
}
