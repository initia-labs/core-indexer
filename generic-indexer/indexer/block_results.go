package indexer

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/db"
	indexererrors "github.com/initia-labs/core-indexer/pkg/errors"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
	"github.com/initia-labs/core-indexer/pkg/txparser"
)

func (f *Indexer) produceTxResultMessage(txHash string, height int64, txResultJsonBytes []byte, logger *zerolog.Logger) {
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

func (f *Indexer) decodeAndInsertTxs(parentCtx context.Context, dbTx *gorm.DB, blockResults *mq.BlockResultMsg) (err error) {
	defer func() {
		// recover from panic if one occurred. Set err to nil otherwise.
		if recover() != nil {
			err = fmt.Errorf("%w: panic inside DecodeAndInsertTxs, possibly invalid message", indexererrors.ErrorNonRetryable)
		}
	}()

	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "decodeAndInsertTxs", "Decode txs in the block and insert them into DB")
	defer span.Finish()

	txs := make([]db.Transaction, 0)
	accounts := make(map[string]db.Account)
	accTxs := make([]db.AccountTransaction, 0)

	for i, txResult := range blockResults.Txs {
		if strings.Contains(txResult.ExecTxResults.Log, indexererrors.TxPareserError) || strings.Contains(txResult.ExecTxResults.Log, indexererrors.TxParseErrorV2) {
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

		accs, accTx, err := f.getAccountTransactions(txResult.Hash, blockResults.Height, txResult.ExecTxResults.Events, txData.Sender)
		if err != nil {
			return fmt.Errorf("error processing account transaction: %v", err)
		}
		maps.Copy(accounts, accs)
		accTxs = append(accTxs, accTx...)

		txs = append(txs, *txData)
		f.produceTxResultMessage(txResult.Hash, blockResults.Height, txResultJsonByte, logger)
	}

	if err := db.InsertVMAddressesAndAccountsIgnoreConflict(ctx, dbTx, accounts); err != nil {
		return fmt.Errorf("error inserting vm addresses and accounts: %v", err)
	}

	if err = db.InsertTransactionIgnoreConflict(ctx, dbTx, txs); err != nil {
		return fmt.Errorf("error inserting transactions: %v", err)
	}

	if err = db.InsertAccountTxsIgnoreConflict(ctx, dbTx, accTxs); err != nil {
		return fmt.Errorf("error inserting account transactions: %v", err)
	}
	return nil
}

func (f *Indexer) processBlockResults(parentCtx context.Context, blockResults *mq.BlockResultMsg) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "processBlock", "Parse Block and insert blocks & transactions into DB")
	defer span.Finish()

	logger.Info().Msgf("Processing block: %d", blockResults.Height)
	if err := f.dbClient.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		hashBytes, err := hex.DecodeString(blockResults.Hash)
		if err != nil {
			return indexererrors.ErrorNonRetryable
		}

		proposer, err := db.QueryValidatorAddress(ctx, dbTx, blockResults.ProposerConsensusAddress)
		if err != nil {
			logger.Error().Int64("height", blockResults.Height).Msgf("Error querying validator address: %v", err)
			return err
		}

		err = db.InsertBlockIgnoreConflict(ctx, dbTx, db.Block{
			Height:    blockResults.Height,
			Hash:      hashBytes,
			Proposer:  proposer,
			Timestamp: blockResults.Timestamp,
		})
		if err != nil {
			logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting block: %v", err)
			return err
		}

		err = f.decodeAndInsertTxs(ctx, dbTx, blockResults)
		if err != nil {
			logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting transactions: %v", err)
			return errors.Join(indexererrors.ErrorNonRetryable, err)
		}
		return nil
	}); err != nil {
		logger.Error().Int64("height", blockResults.Height).Msgf("Error processing block: %v", err)
		return errors.Join(indexererrors.ErrorNonRetryable, err)
	}

	logger.Info().Int64("height", blockResults.Height).Msgf("Successfully indexed block: %d", blockResults.Height)

	return nil
}
