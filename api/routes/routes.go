package routes

import (
	"github.com/gofiber/fiber/v2"
	"gocloud.dev/blob"
	"gorm.io/gorm"
)

// SetupRoutes configures all the routes for the API
func SetupRoutes(app *fiber.App, dbClient *gorm.DB, bucket *blob.Bucket) {
	SetupNFTRoutes(app, dbClient)
	SetupTxRoutes(app, dbClient, bucket)
	SetupBlockRoutes(app, dbClient)
	SetupAccountRoutes(app, dbClient)
}
