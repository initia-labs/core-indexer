package routes

import (
	"github.com/gofiber/fiber/v2"
	"gocloud.dev/blob"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/api/config"
	"github.com/initia-labs/core-indexer/api/repositories"
)

// SetupRoutes configures all the routes for the API
func SetupRoutes(app *fiber.App, dbClient *gorm.DB, buckets []*blob.Bucket, config *config.Config) {
	repos := repositories.SetupRepositories(dbClient, buckets, config.Repository.CountQueryTimeout)

	SetupBlockRoutes(app, repos.BlockRepository)
	SetupModuleRoutes(app, repos.ModuleRepository)
	SetupNftRoutes(app, repos.NftRepository)
	SetupProposalRoutes(app, repos.ProposalRepository)
	SetupTxRoutes(app, repos.TxRepository, repos.AccountRepository)
	SetupValidatorRoutes(app, repos.ValidatorRepository, repos.BlockRepository, repos.ProposalRepository)
	SetupAccountRoutes(app, repos.AccountRepository)
}
