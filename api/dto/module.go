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

// ModuleProposalResponse represents the response for a module proposal
type ModuleProposalResponse struct {
	Proposals  []ModuleProposalModel `json:"proposals"`
	Pagination PaginationResponse    `json:"pagination"`
}
