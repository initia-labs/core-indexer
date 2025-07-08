package flusher

import (
	"context"
	"errors"
	"fmt"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/initia-labs/core-indexer/generic-indexer/common"
	"github.com/initia-labs/core-indexer/generic-indexer/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
)

func (f *Flusher) produceTxResultMessage(txHash []byte, blockHeight int64, txResultJsonBytes []byte, logger *zerolog.Logger) {
	messageKey := mq.NEW_LCD_TX_RESPONSE_KAFKA_MESSAGE_KEY + fmt.Sprintf("_%X", txHash)
	kafkaMessage := kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &f.config.KafkaTxResponseTopic, Partition: int32(kafka.PartitionAny)},
		Headers:        []kafka.Header{{Key: "height", Value: []byte(fmt.Sprint(blockHeight))}, {Key: "tx_hash", Value: []byte(fmt.Sprintf("%X", txHash))}},
		Key:            []byte(messageKey),
		Value:          txResultJsonBytes,
	}

	f.producer.ProduceWithClaimCheck(&mq.ProduceWithClaimCheckInput{
		Topic:          f.config.KafkaTxResponseTopic,
		Key:            kafkaMessage.Key,
		MessageInBytes: kafkaMessage.Value,

		ClaimCheckKey:           []byte(mq.NEW_LCD_TX_RESPONSE_CLAIM_CHECK_KAFKA_MESSAGE_KEY + fmt.Sprintf("_%X", txHash)),
		ClaimCheckThresholdInMB: f.config.ClaimCheckThresholdInMB,
		ClaimCheckBucket:        f.config.LCDTxResponseClaimCheckBucket,
		ClaimCheckObjectPath:    fmt.Sprintf("%X/%d", txHash, blockHeight),

		StorageClient: f.storageClient,

		Headers: kafkaMessage.Headers,
	}, logger)
}

func (f *Flusher) getAccountTransaction(txHash []byte, height int64, events []abci.Event, signer sdk.AccAddress) (db.AccountTransaction, error) {
	relatedAccs, err := grepAddressesFromTx(events)
	if err != nil {
		logger.Error().Msgf("Error grep addresses from tx: %v", err)
		return db.AccountTransaction{}, err
	}

	accTx := db.AccountTransaction{
		TxId:        fmt.Sprintf("%X/%d", txHash, height),
		Accounts:    make([]string, 0),
		BlockHeight: height,
		Signer:      signer.String(),
	}

	// Add the signer as the first account
	accTx.Accounts = append(accTx.Accounts, signer.String())

	// Add related accounts, avoiding duplicates
	relatedAccTxs := make(map[string]bool)
	relatedAccTxs[signer.String()] = true
	for _, relatedAcc := range relatedAccs {
		if _, ok := relatedAccTxs[relatedAcc]; !ok {
			relatedAccTxs[relatedAcc] = false
			accTx.Accounts = append(accTx.Accounts, relatedAcc)
		}
	}

	return accTx, nil
}

