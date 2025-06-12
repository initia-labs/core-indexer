package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/services"
	"gorm.io/gorm"
)

func SetupAccountRoutes(app *fiber.App, dbClient *gorm.DB) {
	// Initialize repositories
	accountRepo := repositories.NewAccountRepository(dbClient)

	// Initialize services
	accountService := services.NewAccountService(accountRepo)

	// Initialize handlers
	accountHandler := handlers.NewAccountHandler(accountService)

	// Account routes
	v1 := app.Group("/indexer/account/v1")
	{
		v1.Get("/:accountAddress", accountHandler.GetAccountByAccountAddress)
		v1.Get("/:accountAddress/proposals", accountHandler.GetAccountProposals)
		v1.Get("/:accountAddress/txs", accountHandler.GetAccountTxs)
	}
}
