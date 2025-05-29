package routes

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories/raw"
	"github.com/initia-labs/core-indexer/api/services"
)

func SetupTxRoutes(app *fiber.App, db *sql.DB) {
	txRepo := raw.NewTxRepository(db)

	txService := services.NewTxService(txRepo)

	txHandler := handlers.NewTxHandler(txService)

	v1 := app.Group(("/indexer/tx/v1/txs"))
	{
		v1.Get("/count", txHandler.GetTxCount)
	}
}
