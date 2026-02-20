package db

import (
	"context"

	"github.com/getsentry/sentry-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func InsertTransactionIgnoreConflict(ctx context.Context, dbTx *gorm.DB, txs []Transaction) error {
	span := sentry.StartSpan(ctx, "InsertTransaction")
	span.Description = "Bulk insert transactions into the database"
	defer span.Finish()

	if len(txs) == 0 {
		return nil
	}
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(txs, BatchSize)
	return result.Error
}

func UpsertTransactions(ctx context.Context, dbTx *gorm.DB, txs []Transaction) error {
	span := sentry.StartSpan(ctx, "UpsertTransactions")
	span.Description = "Bulk upsert transactions into the database"
	defer span.Finish()

	if len(txs) == 0 {
		return nil
	}
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"is_ibc",
				"is_send",
				"is_move_publish",
				"is_move_execute_event",
				"is_move_execute",
				"is_move_upgrade",
				"is_move_script",
				"is_nft_transfer",
				"is_nft_mint",
				"is_nft_burn",
				"is_collection_create",
				"is_opinit",
				"is_instantiate",
				"is_migrate",
				"is_update_admin",
				"is_clear_admin",
				"is_store_code",
			}),
		}).
		CreateInBatches(txs, BatchSize)
	return result.Error
}

func InsertAccountTxsIgnoreConflict(ctx context.Context, dbTx *gorm.DB, txs []AccountTransaction) error {
	span := sentry.StartSpan(ctx, "InsertAccountTxs")
	span.Description = "Bulk insert account_txs into the database"
	defer span.Finish()

	if len(txs) == 0 {
		return nil
	}
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(txs, BatchSize)
	return result.Error
}

func InsertTransactionEventsIgnoreConflict(ctx context.Context, dbTx *gorm.DB, txEvents []*TransactionEvent) error {
	span := sentry.StartSpan(ctx, "InsertTransactionEvents")
	span.Description = "Bulk insert transaction_events into the database"
	defer span.Finish()

	if len(txEvents) == 0 {
		return nil
	}
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(txEvents, BatchSize)
	return result.Error
}

func InsertMoveEventsIgnoreConflict(ctx context.Context, dbTx *gorm.DB, moveEvents []*MoveEvent) error {
	span := sentry.StartSpan(ctx, "InsertMoveEvents")
	span.Description = "Bulk insert move_events into the database"
	defer span.Finish()

	if len(moveEvents) == 0 {
		return nil
	}
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(moveEvents, BatchSize)
	return result.Error
}

func InsertFinalizeBlockEventsIgnoreConflict(ctx context.Context, dbTx *gorm.DB, blockEvents []*FinalizeBlockEvent) error {
	span := sentry.StartSpan(ctx, "InsertFinalizeBlockEvents")
	span.Description = "Bulk insert finalize_block_events into the database"
	defer span.Finish()

	if len(blockEvents) == 0 {
		return nil
	}
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(blockEvents, BatchSize)
	return result.Error
}
