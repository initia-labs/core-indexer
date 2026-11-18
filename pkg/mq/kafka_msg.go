package mq

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/rs/zerolog"
)

var logger *zerolog.Logger

const (
	NEW_BLOCK_RESULTS_MESSAGE_KEY               string = "NEW_BLOCK_RESULTS"
	NEW_BLOCK_RESULTS_CLAIM_CHECK_MESSAGE_KEY   string = "NEW_BLOCK_RESULTS_CLAIM_CHECK"
	NEW_LCD_TX_RESPONSE_MESSAGE_KEY             string = "NEW_LCD_TX_RESPONSE"
	NEW_LCD_TX_RESPONSE_CLAIM_CHECK_MESSAGE_KEY string = "NEW_LCD_TX_RESPONSE_CLAIM_CHECK"
)

type TxResult struct {
	Hash          string             `json:"hash"`
	ExecTxResults *abci.ExecTxResult `json:"exec_tx_results"`
	Tx            types.Tx           `json:"tx"`
}

type BlockResultMsg struct {
	Hash                     string        `json:"hash"`
	Height                   int64         `json:"height"`
	Timestamp                time.Time     `json:"timestamp"`
	Txs                      []TxResult    `json:"txs"`
	FinalizeBlockEvents      []abci.Event  `json:"finalize_block_events"`
	LastCommit               *types.Commit `json:"last_commit"`
	ProposerConsensusAddress string        `json:"proposer_consensus_address"`

	// version is used to track the version of the message
	// consumer can use this to track the version of the message and handle the message accordingly
	Version int64 `json:"version"`
}

type ClaimCheckMsg struct {
	ObjectPath string `json:"object_path"`
}

func NewBlockResultMsgBytes(block *coretypes.ResultBlock, blockResult *coretypes.ResultBlockResults) ([]byte, error) {
	if block == nil || block.Block == nil || blockResult == nil {
		return nil, fmt.Errorf("missing block or block results data")
	}
	if len(blockResult.TxsResults) > len(block.Block.Data.Txs) {
		return nil, fmt.Errorf("block results contain more transactions than block data")
	}
	consensusAddress, err := bech32.ConvertAndEncode("initvalcons", block.Block.ProposerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid block proposer address: %w", err)
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
		Hash:                     block.Block.Hash().String(),
		Timestamp:                block.Block.Time,
		Height:                   block.Block.Height,
		Txs:                      txResults,
		FinalizeBlockEvents:      blockResult.FinalizeBlockEvents,
		LastCommit:               block.Block.LastCommit,
		ProposerConsensusAddress: consensusAddress,

		// version is used to track the version of the message
		Version: 0,
	}

	v, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return v, nil
}
