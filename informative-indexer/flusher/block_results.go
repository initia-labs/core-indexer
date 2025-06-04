package flusher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
	"github.com/initia-labs/core-indexer/pkg/txparser"
)

func (f *Flusher) parseAndInsertTransactionEvents(parentCtx context.Context, dbTx *gorm.DB, blockResults *mq.BlockResultMsg) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "parseAndInsertTransactionEvents", "Parse block_results message and insert transaction_events into the database")
	defer span.Finish()

	f.dbBatchInsert = NewDBBatchInsert()
	txs := make([]*db.Transaction, 0)
	txEvents := make([]*db.TransactionEvent, 0)
	for i, txResult := range blockResults.Txs {
		if txResult.ExecTxResults.Log == "tx parse error" {
			continue
		}

		tx, err := f.encodingConfig.TxConfig.TxDecoder()(txResult.Tx)
		if err != nil {
			return errors.Join(ErrorNonRetryable, err)
		}
		feeTx, ok := tx.(types.FeeTx)
		if !ok {
			return errors.Join(ErrorNonRetryable, err)
		}
		memoTx, ok := tx.(types.TxWithMemo)
		if !ok {
			return errors.Join(ErrorNonRetryable, err)
		}

		var errMsg *string
		if !txResult.ExecTxResults.IsOK() {
			escapedErrMsg := strings.ReplaceAll(txResult.ExecTxResults.Log, "\x00", "\uFFFD")
			errMsg = &escapedErrMsg
		}

		txResultJsonDict, protoTx, err := txparser.GetTxResponse(f.encodingConfig, blockResults.Timestamp, coretypes.ResultTx{
			Hash:     txResult.Tx.Hash(),
			Height:   blockResults.Height,
			Index:    uint32(i),
			TxResult: *txResult.ExecTxResults,
			Tx:       txResult.Tx,
		})
		if err != nil {
			return errors.Join(ErrorNonRetryable, err)
		}

		signers, _, err := protoTx.GetSigners(f.encodingConfig.Codec)
		if err != nil {
			return errors.Join(ErrorNonRetryable, err)
		}
		addr := types.AccAddress(signers[0])
		msgs, err := txparser.ParseMessageDicts(txResultJsonDict)
		if err != nil {
			return errors.Join(ErrorNonRetryable, err)
		}

		messagesJSON, err := json.Marshal(msgs)
		if err != nil {
			return errors.Join(ErrorNonRetryable, err)
		}

		txs = append(txs, &db.Transaction{
			ID:          db.GetTxID(txResult.Hash, blockResults.Height),
			Hash:        []byte(txResult.Hash),
			BlockHeight: blockResults.Height,
			BlockIndex:  i,
			GasUsed:     txResult.ExecTxResults.GasUsed,
			GasLimit:    int64(feeTx.GetGas()),
			GasFee:      feeTx.GetFee().String(),
			ErrMsg:      errMsg,
			Success:     txResult.ExecTxResults.IsOK(),
			Sender:      addr.String(),
			Memo:        strings.ReplaceAll(memoTx.GetMemo(), "\x00", "\uFFFD"),
			Messages:    messagesJSON,
		})

		// idx ensures EventIndex is unique within each transaction.
		idx := 0
		for _, event := range txResult.ExecTxResults.Events {
			for _, attr := range event.Attributes {
				txEvents = append(txEvents, &db.TransactionEvent{
					TransactionHash: txResult.Hash,
					BlockHeight:     blockResults.Height,
					EventKey:        fmt.Sprintf("%s.%s", event.Type, attr.Key),
					EventValue:      attr.Value,
					EventIndex:      idx,
				})
				idx++
			}
		}
	}

	if err := db.InsertTransactionIgnoreConflict(ctx, dbTx, txs); err != nil {
		logger.Error().Msgf("Error inserting transactions: %v", err)
		return err
	}

	if err := db.InsertTransactionEventsIgnoreConflict(ctx, dbTx, txEvents); err != nil {
		logger.Error().Msgf("Error inserting transaction_events: %v", err)
		return err
	}

	// Process events
	if err := f.processEvents(blockResults); err != nil {
		logger.Error().Msgf("Error processing events: %v", err)
		return errors.Join(ErrorNonRetryable, err)
	}

	// Sync all data with the chain state
	if err := f.syncValidatorData(ctx); err != nil {
		logger.Error().Msgf("Error syncing validator data: %v", err)
		return err
	}

	// After sync data, flush the batch insert
	if err := f.dbBatchInsert.Flush(ctx, dbTx); err != nil {
		logger.Error().Msgf("Error flushing batch insert: %v", err)
		return err
	}

	return nil
}

func (f *Flusher) processEvents(blockResults *mq.BlockResultMsg) error {
	f.blockStateUpdates = NewBlockStateUpdates()
	if err := f.processAccounts(blockResults); err != nil {
		logger.Error().Msgf("Error processing related accounts: %v", err)
		return err
	}
	if err := f.processValidatorEvents(blockResults); err != nil {
		logger.Error().Msgf("Error processing validator events: %v", err)
		return err
	}

	return nil
}

func (f *Flusher) parseAndInsertMoveEvents(parentCtx context.Context, dbTx *gorm.DB, blockResults *mq.BlockResultMsg) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "parseAndInsertMoveEvents", "Parse block_results message and insert move_events into the database")
	defer span.Finish()

	moveEvents := make([]*db.MoveEvent, 0)
	for _, tx := range blockResults.Txs {
		if tx.ExecTxResults.Log == "tx parse error" {
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

func (f *Flusher) parseAndInsertFinalizeBlockEvents(parentCtx context.Context, dbTx *gorm.DB, blockResults *mq.BlockResultMsg) error {
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

func (f *Flusher) processBlockResults(parentCtx context.Context, blockResults *mq.BlockResultMsg) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "processBlockResults", "Parse block_results message and insert tx events into the database")
	defer span.Finish()

	logger.Info().Msgf("Processing block_results at height: %d", blockResults.Height)

	if err := f.dbClient.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		if err := f.parseAndInsertTransactionEvents(ctx, dbTx, blockResults); err != nil {
			logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting transaction_events: %v", err)
			return err
		}

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
		return err
	}

	logger.Info().Int64("height", blockResults.Height).Msgf("Successfully flushed block: %d", blockResults.Height)

	return nil
}
