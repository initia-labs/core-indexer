package indexer

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

	statetracker "github.com/initia-labs/core-indexer/informative-indexer/indexer/state-tracker"
	"github.com/initia-labs/core-indexer/informative-indexer/indexer/types"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
	"github.com/initia-labs/core-indexer/pkg/txparser"
)

func (f *Indexer) parseAndInsertBlock(parentCtx context.Context, dbTx *gorm.DB, blockResults *mq.BlockResultMsg, proposer *db.ValidatorAddress) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "parseAndInsertBlock", "Parse block_results message and insert block into the database")
	defer span.Finish()

	hashBytes, err := hex.DecodeString(blockResults.Hash)
	if err != nil {
		return types.ErrorNonRetryable
	}

	err = db.UpsertBlock(ctx, dbTx, db.Block{
		Height:    blockResults.Height,
		Hash:      hashBytes,
		Proposer:  &proposer.OperatorAddress,
		Timestamp: blockResults.Timestamp,
	})
	if err != nil {
		return err
	}

	return err
}

func (f *Indexer) processTransactions(parentCtx context.Context, blockResults *mq.BlockResultMsg) error {
	span, _ := sentry_integration.StartSentrySpan(parentCtx, "processTransactions", "Parse block_results message and insert transaction_events into the database")
	defer span.Finish()

	for i, txResult := range blockResults.Txs {
		if txResult.ExecTxResults.Log == types.TxParseError {
			continue
		}

		tx, err := f.encodingConfig.TxConfig.TxDecoder()(txResult.Tx)
		if err != nil {
			return errors.Join(types.ErrorNonRetryable, fmt.Errorf("failed to decode SDK transaction: %w", err))
		}
		feeTx, ok := tx.(sdk.FeeTx)
		if !ok {
			return errors.Join(types.ErrorNonRetryable, fmt.Errorf("failed to cast SDK transaction to FeeTx: %w", err))
		}
		memoTx, ok := tx.(sdk.TxWithMemo)
		if !ok {
			return errors.Join(types.ErrorNonRetryable, fmt.Errorf("failed to cast SDK transaction to TxWithMemo: %w", err))
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
			return errors.Join(types.ErrorNonRetryable, fmt.Errorf("failed to get signers: %w", err))
		}

		signers, _, err := protoTx.GetSigners(f.encodingConfig.Codec)
		if err != nil {
			return errors.Join(types.ErrorNonRetryable, fmt.Errorf("failed to get signers: %w", err))
		}
		addr := sdk.AccAddress(signers[0])
		msgs, err := txparser.ParseMessageDicts(txResultJsonDict)
		if err != nil {
			return errors.Join(types.ErrorNonRetryable, fmt.Errorf("failed to parse message dicts: %w", err))
		}

		messagesJSON, err := json.Marshal(msgs)
		if err != nil {
			return errors.Join(types.ErrorNonRetryable, fmt.Errorf("failed to marshal messages: %w", err))
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

		for _, processor := range f.processors {
			processor.NewTxProcessor(txData)
			if err := processor.ProcessSDKMessages(&txResult, f.encodingConfig); err != nil {
				logger.Error().Msgf("Error processing %s sdk messages: %v", processor.Name(), err)
				return errors.Join(types.ErrorNonRetryable, fmt.Errorf("failed to process %s sdk messages: %w", processor.Name(), err))
			}
			if err := processor.ProcessTransactionEvents(&txResult); err != nil {
				logger.Error().Msgf("Error processing %s tx events: %v", processor.Name(), err)
				return errors.Join(types.ErrorNonRetryable, fmt.Errorf("failed to process %s tx events: %w", processor.Name(), err))
			}
			if err := processor.ResolveTxProcessor(); err != nil {
				logger.Error().Msgf("Error resolving %s tx: %v", processor.Name(), err)
				return errors.Join(types.ErrorNonRetryable, fmt.Errorf("failed to resolve %s tx: %w", processor.Name(), err))
			}
		}
		f.dbBatchInsert.AddTransaction(*txData)
	}

	return nil
}

func (f *Indexer) processBlockResults(parentCtx context.Context, blockResults *mq.BlockResultMsg, proposer *db.ValidatorAddress) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "processBlockResults", "Parse block_results message and insert tx events into the database")
	defer span.Finish()

	logger.Info().Msgf("Processing block_results at height: %d", blockResults.Height)

	if err := f.dbClient.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		if err := f.parseAndInsertBlock(ctx, dbTx, blockResults, proposer); err != nil {
			logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting block: %v", err)
			return err
		}

		f.dbBatchInsert = statetracker.NewDBBatchInsert(f.cacher, logger)
		f.stateUpdateManager = statetracker.NewStateUpdateManager(f.dbBatchInsert, f.encodingConfig, &blockResults.Height)

		for _, processor := range f.processors {
			processor.InitProcessor(blockResults.Height, f.cacher)

			if err := processor.ProcessBeginBlockEvents(&blockResults.FinalizeBlockEvents); err != nil {
				logger.Error().Msgf("Error processing %s messages: %v", processor.Name(), err)
				return errors.Join(types.ErrorNonRetryable, err)
			}
		}

		if err := f.processTransactions(ctx, blockResults); err != nil {
			logger.Error().Int64("height", blockResults.Height).Msgf("Error processing transactions: %v", err)
			return err
		}

		for _, processor := range f.processors {
			if err := processor.ProcessEndBlockEvents(&blockResults.FinalizeBlockEvents); err != nil {
				logger.Error().Msgf("Error processing %s messages: %v", processor.Name(), err)
				return errors.Join(types.ErrorNonRetryable, err)
			}

			if err := processor.TrackState(f.stateUpdateManager, f.dbBatchInsert); err != nil {
				logger.Error().Msgf("Error tracking state %s: %v", processor.Name(), err)
				return errors.Join(types.ErrorNonRetryable, err)
			}
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

		// TODO: separate service for this
		// if err := f.parseAndInsertMoveEvents(ctx, dbTx, blockResults); err != nil {
		// 	logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting move_events: %v", err)
		// 	return err
		// }

		// if err := f.parseAndInsertFinalizeBlockEvents(ctx, dbTx, blockResults); err != nil {
		// 	logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting finalize_block_events: %v", err)
		// 	return err
		// }
		return nil
	}); err != nil {
		logger.Error().Int64("height", blockResults.Height).Msgf("Error processing block: %v", err)
		return errors.Join(types.ErrorNonRetryable, err)
	}

	logger.Info().Int64("height", blockResults.Height).Msgf("Successfully indexed block: %d", blockResults.Height)

	return nil
}
