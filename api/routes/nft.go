package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/services"
)

// SetupNftRoutes sets up the Nft routes
func SetupNftRoutes(app *fiber.App, nftRepo repositories.NftRepositoryI) {
	// Initialize services
	nftService := services.NewNftService(nftRepo)

	// Initialize handlers
	nftHandler := handlers.NewNftHandler(nftService)

	// Nft routes
	v1 := app.Group("/indexer/nft/v1")
	{
		// Collections
		collections := v1.Group("/collections")
		{
			collections.Get("/", nftHandler.GetCollections)
			collections.Get("/by_account/:accountAddress", nftHandler.GetCollectionsByAccountAddress)
			collections.Get("/:collectionAddress", nftHandler.GetCollectionsByCollectionAddress)
			collections.Get("/:collectionAddress/activities", nftHandler.GetCollectionActivities)
			collections.Get("/:collectionAddress/creator", nftHandler.GetCollectionCreator)
			collections.Get("/:collectionAddress/mutate_events", nftHandler.GetCollectionMutateEvents)
		}

		tokens := v1.Group("/tokens")
		{
			tokens.Get("/by_collection/:collectionAddress/:nftAddress", nftHandler.GetNftByNftAddress)
			tokens.Get("/by_collection/:collectionAddress", nftHandler.GetNftsByCollectionAddress)
			tokens.Get("/by_account/:accountAddress", nftHandler.GetNftsByAccountAddress)
		}

		token := v1.Group("/token")
		{
			token.Get("/:nftAddress/mint_info", nftHandler.GetNftMintInfo)
			token.Get("/:nftAddress/mutate_events", nftHandler.GetNftMutateEvents)
			token.Get("/:nftAddress/txs", nftHandler.GetNftTxs)
		}
	}
}
