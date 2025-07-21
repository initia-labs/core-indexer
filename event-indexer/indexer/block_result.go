package indexer

import (
	"context"
	"errors"
	"fmt"

	movetypes "github.com/initia-labs/initia/x/move/types"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/db"
	indexererror "github.com/initia-labs/core-indexer/pkg/indexer-error"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
)

func (f *Indexer) parseAndInsertMoveEvents(parentCtx context.Context, dbTx *gorm.DB, blockResults *mq.BlockResultMsg) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "parseAndInsertMoveEvents", "Parse block_results message and insert move_events into the database")
	defer span.Finish()

	moveEvents := make([]*db.MoveEvent, 0)
	for _, tx := range blockResults.Txs {
		if tx.ExecTxResults.Log == indexererror.TxParseError {
			continue
		}

		// idx ensures EventIndex is unique within each transaction.
		idx := 0
		for _, event := range tx.ExecTxResults.Events {
			if event.Type == movetypes.EventTypeMove {
				moveEvent := &db.MoveEvent{
					BlockHeight:     blockResults.Height,
					TransactionHash: tx.Hash,
					EventIndex:      idx,
				}
				for _, attr := range event.Attributes {
					switch attr.Key {
					case movetypes.AttributeKeyTypeTag:
						moveEvent.TypeTag = attr.Value
					case movetypes.AttributeKeyData:
						moveEvent.Data = []byte(attr.Value)
					}
				}
				moveEvents = append(moveEvents, moveEvent)
				idx++
			}
		}
	}

	if err := db.InsertMoveEventsIgnoreConflict(ctx, dbTx, moveEvents); err != nil {
		logger.Error().Msgf("Error inserting move_events: %v", err)
		return err
	}

	return nil
}

func (f *Indexer) parseAndInsertFinalizeBlockEvents(parentCtx context.Context, dbTx *gorm.DB, blockResults *mq.BlockResultMsg) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "parseAndInsertFinalizeBlockEvents", "Parse block_results message and insert finalize_block_events into the database")
	defer span.Finish()

	finalizeBlockEvents := make([]*db.FinalizeBlockEvent, 0)
	// id ensures EventIndex is unique within each block.
	idx := 0
	for _, event := range blockResults.FinalizeBlockEvents {
		// Check if the last attribute key is "mode"
		// We set mode to the value of the last attribute key "mode" here; since in CometBFT the "mode" will get
		// append at the end.
		// There is a case, however, where this may not apply. Ones being messages executed in the upgrade handler.
		// Optimistically, since events emitted from the upgrade handler do not have the mode attributes appended,
		// they also won't be passed to the function `sdk.MarkEventsToIndex`.
		// Thus, we can safely assume they do not belong to the finalized block events set.

		attrs := event.Attributes
		if len(attrs) == 0 {
			continue
		}

		// Check if at least one attribute has index: true
		hasIndexedAttr := false
		for _, attr := range attrs {
			if attr.Index {
				hasIndexedAttr = true
				break
			}
		}

		// If there are indexed attributes, enforce the "mode" key rule
		if hasIndexedAttr {
			if attrs[len(attrs)-1].Key != "mode" {
				err := fmt.Errorf("expected 'mode' as the last attribute in event, but found '%s'", attrs[len(attrs)-1].Key)
				logger.Error().Msgf("%v", err)
				return err
			}

			mode := attrs[len(attrs)-1].Value

			// Process all attributes except the last one (which should be "mode")
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
		}
	}

	if err := db.InsertFinalizeBlockEventsIgnoreConflict(ctx, dbTx, finalizeBlockEvents); err != nil {
		logger.Error().Msgf("Error inserting finalize_block_events: %v", err)
		return err
	}

	return nil
}

func (f *Indexer) processBlockResults(parentCtx context.Context, blockResults *mq.BlockResultMsg) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "processBlockResults", "Parse block_results message and insert tx events into the database")
	defer span.Finish()

	logger.Info().Msgf("Processing block_results at height: %d", blockResults.Height)

	if err := f.dbClient.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		if err := f.parseAndInsertMoveEvents(ctx, dbTx, blockResults); err != nil {
			logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting move_events: %v", err)
			return err
		}

		if err := f.parseAndInsertFinalizeBlockEvents(ctx, dbTx, blockResults); err != nil {
			logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting finalize_block_events: %v", err)
			return err
		}
		return nil
	}); err != nil {
		logger.Error().Int64("height", blockResults.Height).Msgf("Error processing block: %v", err)
		return errors.Join(indexererror.ErrorNonRetryable, err)
	}

	logger.Info().Int64("height", blockResults.Height).Msgf("Successfully flushed block: %d", blockResults.Height)

	return nil
}
