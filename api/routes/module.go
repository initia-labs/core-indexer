package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/services"
)

func SetupModuleRoutes(app *fiber.App, db *gorm.DB) {
	// Initialize repositories
	moduleRepo := repositories.NewModuleRepository(db)

	// Initialize services
	moduleService := services.NewModuleService(moduleRepo)

	// Initialize handlers
	moduleHandler := handlers.NewModuleHandler(moduleService)

	// Module routes
	v1 := app.Group("/indexer/module/v1")
	{
		// Modules
		modules := v1.Group("/modules")
		{
			modules.Get("/", moduleHandler.GetModules)
			modules.Get("/:vmAddress/:name", moduleHandler.GetModuleById)
			modules.Get("/:vmAddress/:name/histories", moduleHandler.GetModuleHistories)
			modules.Get("/:vmAddress/:name/publish_info", moduleHandler.GetModulePublishInfo)
			modules.Get("/:vmAddress/:name/proposals", moduleHandler.GetModuleProposals)
			modules.Get("/:vmAddress/:name/transactions", moduleHandler.GetModuleTransactions)
			modules.Get("/:vmAddress/:name/stats", moduleHandler.GetModuleStats)
		}
	}
}
