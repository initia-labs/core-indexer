package dto

import (
	"time"

	"github.com/initia-labs/core-indexer/pkg/db"
)

// /indexer/validator/v1/validators

type ValidatorsResponse struct {
	Items    []ValidatorInfo    `json:"items"`
	Metadata ValidatorsMetadata `json:"metadata"`
	Total    int64              `json:"total"`
}

type ValidatorInfo struct {
	AccountAddress   string `json:"account_address"`
	CommissionRate   string `json:"commission_rate"`
	ConsensusAddress string `json:"consensus_address"`
	Details          string `json:"details"`
	Identity         string `json:"identity"`
	IsActive         bool   `json:"is_active"`
	IsJailed         bool   `json:"is_jailed"`
	Moniker          string `json:"moniker"`
	Rank             int    `json:"rank"`
	Uptime           int32  `json:"uptime,omitempty"`
	ValidatorAddress string `json:"validator_address"`
	VotingPower      string `json:"voting_power"`
	Website          string `json:"website"`
}

type ValidatorsMetadata struct {
	ActiveCount       int    `json:"active_count"`
	InactiveCount     int    `json:"inactive_count"`
	MinCommissionRate string `json:"min_commission_rate"`
	Percent33Rank     int    `json:"percent_33_rank"`
	Percent66Rank     int    `json:"percent_66_rank"`
	TotalVotingPower  string `json:"total_voting_power"`
}

// /indexer/validator/v1/validators/{operatorAddr}/info

type ValidatorInfoResponse struct {
	Info             ValidatorInfo `json:"info"`
	TotalVotingPower string        `json:"total_voting_power"`
}

// /indexer/validator/v1/validators/{operatorAddr}/uptime

type ValidatorUptimeResponse struct {
	Events          []ValidatorUptimeEventModel `json:"events"`
	Recent100Blocks []ValidatorBlockVoteModel   `json:"recent_100_blocks"`
	Uptime          ValidatorUptimeSummary      `json:"uptime"`
}

type ValidatorUptimeEventModel struct {
	Height    int64     `json:"height"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
}

type ValidatorBlockVoteModel struct {
	Height int64  `json:"height"`
	Vote   string `json:"vote"`
}

type ValidatorUptimeSummary struct {
	MissedBlocks   int `json:"missed_blocks"`
	ProposedBlocks int `json:"proposed_blocks"`
	SignedBlocks   int `json:"signed_blocks"`
	Total          int `json:"total"`
}

// /indexer/validator/v1/validators/{operatorAddr}/delegation-related-txs

type ValidatorDelegationRelatedTxsResponse struct {
	Items []ValidatorDelegationTransaction `json:"items"`
	Total int64                            `json:"total"`
}

type ValidatorDelegationTransaction struct {
	Height    int           `json:"height"`
	Messages  []MessageType `json:"messages"`
	Sender    string        `json:"sender"`
	Timestamp time.Time     `json:"timestamp"`
	Tokens    Coins         `json:"tokens"`
	TxHash    string        `json:"tx_hash"`
}

type MessageType struct {
	Type string `json:"type"`
}

// /indexer/validator/v1/validators/{operatorAddr}/proposed-blocks

type ValidatorProposedBlocksResponse struct {
	Items []ValidatorProposedBlockModel `json:"items"`
	Total int64                         `json:"total"`
}

type ValidatorProposedBlockModel struct {
	Hash             string        `json:"hash"`
	Height           int           `json:"height"`
	Timestamp        time.Time     `json:"timestamp"`
	TransactionCount int           `json:"transaction_count"`
	Validator        BlockProposer `json:"validator"`
}

// /indexer/validator/v1/validators/{operatorAddr}/historical-powers

type ValidatorHistoricalPowersResponse struct {
	Items []ValidatorHistoricalPowerModel `json:"items"`
	Total int64                           `json:"total"`
}

type ValidatorHistoricalPowerModel struct {
	HourRoundedTimestamp time.Time `json:"hour_rounded_timestamp"`
	Timestamp            time.Time `json:"timestamp"`
	VotingPower          string    `json:"voting_power"`
}

// /indexer/validator/v1/validators/{operatorAddr}/voted-proposals

type ValidatorVotedProposalsResponse struct {
	Items []ValidatorVotedProposal `json:"items"`
	Total int64                    `json:"total"`
}

type ValidatorVotedProposal struct {
	Abstain        float64   `json:"abstain"`
	IsEmergency    bool      `json:"is_emergency"`
	IsExpedited    bool      `json:"is_expedited"`
	IsVoteWeighted bool      `json:"is_vote_weighted"`
	No             float64   `json:"no"`
	NoWithVeto     float64   `json:"no_with_veto"`
	ProposalId     int       `json:"proposal_id"`
	Status         string    `json:"status"`
	Timestamp      time.Time `json:"timestamp"`
	Title          string    `json:"title"`
	TxHash         string    `json:"tx_hash"`
	Types          []string  `json:"types"`
	Yes            float64   `json:"yes"`
}

// /indexer/validator/v1/validators/{operatorAddr}/answer-counts

type ValidatorAnswerCountsResponse struct {
	Abstain    int `json:"abstain"`
	All        int `json:"all"`
	DidNotVote int `json:"did_not_vote"`
	No         int `json:"no"`
	NoWithVeto int `json:"no_with_veto"`
	Weighted   int `json:"weighted"`
	Yes        int `json:"yes"`
}

// Models

type ValidatorWithVoteCountModel struct {
	db.Validator
	db.ValidatorVoteCount
}
