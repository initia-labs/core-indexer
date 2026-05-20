package indexer

import (
	"context"

	"github.com/initia-labs/core-indexer/pkg/db"
	"gorm.io/gorm"
)

func runPatcher(ctx context.Context, dbClient *gorm.DB, chainID string) error {
	logger.Info().Msgf("Starting patcher for chain %s", chainID)

	if chainID == "interwoven-1" {
		logger.Info().Msgf("Applying interwoven-1 specific patches")
		err := db.UpdateOnlyFailedReason(ctx, dbClient, []db.Proposal{{ID: 83, FailedReason: "rate limit not found"}})
		if err != nil {
			logger.Error().Msgf("Error updating only failed reasons: %v", err)
			return err
		}
		logger.Info().Msgf("Successfully updated failed reasons")
	}

	logger.Info().Msgf("Done running patcher for chain %s", chainID)
	return nil
}
