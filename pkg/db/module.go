package db

import (
	"context"

	"github.com/getsentry/sentry-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func UpsertModules(ctx context.Context, dbTx *gorm.DB, modules []Module) error {
	span := sentry.StartSpan(ctx, "UpsertModules")
	span.Description = "Bulk upsert modules into the database"
	defer span.Finish()

	if len(modules) == 0 {
		return nil
	}
	columns := []string{
		"upgrade_policy",
		"digest",
	}
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns(columns),
			UpdateAll: false,
		}).
		CreateInBatches(&modules, BatchSize)
	return result.Error
}

func InsertModuleTransactions(ctx context.Context, dbTx *gorm.DB, moduleTransactions []ModuleTransaction) error {
	if len(moduleTransactions) == 0 {
		return nil
	}
	return dbTx.WithContext(ctx).CreateInBatches(moduleTransactions, BatchSize).Error
}

func InsertModuleHistories(ctx context.Context, dbTx *gorm.DB, moduleHistories []ModuleHistory) error {
	if len(moduleHistories) == 0 {
		return nil
	}
	return dbTx.WithContext(ctx).CreateInBatches(moduleHistories, BatchSize).Error
}

func InsertModuleProposalsIgnoreConflict(ctx context.Context, dbTx *gorm.DB, moduleProposals []ModuleProposal) error {
	if len(moduleProposals) == 0 {
		return nil
	}
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		Create(&moduleProposals)
	return result.Error
}
