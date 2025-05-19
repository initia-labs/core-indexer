package mq

import (
	"encoding/json"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

const (
	NEW_BLOCK_RESULTS_KAFKA_MESSAGE_KEY             string = "NEW_BLOCK_RESULTS"
	NEW_BLOCK_RESULTS_CLAIM_CHECK_KAFKA_MESSAGE_KEY string = "NEW_BLOCK_RESULTS_CLAIM_CHECK"
)

type KafkaMsg interface {
	Emit() kafka.Message
}

type BlockMsg struct {
	Height        int64          `json:"height"`
	Hash          string         `json:"hash"`
	Proposer      *string        `json:"proposer"`
	Timestamp     time.Time      `json:"timestamp"`
	Txs           types.Txs      `json:"txs"`
	LastCommit    *types.Commit  `json:"last_commit"`
	BlockProposer *BlockProposer `json:"block_proposer"`
}

type ClaimCheckMsg struct {
	ObjectPath string `json:"object_path"`
}

type BlockProposer struct {
	ValidatorAddress *string `json:"validator_address"`
	ConsensusAddress *string `json:"consensus_address"`
}

func NewBlockMsgBytes(resultBlock *coretypes.ResultBlock, proposerAddress, consensusAddress *string) ([]byte, error) {
	msg := &BlockMsg{
		Height:     resultBlock.Block.Height,
		Hash:       resultBlock.Block.Hash().String(),
		Proposer:   proposerAddress,
		Timestamp:  resultBlock.Block.Time,
		Txs:        resultBlock.Block.Txs,
		LastCommit: resultBlock.Block.LastCommit,
		BlockProposer: &BlockProposer{
			ValidatorAddress: proposerAddress,
			ConsensusAddress: consensusAddress,
		},
	}

	v, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return v, nil
}

type BlockResultMsg struct {
	Height              int64        `json:"height"`
	Txs                 []TxResult   `json:"txs"`
	FinalizeBlockEvents []abci.Event `json:"finalize_block_events"`
}

type TxResult struct {
	Hash          string             `json:"hash"`
	ExecTxResults *abci.ExecTxResult `json:"exec_tx_results"`
}

func NewBlockResultMsgBytes(msg BlockResultMsg) ([]byte, error) {
	v, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return v, nil
}
