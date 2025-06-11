package dto

import (
	"encoding/json"
)

// ModuleResponse represents the response for a module
type ModuleResponse struct {
	ModuleName    string `json:"module_name"`
	Digest        string `json:"digest"`
	IsVerified    bool   `json:"is_verified"`
	Address       string `json:"address"`
	Height        int64  `json:"height"`
	LatestUpdated string `json:"latest_updated"`
	IsRepublished bool   `json:"is_republished"`
}

// ModulesResponse represents the response for a list of modules
type ModulesResponse struct {
	Modules    []ModuleResponse   `json:"modules"`
	Pagination PaginationResponse `json:"pagination"`
}

// ModuleHistoryResponse represents the response for a module history
type ModuleHistoryResponse struct {
	Height         int64           `json:"height"`
	Remark         json.RawMessage `json:"remark"`
	UpgradePolicy  string          `json:"upgrade_policy"`
	Timestamp      string          `json:"timestamp"`
	PreviousPolicy *string         `json:"previous_policy"`
}

// ModuleHistoriesResponse represents the response for a list of module histories
type ModuleHistoriesResponse struct {
	ModuleHistories []ModuleHistoryResponse `json:"module_histories"`
	Pagination      PaginationResponse      `json:"pagination"`
}

type Proposal struct {
	ProposalID    int32  `json:"proposal_id"`
	ProposalTitle string `json:"proposal_title"`
}

type ModulePublishInfoModel struct {
	Height          int64     `json:"height"`
	Proposal        *Proposal `gorm:"embedded;embeddedPrefix:proposal_" json:"proposal"`
	Timestamp       string    `json:"timestamp"`
	TransactionHash *string   `json:"transaction_hash"`
}

// ModulePublishInfoResponse represents the response for a module publish info
type ModulePublishInfoResponse struct {
	RecentPublishTransaction    *string   `json:"recent_publish_transaction"`
	IsRepublished               bool      `json:"is_republished"`
	RecentPublishBlockHeight    int64     `json:"recent_publish_block_height"`
	RecentPublishBlockTimestamp string    `json:"recent_publish_block_timestamp"`
	RecentPublishProposal       *Proposal `json:"recent_publish_proposal"`
}

type ModuleProposalModel struct {
	ID             int32  `json:"id"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	VotingEndTime  string `json:"voting_end_time"`
	DepositEndTime string `json:"deposit_end_time"`
	Types          string `json:"types"`
	IsExpedited    bool   `json:"is_expedited"`
	IsEmergency    bool   `json:"is_emergency"`
	ResolvedHeight int64  `json:"resolved_height"`
	Proposer       string `json:"proposer"`
}

// ModuleProposalsResponse represents the response for a module proposal
type ModuleProposalsResponse struct {
	Proposals  []ModuleProposalModel `json:"proposals"`
	Pagination PaginationResponse    `json:"pagination"`
}

// ModuleTxResponse represents the response for a module tx
type ModuleTxResponse struct {
	Height             int64           `json:"height"`
	Timestamp          string          `json:"timestamp"`
	Sender             string          `json:"sender"`
	TxHash             string          `json:"hash" gorm:"column:hash"`
	Success            bool            `json:"success"`
	Messages           json.RawMessage `json:"messages"`
	IsSend             bool            `json:"is_send"`
	IsIBC              bool            `json:"is_ibc"`
	IsMoveExecute      bool            `json:"is_move_execute"`
	IsMoveExecuteEvent bool            `json:"is_move_execute_event"`
	IsMovePublish      bool            `json:"is_move_publish"`
	IsMoveScript       bool            `json:"is_move_script"`
	IsMoveUpgrade      bool            `json:"is_move_upgrade"`
	IsOpinit           bool            `json:"is_opinit"`
}

// ModuleTxsResponse represents the response for a list of module txs
type ModuleTxsResponse struct {
	ModuleTxs  []ModuleTxResponse `json:"module_txs"`
	Pagination PaginationResponse `json:"pagination"`
}

// ModuleStatsResponse represents the response for a module stats
type ModuleStatsResponse struct {
	TotalHistories int64  `json:"total_histories"`
	TotalProposals *int64 `json:"total_proposals"`
	TotalTxs       int64  `json:"total_txs"`
}
