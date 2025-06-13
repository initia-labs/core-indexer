package routes

import (
	"github.com/gofiber/fiber/v2"
	"gocloud.dev/blob"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/api/repositories"
)

// SetupRoutes configures all the routes for the API
func SetupRoutes(app *fiber.App, dbClient *gorm.DB, bucket *blob.Bucket) {
	repos := repositories.SetupRepositories(dbClient, bucket)

	SetupBlockRoutes(app, repos.BlockRepository)
	SetupModuleRoutes(app, repos.ModuleRepository)
	SetupNFTRoutes(app, repos.NftRepository)
	SetupProposalRoutes(app, repos.ProposalRepository)
	SetupTxRoutes(app, repos.TxRepository)
	SetupValidatorRoutes(app, repos.ValidatorRepository, repos.BlockRepository, repos.ProposalRepository)
}
