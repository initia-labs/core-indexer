package mq

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/rs/zerolog"
)

var logger *zerolog.Logger

const (
	NEW_BLOCK_RESULTS_KAFKA_MESSAGE_KEY             string = "NEW_BLOCK_RESULTS"
	NEW_BLOCK_RESULTS_CLAIM_CHECK_KAFKA_MESSAGE_KEY string = "NEW_BLOCK_RESULTS_CLAIM_CHECK"
)

type KafkaMsg interface {
	Emit() kafka.Message
}

type TxResult struct {
	Hash          string             `json:"hash"`
	ExecTxResults *abci.ExecTxResult `json:"exec_tx_results"`
	Tx            types.Tx           `json:"tx"`
}

type BlockResultMsg struct {
	Timestamp                time.Time     `json:"timestamp"`
	Height                   int64         `json:"height"`
	Txs                      []TxResult    `json:"txs"`
	FinalizeBlockEvents      []abci.Event  `json:"finalize_block_events"`
	LastCommit               *types.Commit `json:"last_commit"`
	ProposerConsensusAddress string        `json:"proposer_consensus_address"`
}

type ClaimCheckMsg struct {
	ObjectPath string `json:"object_path"`
}

func NewBlockResultMsgBytes(block *coretypes.ResultBlock, blockResult *coretypes.ResultBlockResults) ([]byte, error) {
	consensusAddress, err := bech32.ConvertAndEncode("initvalcons", block.Block.ProposerAddress)
	if err != nil {
		logger.Error().Msgf("Failed to convert and encode Bech32: %v\n", err)
		return nil, err
	}

	txResults := make([]TxResult, len(blockResult.TxsResults))
	for idx, txResult := range blockResult.TxsResults {
		hash := sha256.Sum256(block.Block.Data.Txs[idx])
		txResults[idx] = TxResult{
			Hash:          hex.EncodeToString(hash[:]),
			ExecTxResults: txResult,
			Tx:            block.Block.Data.Txs[idx],
		}
	}

	msg := &BlockResultMsg{
		Timestamp:                block.Block.Time,
		Height:                   block.Block.Height,
		Txs:                      txResults,
		FinalizeBlockEvents:      blockResult.FinalizeBlockEvents,
		LastCommit:               block.Block.LastCommit,
		ProposerConsensusAddress: consensusAddress,
	}

	v, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return v, nil
}
