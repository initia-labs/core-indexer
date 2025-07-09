package flusher

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"gorm.io/gorm"

	stateTracker "github.com/initia-labs/core-indexer/informative-indexer/flusher/state-tracker"
	"github.com/initia-labs/core-indexer/informative-indexer/flusher/types"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
	"github.com/initia-labs/core-indexer/pkg/txparser"
	movetypes "github.com/initia-labs/initia/x/move/types"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

func (f *Flusher) parseAndInsertBlock(parentCtx context.Context, dbTx *gorm.DB, blockResults *mq.BlockResultMsg, proposer *mstakingtypes.Validator) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "parseAndInsertBlock", "Parse block_results message and insert block into the database")
	defer span.Finish()

	hashBytes, err := hex.DecodeString(blockResults.Hash)
	if err != nil {
		return types.ErrorNonRetryable
	}

	err = db.InsertBlockIgnoreConflict(ctx, dbTx, db.Block{
		Height:    int32(blockResults.Height),
		Hash:      hashBytes,
		Proposer:  proposer.OperatorAddress,
		Timestamp: blockResults.Timestamp,
	})
	if err != nil {
		return err
	}

	return err
}

func (f *Flusher) parseAndInsertTransactionEvents(parentCtx context.Context, blockResults *mq.BlockResultMsg) error {
	span, _ := sentry_integration.StartSentrySpan(parentCtx, "parseAndInsertTransactionEvents", "Parse block_results message and insert transaction_events into the database")
	defer span.Finish()

	for i, txResult := range blockResults.Txs {
		if txResult.ExecTxResults.Log == types.TxParseError {
			continue
		}

		tx, err := f.encodingConfig.TxConfig.TxDecoder()(txResult.Tx)
		if err != nil {
			return errors.Join(types.ErrorNonRetryable, err)
		}
		feeTx, ok := tx.(sdk.FeeTx)
		if !ok {
			return errors.Join(types.ErrorNonRetryable, err)
		}
		memoTx, ok := tx.(sdk.TxWithMemo)
		if !ok {
			return errors.Join(types.ErrorNonRetryable, err)
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
			return errors.Join(types.ErrorNonRetryable, err)
		}

		signers, _, err := protoTx.GetSigners(f.encodingConfig.Codec)
		if err != nil {
			return errors.Join(types.ErrorNonRetryable, err)
		}
		addr := sdk.AccAddress(signers[0])
		msgs, err := txparser.ParseMessageDicts(txResultJsonDict)
		if err != nil {
			return errors.Join(types.ErrorNonRetryable, err)
		}

		messagesJSON, err := json.Marshal(msgs)
		if err != nil {
			return errors.Join(types.ErrorNonRetryable, err)
		}

		txData := &db.Transaction{
			ID:          db.GetTxID(txResult.Hash, blockResults.Height),
			Hash:        txResult.Tx.Hash(),
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
		}

		// idx ensures EventIndex is unique within each transaction.
		idx := 0
		for _, event := range txResult.ExecTxResults.Events {
			for _, attr := range event.Attributes {
				f.dbBatchInsert.AddTransactionEvent(db.TransactionEvent{
					TransactionHash: txResult.Hash,
					BlockHeight:     blockResults.Height,
					EventKey:        fmt.Sprintf("%s.%s", event.Type, attr.Key),
					EventValue:      attr.Value,
					EventIndex:      idx,
				})
				idx++
			}
		}

		// TODO: replace with processor
		// Process events
		if err := f.processEvents(&txResult, blockResults.Height, txData); err != nil {
			logger.Error().Msgf("Error processing events: %v", err)
			return errors.Join(types.ErrorNonRetryable, err)
		}

		for _, processor := range f.processors {
			processor.InitProcessor()
		}
		for _, processor := range f.processors {
			processor.InitProcessor()
			if err := processor.ProcessTxEvents(&txResult, blockResults.Height, f.stateUpdateManager, txData); err != nil {
				logger.Error().Msgf("Error processing %s events: %v", processor.Name(), err)
				return errors.Join(types.ErrorNonRetryable, err)
			}
		}
		f.dbBatchInsert.AddTransaction(*txData)
	}

	return nil
}

