package mq

import (
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
)

type Block struct {
	Height    int64     `json:"height"`
	Hash      string    `json:"hash"`
	Proposer  string    `json:"proposer"`
	Timestamp time.Time `json:"timestamp"`
}

type Transaction struct {
	Hash               []byte           `json:"hash"`
	BlockHeight        int64            `json:"block_height"`
	BlockIndex         int              `json:"block_index"`
	GasUsed            int64            `json:"gas_used"`
	GasLimit           uint64           `json:"gas_limit"`
	GasFee             string           `json:"gas_fee"`
	ErrMsg             *string          `json:"err_msg"`
	Success            bool             `json:"success"`
	Sender             string           `json:"sender"`
	Memo               string           `json:"memo"`
	Messages           []map[string]any `json:"messages"`
	IsIBC              bool             `json:"is_ibc"`
	IsSend             bool             `json:"is_send"`
	IsMovePublish      bool             `json:"is_move_publish"`
	IsMoveExecuteEvent bool             `json:"is_move_execute_event"`
	IsMoveExecute      bool             `json:"is_move_execute"`
	IsMoveUpgrade      bool             `json:"is_move_upgrade"`
	IsMoveScript       bool             `json:"is_move_script"`
	IsNFTTransfer      bool             `json:"is_nft_transfer"`
	IsNFTMint          bool             `json:"is_nft_mint"`
	IsNFTBurn          bool             `json:"is_nft_burn"`
	IsCollectionCreate bool             `json:"is_collection_create"`
	IsOPInit           bool             `json:"is_opinit"`
	IsInstantiate      bool             `json:"is_instantiate"`
	IsMigrate          bool             `json:"is_migrate"`
	IsUpdateAdmin      bool             `json:"is_update_admin"`
	IsClearAdmin       bool             `json:"is_clear_admin"`
	IsStoreCode        bool             `json:"is_store_code"`
}

type LCDTxResponses struct {
	Height int64          `json:"height"`
	Hash   string         `json:"hash"`
	Result map[string]any `json:"result"`
}

type RPCEndpoints struct {
	RPCs []RPC `json:"rpcs"`
}

type RPC struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

type BlockResults struct {
	Height              int64                `json:"height"`
	TxsResults          []*abci.ExecTxResult `json:"txs_results"`
	FinalizeBlockEvents []abci.Event         `json:"finalize_block_events"`
}
