package routes

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories/raw"
	"github.com/initia-labs/core-indexer/api/services"
	"gocloud.dev/blob"
)

// SetupTxRoutes sets up the Tx routes
func SetupTxRoutes(app *fiber.App, db *sql.DB, bucket *blob.Bucket) {
	// Initialize repositories
	// Change this to gorm when we have gorm
	txRepo := raw.NewTxRepository(db, bucket)

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
