package dto

import (
	"encoding/json"
	"time"
)

// /indexer/proposal/v1/proposals

type ProposalsResponse struct {
	Items []ProposalSummary `json:"items"`
	Total int64             `json:"total"`
}

type ProposalSummary struct {
	DepositEndTime time.Time       `json:"deposit_end_time"`
	Id             int             `json:"id"`
	IsEmergency    bool            `json:"is_emergency"`
	IsExpedited    bool            `json:"is_expedited"`
	Proposer       string          `json:"proposer"`
	ResolvedHeight int             `json:"resolved_height"`
	Status         string          `json:"status"`
	Title          string          `json:"title"`
	Types          json.RawMessage `json:"types"`
	VotingEndTime  time.Time       `json:"voting_end_time"`
}

// /indexer/proposal/v1/proposals/types

type ProposalsTypesResponse []string

// /indexer/proposal/v1/proposals/{proposalId}/info

type ProposalInfoResponse struct {
	Info ProposalInfo `json:"info"`
}

type ProposalInfo struct {
	Abstain                  string            `json:"abstain"`
	Content                  ProposalContent   `json:"content"`
	CreatedHeight            int               `json:"created_height"`
	CreatedTimestamp         time.Time         `json:"created_timestamp"`
	CreatedTxHash            string            `json:"created_tx_hash"`
	DepositEndTime           time.Time         `json:"deposit_end_time"`
	Description              string            `json:"description"`
	FailedReason             string            `json:"failed_reason"`
	Id                       int               `json:"id"`
	IsEmergency              bool              `json:"is_emergency"`
	IsExpedited              bool              `json:"is_expedited"`
	Messages                 json.RawMessage   `json:"messages"`
	Metadata                 string            `json:"metadata"`
	No                       string            `json:"no"`
	NoWithVeto               string            `json:"no_with_veto"`
	ProposalDeposits         []ProposalDeposit `json:"proposal_deposits"`
	Proposer                 string            `json:"proposer"`
	ResolvedHeight           int               `json:"resolved_height"`
	ResolvedTimestamp        time.Time         `json:"resolved_timestamp"`
	ResolvedTotalVotingPower string            `json:"resolved_total_voting_power"`
	Status                   string            `json:"status"`
	SubmitTime               time.Time         `json:"submit_time"`
	Title                    string            `json:"title"`
	TotalDeposit             Coins             `json:"total_deposit"`
	Types                    json.RawMessage   `json:"types"`
	Version                  string            `json:"version"`
	VotingEndTime            time.Time         `json:"voting_end_time"`
	VotingTime               time.Time         `json:"voting_time"`
	Yes                      string            `json:"yes"`
}

type ProposalContent struct {
	Messages json.RawMessage `json:"messages"`
	Metadata string          `json:"metadata"`
}

type ProposalDeposit struct {
	Amount    Coins     `json:"amount"`
	Depositor string    `json:"depositor"`
	Timestamp time.Time `json:"timestamp"`
	TxHash    string    `json:"tx_hash"`
}

type ProposalInfoModel struct {
	Id                       int       `gorm:"column:id"`
	Proposer                 string    `gorm:"column:proposer_address"`
	Types                    string    `gorm:"column:types"`
	Title                    string    `gorm:"column:title"`
	Description              string    `gorm:"column:description"`
	Status                   string    `gorm:"column:status"`
	FailedReason             string    `gorm:"column:failed_reason"`
	SubmitTime               time.Time `gorm:"column:submit_time"`
	DepositEndTime           time.Time `gorm:"column:deposit_end_time"`
	VotingTime               time.Time `gorm:"column:voting_time"`
	VotingEndTime            time.Time `gorm:"column:voting_end_time"`
	Content                  string    `gorm:"column:content"`
	Messages                 string    `gorm:"column:messages"`
	IsExpedited              bool      `gorm:"column:is_expedited"`
	IsEmergency              bool      `gorm:"column:is_emergency"`
	TotalDeposit             string    `gorm:"column:total_deposit"`
	Version                  string    `gorm:"column:version"`
	CreatedTxHash            string    `gorm:"column:created_tx_hash"`
	CreatedHeight            int       `gorm:"column:created_height"`
	CreatedTimestamp         time.Time `gorm:"column:created_timestamp"`
	ResolvedHeight           int       `gorm:"column:resolved_height"`
	ResolvedTimestamp        time.Time `gorm:"column:resolved_timestamp"`
	Metadata                 string    `gorm:"column:metadata"`
	Yes                      string    `gorm:"column:yes"`
	Abstain                  string    `gorm:"column:abstain"`
	No                       string    `gorm:"column:no"`
	NoWithVeto               string    `gorm:"column:no_with_veto"`
	ResolvedTotalVotingPower string    `gorm:"column:resolved_total_voting_power"`
}