func (f *Flusher) decodeAndInsertTxs(parentCtx context.Context, dbTx pgx.Tx, block *mq.BlockResultMsg, blockResults *coretypes.ResultBlockResults) (err error) {
	defer func() {
		// recover from panic if one occurred. Set err to nil otherwise.
		if recover() != nil {
			err = fmt.Errorf("%w: panic inside DecodeAndInsertTxs, possibly invalid message", ErrorNonRetryable)
		}
	}()

	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "decodeAndInsertTxs", "Decode txs in the block and insert them into DB")
	defer span.Finish()

	txs := make([]*common.Transaction, 0)
	accountInBlock := make(map[string]bool)
	accTxs := make([]db.AccountTransaction, 0)

	for idx, cosmosTx := range block.Txs {
		txResult := blockResults.TxsResults[idx]
		if txResult.Log == "tx parse error" {
			continue
		}

		tx, err := f.encodingConfig.TxConfig.TxDecoder()(cosmosTx.Tx)
		if err != nil {
			return errors.Join(ErrorNonRetryable, err)
		}

		txResultJsonDict, txResultJsonByte, protoTx := f.getTxResponse(block.Timestamp, cosmosTx.Tx, coretypes.ResultTx{
			Hash:     cosmosTx.Tx.Hash(),
			Height:   block.Height,
			Index:    uint32(idx),
			TxResult: *txResult,
			Tx:       cosmosTx.Tx,
		})

		md := getMessageDicts(txResultJsonDict)

		feeTx, ok := tx.(types.FeeTx)
		if !ok {
			return errors.Join(ErrorNonRetryable, err)
		}
		memoTx, ok := tx.(types.TxWithMemo)
		if !ok {
			return errors.Join(ErrorNonRetryable, err)
		}

		var errMsg *string
		if !txResult.IsOK() {
			escapedErrMsg := strings.ReplaceAll(txResult.Log, "\x00", "\uFFFD")
			errMsg = &escapedErrMsg
		}

		signers, _, err := protoTx.GetSigners(f.encodingConfig.Codec)
		if err != nil {
			return err
		}

		addr := types.AccAddress(signers[0])
		accountInBlock[addr.String()] = true
		txHash := cosmosTx.Tx.Hash()
		txs = append(txs, &common.Transaction{
			Hash:               txHash,
			BlockHeight:        block.Height,
			BlockIndex:         idx,
			GasUsed:            txResult.GasUsed,
			GasLimit:           feeTx.GetGas(),
			GasFee:             feeTx.GetFee().String(),
			ErrMsg:             errMsg,
			Success:            txResult.IsOK(),
			Sender:             addr.String(),
			Memo:               strings.ReplaceAll(memoTx.GetMemo(), "\x00", "\uFFFD"),
			Messages:           parseTxMessages(tx.GetMsgs(), md),
			IsIBC:              false,
			IsSend:             false,
			IsMovePublish:      false,
			IsMoveExecuteEvent: false,
			IsMoveExecute:      false,
			IsMoveUpgrade:      false,
			IsMoveScript:       false,
			IsNFTTransfer:      false,
			IsNFTMint:          false,
			IsNFTBurn:          false,
			IsCollectionCreate: false,
			IsOPInit:           false,
			IsInstantiate:      false,
			IsMigrate:          false,
			IsUpdateAdmin:      false,
			IsClearAdmin:       false,
			IsStoreCode:        false,
		})

		if !f.config.DisableLCDTXResponse {
			f.produceTxResultMessage(txHash, block.Height, txResultJsonByte, logger)
		}

		if !f.config.DisableIndexingAccountTransaction {
			accTx, err := f.getAccountTransaction(txHash, block.Height, txResult.Events, addr)
			if err != nil {
				logger.Error().Msgf("Error processing account transaction %v", err)
				return err
			}
			accTxs = append(accTxs, accTx)
			for _, acc := range accTx.Accounts {
				accountInBlock[acc] = true
			}
		}
	}

	accs := make([]string, 0)
	for acc := range accountInBlock {
		accs = append(accs, acc)
	}

	newAccs, err := db.GetAccountsIfNotExist(ctx, dbTx, accs)
	if err != nil {
		logger.Error().Msgf("Error get accounts if not exsit %v", err)
		return err

	}

	if err = db.InsertAccounts(ctx, dbTx, newAccs); err != nil {
		logger.Error().Msgf("Error inserting accounts %v", err)
		return err
	}

	if err = db.InsertTransactionsIgnoreConflict(ctx, dbTx, txs); err != nil {
		logger.Error().Msgf("Error inserting transactions %v", err)
		if err == db.ErrorNonRetryable || err == db.ErrorLengthMismatch {
			return errors.Join(ErrorNonRetryable, err)
		}
		return err
	}

	if !f.config.DisableIndexingAccountTransaction {
		if err = db.InsertAccountTransactions(ctx, dbTx, accTxs); err != nil {
			logger.Error().Msgf("Error inserting account transactions %v", err)
			return err
		}
	}
	return nil
}

func (f *Flusher) getBlockResults(parentCtx context.Context, height int64) (*coretypes.ResultBlockResults, error) {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "getBlockResults", "Calling /block_results from RPCs")
	defer span.Finish()

	var res *coretypes.ResultBlockResults
	var err error
	res, err = f.rpcClient.BlockResults(ctx, &height)
	if err == nil {
		return res, nil
	}

	return nil, err
}

func (f *Flusher) processBlock(parentCtx context.Context, block *mq.BlockResultMsg) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "processBlock", "Parse Block and insert blocks & transactions into DB")
	defer span.Finish()

	logger.Info().Msgf("Processing block: %d", block.Height)

	blockResults, err := f.getBlockResults(ctx, block.Height)
	if err != nil {
		logger.Error().Int64("height", block.Height).Msgf("Error getting block results: %v", err)
		return err
	}

	dbTx, err := f.dbClient.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		logger.Error().Int64("height", block.Height).Msgf("Error beginning transaction: %v", err)
		return err
	}
	defer dbTx.Rollback(ctx)
	proposer, _ := f.validators[block.ProposerConsensusAddress]
	err = db.InsertBlockIgnoreConflict(ctx, dbTx, block, &proposer.OperatorAddress)
	if err != nil {
		if err == db.ErrorNonRetryable {
			logger.Error().Int64("height", block.Height).Msgf("Cannot decode hex.DecodeString(block.Hash)")
			return errors.Join(ErrorNonRetryable, err)
		}
		logger.Error().Int64("height", block.Height).Msgf("Error inserting block: %v", err)
		return err
	}

	err = f.decodeAndInsertTxs(ctx, dbTx, block, blockResults)
	if err != nil {
		logger.Error().Int64("height", block.Height).Msgf("Error inserting transactions: %v", err)
		return err
	}

	err = dbTx.Commit(ctx)
	if err != nil {
		logger.Error().Int64("height", block.Height).Msgf("Error committing transaction: %v", err)
		return err
	}

	logger.Info().Int64("height", block.Height).Msgf("Successfully flushed block: %d", block.Height)

	return nil
}
