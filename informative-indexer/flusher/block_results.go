package flusher

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/initia-labs/core-indexer/informative-indexer/common"
	"github.com/initia-labs/core-indexer/informative-indexer/db"
)

func (f *Flusher) parseAndInsertTransactionEvents(parentCtx context.Context, dbTx pgx.Tx, blockResults *common.BlockResultMsg) error {
	span, ctx := common.StartSentrySpan(parentCtx, "parseAndInsertTransactionEvents", "Parse block_results message and insert transaction_events into the database")
	defer span.Finish()

	txEvents := make([]*db.TransactionEvent, 0)
	for _, tx := range blockResults.Txs {
		if tx.ExecTxResults.Log == "tx parse error" {
			continue
		}

		// idx ensures EventIndex is unique within each transaction.
		idx := 0
		for _, event := range tx.ExecTxResults.Events {
			for _, attr := range event.Attributes {
				txEvents = append(txEvents, &db.TransactionEvent{
					TransactionHash: tx.Hash,
					BlockHeight:     blockResults.Height,
					EventKey:        fmt.Sprintf("%s.%s", event.Type, attr.Key),
					EventValue:      attr.Value,
					EventIndex:      idx,
				})
				idx++
			}
		}
	}

	if err := db.InsertTransactionEventsIgnoreConflict(ctx, dbTx, txEvents); err != nil {
		logger.Error().Msgf("Error inserting transaction_events: %v", err)
		return err
	}

	return nil
}

func (f *Flusher) parseAndInsertFinalizeBlockEvents(parentCtx context.Context, dbTx pgx.Tx, blockResults *common.BlockResultMsg) error {
	span, ctx := common.StartSentrySpan(parentCtx, "parseAndInsertFinalizeBlockEvents", "Parse block_results message and insert finalize_block_events into the database")
	defer span.Finish()

	finalizeBlockEvents := make([]*db.FinalizeBlockEvent, 0)
	// id ensures EventIndex is unique within each block.
	idx := 0
	for _, event := range blockResults.FinalizeBlockEvents {
		attrs := event.Attributes
		if len(attrs) > 0 {
			// Check if the last attribute key is "mode"
			// We set mode to the value of the last attribute key "mode" here, since in CometBFT the "mode" will get
			// append at the end
			if attrs[len(attrs)-1].Key == "mode" {
				mode, err := db.ParseMode(attrs[len(attrs)-1].Value)
				if err != nil {
					logger.Error().Msgf("Error parsing `mode` into db.Mode: %v", err)
					return err
				}
				// Process all attributes except the last one (which should be the mode)
				for _, attr := range attrs[:len(attrs)-1] {
					finalizeBlockEvents = append(finalizeBlockEvents, &db.FinalizeBlockEvent{
						BlockHeight: blockResults.Height,
						EventKey:    fmt.Sprintf("%s.%s", event.Type, attr.Key),
						EventValue:  attr.Value,
						EventIndex:  idx,
						Mode:        mode,
					})
					idx++
				}
			} else {
				err := fmt.Errorf("expected 'mode' as the last attribute in event, but found '%s'", attrs[len(attrs)-1].Key)
				logger.Error().Msgf("%v", err)
				return err
			}
		}
	}

	if err := db.InsertFinalizeBlockEventsIgnoreConflict(ctx, dbTx, finalizeBlockEvents); err != nil {
		logger.Error().Msgf("Error inserting finalize_block_events: %v", err)
		return err
	}

	return nil
}

func (f *Flusher) processBlockResults(parentCtx context.Context, blockResults *common.BlockResultMsg) error {
	span, ctx := common.StartSentrySpan(parentCtx, "processBlockResults", "Parse block_results message and insert tx events into the database")
	defer span.Finish()

	logger.Info().Msgf("Processing block_results at height: %d", blockResults.Height)

	dbTx, err := f.dbClient.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		logger.Error().Int64("height", blockResults.Height).Msgf("Error beginning transaction: %v", err)
		return err
	}
	defer dbTx.Rollback(ctx)

	err = f.parseAndInsertTransactionEvents(ctx, dbTx, blockResults)
	if err != nil {
		logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting transaction_events: %v", err)
		return err
	}

	err = f.parseAndInsertFinalizeBlockEvents(ctx, dbTx, blockResults)
	if err != nil {
		logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting finalize_block_events: %v", err)
		return err
	}

	err = dbTx.Commit(ctx)
	if err != nil {
		logger.Error().Int64("height", blockResults.Height).Msgf("Error committing transaction: %v", err)
		return err
	}

	logger.Info().Int64("height", blockResults.Height).Msgf("Successfully flushed block: %d", blockResults.Height)

	return nil
}
