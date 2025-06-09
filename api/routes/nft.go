package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/services"
)

// SetupNFTRoutes sets up the NFT routes
func SetupNFTRoutes(app *fiber.App, dbClient *gorm.DB) {
	// Initialize repositories
	nftRepo := repositories.NewNFTRepository(dbClient)

	// Initialize services
	nftService := services.NewNFTService(nftRepo)

	// Initialize handlers
	nftHandler := handlers.NewNFTHandler(nftService)

	// NFT routes
	v1 := app.Group("/indexer/nft/v1")
	{
		// Collections
		collections := v1.Group("/collections")
		{
			collections.Get("/", nftHandler.GetCollections)
			// collections.Get("/by_account/:accountAddress", nftHandler.GetCollectionsByAccountAddress)
			// collections.Get("/collectionAddress", nftHandler.GetCollectionsByCollectionAddress)
		}

		tokens := v1.Group("/tokens")
		{
			tokens.Get("/by_collection/:collectionAddress/:nftAddress", nftHandler.GetNFTByNFTAddress)
			tokens.Get("/by_collection/:collectionAddress", nftHandler.GetNFTsByCollectionAddress)
			tokens.Get("/by_account/:accountAddress", nftHandler.GetNFTsByAccountAddress)
		}

		token := v1.Group("/token")
		{
			token.Get("/:nftAddress/mint_info", nftHandler.GetNFTMintInfo)
			token.Get("/:nftAddress/mutate_events", nftHandler.GetNFTMutateEvents)
			token.Get("/:nftAddress/txs", nftHandler.GetNFTTxs)
		}
	}
}
