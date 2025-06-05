package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories/raw"
	"github.com/initia-labs/core-indexer/api/services"
	"gocloud.dev/blob"
	"gorm.io/gorm"
)

// SetupTxRoutes sets up the Tx routes
func SetupTxRoutes(app *fiber.App, dbClient *gorm.DB, bucket *blob.Bucket) {
	// Initialize repositories
	// Change this to gorm when we have gorm
	txRepo := raw.NewTxRepository(dbClient, bucket)

	// Initialize services
	txService := services.NewTxService(txRepo)

	// Initialize handlers
	txHandler := handlers.NewTxHandler(txService)

	// Tx routes
	v1 := app.Group("/tx/v1")
	{
		// Txs
		txs := v1.Group("/txs")
		{
			txs.Get("/:hash", txHandler.GetTxByHash)
		}
	}
}