func (f *Flusher) processEvents(txResult *mq.TxResult, height int64, txData *db.Transaction) error {
	if err := f.processAccounts(txResult, height, txData); err != nil {
		logger.Error().Msgf("Error processing related accounts: %v", err)
		return err
	}
	if err := f.processValidatorEvents(txResult, height, txData); err != nil {
		logger.Error().Msgf("Error processing validator events: %v", err)
		return err
	}

	if err := f.processProposalEvents(txResult, height, txData); err != nil {
		logger.Error().Msgf("Error processing proposal events: %v", err)
		return err
	}

	if err := f.processMoveEvents(txResult, height, txData); err != nil {
		logger.Error().Msgf("Error processing move events: %v", err)
		return err
	}

	if err := f.processIbcEvents(txResult, height, txData); err != nil {
		logger.Error().Msgf("Error processing ibc events: %v", err)
		return err
	}

	if err := f.processOpinitEvents(txResult, height, txData); err != nil {
		logger.Error().Msgf("Error processing opinit events: %v", err)
		return err
	}
	return nil
}

func (f *Flusher) parseAndInsertTransactionEndBlockEvents(parentCtx context.Context, blockResults *mq.BlockResultMsg) error {
	span, _ := sentry_integration.StartSentrySpan(parentCtx, "parseAndInsertTransactionEndBlockEvents", "Parse block_results message and insert transaction_end_block_events into the database")
	defer span.Finish()

	// TODO: filter only endblock events first
	if err := f.processProposalEndBlockEvents(blockResults); err != nil {
		logger.Error().Msgf("Error processing end block events: %v", err)
		return err
	}

	return nil
}

func (f *Flusher) parseAndInsertMoveEvents(parentCtx context.Context, dbTx *gorm.DB, blockResults *mq.BlockResultMsg) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "parseAndInsertMoveEvents", "Parse block_results message and insert move_events into the database")
	defer span.Finish()

	moveEvents := make([]*db.MoveEvent, 0)
	for _, tx := range blockResults.Txs {
		if tx.ExecTxResults.Log == types.TxParseError {
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

func (f *Flusher) processBlockResults(parentCtx context.Context, blockResults *mq.BlockResultMsg, proposer *mstakingtypes.Validator) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "processBlockResults", "Parse block_results message and insert tx events into the database")
	defer span.Finish()

	logger.Info().Msgf("Processing block_results at height: %d", blockResults.Height)

	if err := f.dbClient.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		if err := f.parseAndInsertBlock(ctx, dbTx, blockResults, proposer); err != nil {
			logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting block: %v", err)
			return err
		}

		f.dbBatchInsert = stateTracker.NewDBBatchInsert(logger)
		f.stateUpdateManager = stateTracker.NewStateUpdateManager(f.dbBatchInsert, f.encodingConfig, &blockResults.Height)

		if err := f.parseAndInsertTransactionEvents(ctx, blockResults); err != nil {
			logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting transaction_events: %v", err)
			return err
		}

		if err := f.parseAndInsertTransactionEndBlockEvents(ctx, blockResults); err != nil {
			logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting finalize_block_events: %v", err)
			return err
		}

		if err := f.stateUpdateManager.UpdateState(ctx, f.rpcClient); err != nil {
			logger.Error().Msgf("Error updating state: %v", err)
			return err
		}
		// After sync data, flush the batch insert
		if err := f.dbBatchInsert.Flush(ctx, dbTx, blockResults.Height); err != nil {
			logger.Error().Msgf("Error flushing batch insert: %v", err)
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
		return errors.Join(types.ErrorNonRetryable, err)
	}

	logger.Info().Int64("height", blockResults.Height).Msgf("Successfully flushed block: %d", blockResults.Height)

	return nil
}
