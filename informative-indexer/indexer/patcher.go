package indexer

import (
	"context"

	"github.com/initia-labs/core-indexer/pkg/db"
	"gorm.io/gorm"
)

func runPatcher(ctx context.Context, dbClient *gorm.DB, chainID string) {
	if chainID == "interwoven-1" {
		db.UpdateOnlyExpeditedProposalStatus(ctx, dbClient, []db.Proposal{{ID: 62, IsExpedited: true}})
		return
	}
}
