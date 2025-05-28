package routes

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures all the routes for the API
func SetupRoutes(app *fiber.App, db *sql.DB) {
	// Setup NFT routes
	SetupNFTRoutes(app, db)
	// setupTxRoutes(app)
}
