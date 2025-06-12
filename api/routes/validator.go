package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/services"
)

func SetupValidatorRoutes(app *fiber.App, validatorRepo repositories.ValidatorRepositoryI, blockRepo repositories.BlockRepositoryI, proposalRepo repositories.ProposalRepositoryI) {
	validatorService := services.NewValidatorService(validatorRepo, blockRepo, proposalRepo)
	validatorHandler := handlers.NewValidatorHandler(validatorService)

	v1 := app.Group("/indexer/validator/v1")

	validators := v1.Group("/validators")
	validators.Get("/", validatorHandler.GetValidators)

	validator := validators.Group("/:operatorAddr")
	validator.Get("/info", validatorHandler.GetValidatorInfo)
	validator.Get("/uptime", validatorHandler.GetValidatorUptime)
	validator.Get("/delegation-related-txs", validatorHandler.GetValidatorDelegationRelatedTxs)
	validator.Get("/proposed-blocks", validatorHandler.GetValidatorProposedBlocks)
	validator.Get("/historical-powers", validatorHandler.GetValidatorHistoricalPowers)
	validator.Get("/voted-proposals", validatorHandler.GetValidatorVotedProposals)
	validator.Get("/answer-counts", validatorHandler.GetValidatorAnswerCounts)
}
