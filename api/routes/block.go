package routes

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories/raw"
	"github.com/initia-labs/core-indexer/api/services"
)

func SetupBlockRoutes(app *fiber.App, db *sql.DB) {
	blockRepo := raw.NewBlockRepository(db)

	blockService := services.NewBlockService(blockRepo)

	blockHandler := handlers.NewBlockHandler(blockService)

	v1 := app.Group("/indexer/block/v1")
	{
		v1.Get("/latest_block_height", blockHandler.GetBlockHeightLatest)
		v1.Get("/avg_block_time", blockHandler.GetBlockTimeAverage)
	}
}
