package routes

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories/raw"
	"github.com/initia-labs/core-indexer/api/services"
)

// SetupNFTRoutes sets up the NFT routes
func SetupNFTRoutes(app *fiber.App, db *sql.DB) {
	// Initialize repositories
	// Change this to gorm when we have gorm
	nftRepo := raw.NewNFTRepository(db)

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
