package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/services"
)

func SetupBlockRoutes(app *fiber.App, blockRepo repositories.BlockRepositoryI) {
	blockService := services.NewBlockService(blockRepo)

	blockHandler := handlers.NewBlockHandler(blockService)

	v1 := app.Group("/indexer/block/v1")
	{
		v1.Get("/blocks", blockHandler.GetBlocks)
		v1.Get("/latest_block_height", blockHandler.GetBlockHeightLatest)
		v1.Get("/latest_informative_block_height", blockHandler.GetBlockHeightInformativeLatest)
		v1.Get("/avg_block_time", blockHandler.GetBlockTimeAverage)
		v1.Get("/blocks/:height/info", blockHandler.GetBlockInfo)
		v1.Get("/blocks/:height/txs", blockHandler.GetBlockTxs)
	}
}
