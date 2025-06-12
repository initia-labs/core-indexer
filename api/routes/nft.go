package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/services"
)

// SetupNFTRoutes sets up the NFT routes
func SetupNFTRoutes(app *fiber.App, nftRepo repositories.NFTRepositoryI) {
	// Initialize services
	nftService := services.NewNFTService(nftRepo)

	// Initialize handlers
	nftHandler := handlers.NewNFTHandler(nftService)

	// NFT routes
	v1 := app.Group("/nft/v1")
	{
		// Collections
		collections := v1.Group("/collections")
		{
			collections.Get("/", nftHandler.GetCollections)
		}
	}
}
