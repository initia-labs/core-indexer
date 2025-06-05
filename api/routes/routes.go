package routes

import (
	"github.com/gofiber/fiber/v2"
	"gocloud.dev/blob"
	"gorm.io/gorm"
)

// SetupRoutes configures all the routes for the API
func SetupRoutes(app *fiber.App, dbClient *gorm.DB, bucket *blob.Bucket) {
	// Setup NFT routes
	SetupNFTRoutes(app, dbClient)
	SetupTxRoutes(app, dbClient, bucket)
}
