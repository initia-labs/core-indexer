package indexer

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	statetracker "github.com/initia-labs/core-indexer/informative-indexer/indexer/state-tracker"
	"github.com/initia-labs/core-indexer/pkg/db"
	indexererrors "github.com/initia-labs/core-indexer/pkg/errors"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
	"github.com/initia-labs/core-indexer/pkg/txparser"
)

func (f *Indexer) parseAndInsertBlock(parentCtx context.Context, dbTx *gorm.DB, blockResults *mq.BlockResultMsg, proposer *db.ValidatorAddress) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "parseAndInsertBlock", "Parse block_results message and insert block into the database")
	defer span.Finish()

	hashBytes, err := hex.DecodeString(blockResults.Hash)
	if err != nil {
		return indexererrors.ErrorNonRetryable
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

func (f *Indexer) produceTxResponseMessage(txHash string, height int64, txResultJsonBytes []byte, logger *zerolog.Logger) {
	messageKey := mq.NEW_LCD_TX_RESPONSE_KAFKA_MESSAGE_KEY + fmt.Sprintf("_%s", txHash)
	kafkaMessage := kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &f.config.KafkaTxResponseTopic, Partition: int32(kafka.PartitionAny)},
		Headers:        []kafka.Header{{Key: "height", Value: fmt.Append(nil, height)}, {Key: "tx_hash", Value: fmt.Append(nil, txHash)}},
		Key:            []byte(messageKey),
		Value:          txResultJsonBytes,
	}

	f.producer.ProduceWithClaimCheck(&mq.ProduceWithClaimCheckInput{
		Topic:          f.config.KafkaTxResponseTopic,
		Key:            kafkaMessage.Key,
		MessageInBytes: kafkaMessage.Value,

		ClaimCheckKey:           []byte(mq.NEW_LCD_TX_RESPONSE_CLAIM_CHECK_KAFKA_MESSAGE_KEY + fmt.Sprintf("_%s", txHash)),
		ClaimCheckThresholdInMB: f.config.ClaimCheckThresholdInMB,
		ClaimCheckBucket:        f.config.LCDTxResponseClaimCheckBucket,
		ClaimCheckObjectPath:    db.GetTxID(txHash, height),

		StorageClient: f.storageClient,

		Headers: kafkaMessage.Headers,
	}, logger)
}

func (f *Indexer) processTransactions(parentCtx context.Context, blockResults *mq.BlockResultMsg) error {
	span, _ := sentry_integration.StartSentrySpan(parentCtx, "processTransactions", "Parse block_results message and insert transaction_events into the database")
	defer span.Finish()

	for i, txResult := range blockResults.Txs {
		if strings.Contains(txResult.ExecTxResults.Log, indexererrors.TxPareserError) || strings.Contains(txResult.ExecTxResults.Log, indexererrors.TxPareserErrorV2) {
			continue
		}

		txResultJsonDict, txResultJsonByte, err := txparser.GetTxResponse(f.encodingConfig, i, &txResult, blockResults)
		if err != nil {
			return fmt.Errorf("failed to get tx response: %w", err)
		}
		txData, err := txparser.ParseTransaction(f.encodingConfig, i, &txResult, txResultJsonDict, blockResults)
		if err != nil {
			return fmt.Errorf("failed to parse DB transaction: %v", err)
		}

		for _, processor := range f.processors {
			processor.NewTxProcessor(txData)
			if err := processor.ProcessSDKMessages(&txResult, f.encodingConfig); err != nil {
				return fmt.Errorf("failed to process %s sdk messages: %w", processor.Name(), err)
			}
			if err := processor.ProcessTransactionEvents(&txResult); err != nil {
				return fmt.Errorf("failed to process %s tx events: %w", processor.Name(), err)
			}
			if err := processor.ResolveTxProcessor(); err != nil {
				return fmt.Errorf("failed to resolve %s tx: %w", processor.Name(), err)
			}
		}
		f.dbBatchInsert.AddTransaction(*txData)

		f.produceTxResponseMessage(txResult.Hash, blockResults.Height, txResultJsonByte, logger)
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
				return err
			}
		}

		if err := f.processTransactions(ctx, blockResults); err != nil {
			logger.Error().Int64("height", blockResults.Height).Msgf("Error processing transactions: %v", err)
			return err
		}

		for _, processor := range f.processors {
			if err := processor.ProcessEndBlockEvents(&blockResults.FinalizeBlockEvents); err != nil {
				logger.Error().Msgf("Error processing %s messages: %v", processor.Name(), err)
				return err
			}

			if err := processor.TrackState(f.stateUpdateManager, f.dbBatchInsert); err != nil {
				logger.Error().Msgf("Error tracking state %s: %v", processor.Name(), err)
				return err
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
		return nil
	}); err != nil {
		logger.Error().Int64("height", blockResults.Height).Msgf("Error processing block: %v", err)
		return errors.Join(indexererrors.ErrorNonRetryable, err)
	}

	logger.Info().Int64("height", blockResults.Height).Msgf("Successfully indexed block: %d", blockResults.Height)

	return nil
}