type ProposalDepositModel struct {
	Amount    json.RawMessage `gorm:"column:amount"`
	Depositor string          `gorm:"column:depositor"`
	TxHash    string          `gorm:"column:tx_hash"`
	Timestamp time.Time       `gorm:"column:timestamp"`
}

// /indexer/proposal/v1/proposals/{proposalId}/votes

type ProposalVotesResponse struct {
	Items []ProposalVote `json:"items"`
	Total int64          `json:"total"`
}

type ProposalVote struct {
	Abstain        float64                `json:"abstain"`
	IsVoteWeighted bool                   `json:"is_vote_weighted"`
	No             float64                `json:"no"`
	NoWithVeto     float64                `json:"no_with_veto"`
	ProposalId     int                    `json:"proposal_id"`
	Timestamp      time.Time              `json:"timestamp"`
	TxHash         string                 `json:"tx_hash"`
	Validator      *ProposalValidatorVote `json:"validator"`
	Voter          string                 `json:"voter"`
	Yes            float64                `json:"yes"`
}

type ProposalValidatorVote struct {
	Identity         string `json:"identity"`
	Moniker          string `json:"moniker"`
	ValidatorAddress string `json:"validator_address"`
}

type ProposalVoteModel struct {
	ProposalID        int       `gorm:"column:proposal_id"`
	Yes               float64   `gorm:"column:yes"`
	No                float64   `gorm:"column:no"`
	NoWithVeto        float64   `gorm:"column:no_with_veto"`
	Abstain           float64   `gorm:"column:abstain"`
	IsVoteWeighted    bool      `gorm:"column:is_vote_weighted"`
	Voter             string    `gorm:"column:voter"`
	TxHash            string    `gorm:"column:tx_hash"`
	Timestamp         time.Time `gorm:"column:timestamp"`
	ValidatorAddr     string    `gorm:"column:validator_address"`
	ValidatorMoniker  string    `gorm:"column:validator_moniker"`
	ValidatorIdentity string    `gorm:"column:validator_identity"`
}

// /indexer/proposal/v1/proposals/{proposalId}/validator_votes

type ProposalValidatorVotesResponse ProposalVotesResponse

type ProposalVoteValidatorInfoModel struct {
	OperatorAddress string `gorm:"column:operator_address"`
	Moniker         string `gorm:"column:moniker"`
	Identity        string `gorm:"column:identity"`
}

type ProposalValidatorVoteModel struct {
	ValidatorAddress string    `gorm:"column:validator_address"`
	Yes              float64   `gorm:"column:yes"`
	No               float64   `gorm:"column:no"`
	NoWithVeto       float64   `gorm:"column:no_with_veto"`
	Abstain          float64   `gorm:"column:abstain"`
	IsVoteWeighted   bool      `gorm:"column:is_vote_weighted"`
	Voter            string    `gorm:"column:voter"`
	TxHash           string    `gorm:"column:tx_hash"`
	Timestamp        time.Time `gorm:"column:timestamp"`
}

// /indexer/proposal/v1/proposals/{proposalId}/answer_counts

type ProposalAnswerCountsResponse struct {
	All       ProposalAnswerCounts          `json:"all"`
	Validator ProposalValidatorAnswerCounts `json:"validator"`
}

type ProposalAnswerCounts struct {
	Abstain    int `json:"abstain"`
	No         int `json:"no"`
	NoWithVeto int `json:"no_with_veto"`
	Total      int `json:"total"`
	Weighted   int `json:"weighted"`
	Yes        int `json:"yes"`
}

type ProposalValidatorAnswerCounts struct {
	ProposalAnswerCounts
	DidNotVote      int `json:"did_not_vote"`
	TotalValidators int `json:"total_validators"`
}

type ProposalAnswerCountsModel struct {
	Yes            float64 `gorm:"column:yes"`
	No             float64 `gorm:"column:no"`
	NoWithVeto     float64 `gorm:"column:no_with_veto"`
	Abstain        float64 `gorm:"column:abstain"`
	IsVoteWeighted bool    `gorm:"column:is_vote_weighted"`
	IsValidator    bool    `gorm:"column:is_validator"`
}
