package dto

import (
	"encoding/json"
	"time"
)

type AccountTxModel struct {
	Height        int64           `json:"height"`
	Timestamp     string          `json:"timestamp"`
	Sender        string          `json:"sender"`
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

type AccountTx struct {
	Created       string          `json:"created"`
	Hash          string          `json:"hash"`
	Height        int64           `json:"height"`
	IsIbc         bool            `json:"is_ibc"`
	IsSend        bool            `json:"is_send"`
	IsSigner      bool            `json:"is_signer"`
	Messages      json.RawMessage `json:"messages" swaggertype:"object"`
	Sender        string          `json:"sender"`
	Success       bool            `json:"success"`
	IsMoveExecute bool            `json:"is_move_execute"`
	IsMovePublish bool            `json:"is_move_publish"`
	IsMoveScript  bool            `json:"is_move_script"`
	IsOpinit      bool            `json:"is_opinit"`
}

type AccounTxsResponse struct {
	AccounTxs  []AccountTx        `json:"account_txs"`
	Pagination PaginationResponse `json:"pagination"`
}

type AccountProposal struct {
	DepositEndTime time.Time  `json:"deposit_end_time"`
	ID             int64      `json:"id"`
	IsEmergency    bool       `json:"is_emergency"`
	IsExpedited    bool       `json:"is_expedited"`
	Proposer       string     `json:"proposer"`
	ResolvedHeight *int64      `json:"resolved_height"`
	Status         string     `json:"status"`
	Title          string     `json:"title"`
	Type           string     `json:"type"`
	VotingEndTime  *time.Time `json:"voting_end_time"`
}

type AccountProposalsResponse struct {
	Proposals  []AccountProposal  `json:"proposals"`
	Pagination PaginationResponse `json:"pagination"`
}
