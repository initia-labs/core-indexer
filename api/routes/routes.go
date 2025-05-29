package routes

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures all the routes for the API
func SetupRoutes(app *fiber.App, db *sql.DB) {
	SetupNFTRoutes(app, db)
	SetupBlockRoutes(app, db)
	SetupTxRoutes(app, db)
}
