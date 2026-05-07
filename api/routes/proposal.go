package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/initia-labs/core-indexer/api/handlers"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/services"
)

func SetupProposalRoutes(app *fiber.App, repo repositories.ProposalRepositoryI) {
	proposalService := services.NewProposalService(repo)
	proposalHandler := handlers.NewProposalHandler(proposalService)

	v1 := app.Group("/indexer/proposal/v1")

	proposals := v1.Group("/proposals")
	proposals.Get("/", proposalHandler.GetProposals)
	proposals.Get("/types", proposalHandler.GetProposalsTypes)

	proposal := proposals.Group("/:proposalId")
	proposal.Get("/info", proposalHandler.GetProposalInfo)
	proposal.Get("/votes", proposalHandler.GetProposalVotes)
	proposal.Get("/validator_votes", proposalHandler.GetProposalValidatorVotes)
	proposal.Get("/answer_counts", proposalHandler.GetProposalAnswerCounts)
}
