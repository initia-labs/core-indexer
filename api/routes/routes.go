package routes

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"gocloud.dev/blob"
)

// SetupRoutes configures all the routes for the API
func SetupRoutes(app *fiber.App, db *sql.DB, bucket *blob.Bucket) {
	// Setup NFT routes
	SetupNFTRoutes(app, db)
	SetupTxRoutes(app, db, bucket)
}
