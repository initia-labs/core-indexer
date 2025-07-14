package indexer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	vmtypes "github.com/initia-labs/movevm/types"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
)

func (f *Indexer) produceTxResultMessage(txHash []byte, blockHeight int64, txResultJsonBytes []byte, logger *zerolog.Logger) {
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

func (f *Indexer) getAccountTransactions(txHash []byte, height int64, events []abci.Event, signer sdk.AccAddress) ([]db.AccountTransaction, error) {
	relatedAccs, err := grepAddressesFromTx(events)
	if err != nil {
		logger.Error().Msgf("Error grep addresses from tx: %v", err)
		return nil, err
	}

	mapAccs := make(map[string]bool)
	for _, acc := range relatedAccs {
		mapAccs[acc] = true
	}
	mapAccs[signer.String()] = true
	accTxs := make([]db.AccountTransaction, 0)
	for acc, isSigner := range mapAccs {
		accTx := db.AccountTransaction{
			TransactionID: fmt.Sprintf("%X/%d", txHash, height),
			AccountID:     acc,
			BlockHeight:   height,
			IsSigner:      isSigner,
		}
		accTxs = append(accTxs, accTx)
	}
	return accTxs, nil
}

func (f *Indexer) decodeAndInsertTxs(parentCtx context.Context, dbTx *gorm.DB, block *mq.BlockResultMsg, blockResults *coretypes.ResultBlockResults) (err error) {
	defer func() {
		// recover from panic if one occurred. Set err to nil otherwise.
		if recover() != nil {
			err = fmt.Errorf("%w: panic inside DecodeAndInsertTxs, possibly invalid message", ErrorNonRetryable)
		}
	}()

	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "decodeAndInsertTxs", "Decode txs in the block and insert them into DB")
	defer span.Finish()

	txs := make([]db.Transaction, 0)
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

		txResultJsonDict, txResultJsonByte, protoTx := f.getTxResponse(block.Timestamp, coretypes.ResultTx{
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

		messagesJSON, err := json.Marshal(md)
		if err != nil {
			return errors.Join(ErrorNonRetryable, err)
		}

		addr := types.AccAddress(signers[0])
		accountInBlock[addr.String()] = true
		txHash := fmt.Sprintf("%X", cosmosTx.Tx.Hash())
		txs = append(txs, db.Transaction{
			ID:          db.GetTxID(txHash, block.Height),
			Hash:        cosmosTx.Tx.Hash(),
			BlockHeight: block.Height,
			BlockIndex:  int(idx),
			GasUsed:     txResult.GasUsed,
			GasLimit:    int64(feeTx.GetGas()),
			GasFee:      feeTx.GetFee().String(),
			ErrMsg:      errMsg,
			Success:     txResult.IsOK(),
			Sender:      addr.String(),
			Memo:        strings.ReplaceAll(memoTx.GetMemo(), "\x00", "\uFFFD"),
			Messages:    messagesJSON,
		})

		f.produceTxResultMessage(cosmosTx.Tx.Hash(), block.Height, txResultJsonByte, logger)

		accTx, err := f.getAccountTransactions(cosmosTx.Tx.Hash(), block.Height, txResult.Events, addr)
		if err != nil {
			logger.Error().Msgf("Error processing account transaction %v", err)
			return err
		}
		accTxs = append(accTxs, accTx...)
		for _, acc := range accTxs {
			accountInBlock[acc.AccountID] = true
		}
	}

	accs := make([]db.Account, 0)
	vmAddresses := make([]db.VMAddress, 0)
	for acc := range accountInBlock {
		accAddr, err := sdk.AccAddressFromBech32(acc)
		if err != nil {
			logger.Error().Msgf("Error getting account address from bech32 %v", err)
			return err
		}
		vmAddr, _ := vmtypes.NewAccountAddressFromBytes(accAddr)
		accs = append(accs, db.Account{
			Address:     accAddr.String(),
			VMAddressID: vmAddr.String(),
			Type:        string(db.BaseAccount),
		})
		vmAddresses = append(vmAddresses, db.VMAddress{VMAddress: vmAddr.String()})
	}
	if err := db.InsertVMAddressesIgnoreConflict(ctx, dbTx, vmAddresses); err != nil {
		logger.Error().Msgf("Error inserting vm addresses: %v", err)
		return err
	}
	if err = db.InsertAccountIgnoreConflict(ctx, dbTx, accs); err != nil {
		logger.Error().Msgf("Error inserting accounts %v", err)
		return err
	}

	if err = db.InsertTransactionIgnoreConflict(ctx, dbTx, txs); err != nil {
		logger.Error().Msgf("Error inserting transactions %v", err)
		return err
	}

	if err = db.InsertAccountTxsIgnoreConflict(ctx, dbTx, accTxs); err != nil {
		logger.Error().Msgf("Error inserting account transactions %v", err)
		return err
	}
	return nil
}

func (f *Indexer) getBlockResults(parentCtx context.Context, height int64) (*coretypes.ResultBlockResults, error) {
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

func (f *Indexer) processBlock(parentCtx context.Context, block *mq.BlockResultMsg) error {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "processBlock", "Parse Block and insert blocks & transactions into DB")
	defer span.Finish()

	logger.Info().Msgf("Processing block: %d", block.Height)

	blockResults, err := f.getBlockResults(ctx, block.Height)
	if err != nil {
		logger.Error().Int64("height", block.Height).Msgf("Error getting block results: %v", err)
		return err
	}

	if err := f.dbClient.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		hashBytes, err := hex.DecodeString(block.Hash)
		if err != nil {
			return ErrorNonRetryable
		}

		proposer, err := db.QueryValidatorAddress(ctx, dbTx, block.ProposerConsensusAddress)
		if err != nil {
			logger.Error().Int64("height", block.Height).Msgf("Error querying validator address: %v", err)
			return err
		}

		err = db.InsertBlockIgnoreConflict(ctx, dbTx, db.Block{
			Height:    int32(blockResults.Height),
			Hash:      hashBytes,
			Proposer:  proposer,
			Timestamp: block.Timestamp,
		})

		err = f.decodeAndInsertTxs(ctx, dbTx, block, blockResults)
		if err != nil {
			logger.Error().Int64("height", block.Height).Msgf("Error inserting transactions: %v", err)
			return err
		}
		return nil
	}); err != nil {
		logger.Error().Int64("height", blockResults.Height).Msgf("Error processing block: %v", err)
		return errors.Join(ErrorNonRetryable, err)
	}

	logger.Info().Int64("height", block.Height).Msgf("Successfully indexed block: %d", block.Height)

	return nil
}
