package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/services"
)

func SetupValidatorRoutes(app *fiber.App, validatorRepo repositories.ValidatorRepositoryI, blockRepo repositories.BlockRepositoryI, proposalRepo repositories.ProposalRepositoryI) {
	// Create Keybase service with 24 hour cache TTL
	keybaseService := services.NewKeybaseService(24 * time.Hour)
	
	validatorService := services.NewValidatorService(validatorRepo, blockRepo, proposalRepo, keybaseService)
	validatorHandler := handlers.NewValidatorHandler(validatorService)

	v1 := app.Group("/indexer/validator/v1")

	validators := v1.Group("/validators")
	validators.Get("/", validatorHandler.GetValidators)

	validator := validators.Group("/:operatorAddr")
	validator.Get("/info", validatorHandler.GetValidatorInfo)
	validator.Get("/uptime", validatorHandler.GetValidatorUptime)
	validator.Get("/delegation_related_txs", validatorHandler.GetValidatorDelegationRelatedTxs)
	validator.Get("/proposed_blocks", validatorHandler.GetValidatorProposedBlocks)
	validator.Get("/historical_powers", validatorHandler.GetValidatorHistoricalPowers)
	validator.Get("/voted_proposals", validatorHandler.GetValidatorVotedProposals)
	validator.Get("/answer_counts", validatorHandler.GetValidatorAnswerCounts)
}
