package dto

import (
	"encoding/json"

	"github.com/initia-labs/core-indexer/pkg/db"
)

type AccountTxModel struct {
	Height        int64           `json:"height"`
	Timestamp     string          `json:"timestamp"`
	Address       string          `json:"address"`
	Hash          string          `json:"hash"`
	Success       bool            `json:"success"`
	Messages      json.RawMessage `json:"messages" swaggertype:"object"`
	IsSend        bool            `json:"is_send"`
	IsIbc         bool            `json:"is_ibc"`
	IsMovePublish bool            `json:"is_move_publish"`
	IsMoveUpgrade bool            `json:"is_move_upgrade"`
	IsMoveExecute bool            `json:"is_move_execute"`
	IsMoveScript  bool            `json:"is_move_script"`
	IsOpinit      bool            `json:"is_opinit"`
	IsSigner      bool            `json:"is_signer"`
}

type AccountTxResponse struct {
	Created  string          `json:"created"`
	Hash     string          `json:"hash"`
	Height   int64           `json:"height"`
	IsIbc    bool            `json:"is_ibc"`
	IsSend   bool            `json:"is_send"`
	IsSigner bool            `json:"is_signer"`
	Messages json.RawMessage `json:"messages" swaggertype:"object"`
	Sender   string          `json:"sender"`
	Success  bool            `json:"success"`
}

type AccounTxsResponse struct {
	AccounTxs  []AccountTxResponse `json:"account_txs"`
	Pagination PaginationResponse  `json:"pagination"`
}

type AccountProposalsResponse struct {
	Proposals  []db.Proposal      `json:"proposals"`
	Pagination PaginationResponse `json:"pagination"`
}
