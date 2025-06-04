package db

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

type Validator struct {
	AccountId           string          `json:"account_id"`
	OperatorAddress     string          `json:"operator_address"`
	ConsensusAddress    string          `json:"consensus_address"`
	Moniker             string          `json:"moniker"`
	Identity            string          `json:"identity"`
	Website             string          `json:"website"`
	Details             string          `json:"details"`
	CommissionRate      string          `json:"commission_rate"`
	CommissionMaxRate   string          `json:"commission_max_rate"`
	CommissionMaxChange string          `json:"commission_max_change"`
	Jailed              bool            `json:"jailed"`
	IsActive            bool            `json:"is_active"`
	ConsensusPubkey     string          `json:"consensus_pubkey"`
	VotingPower         int64           `json:"voting_power"`
	VotingPowers        json.RawMessage `json:"voting_powers"`
}

func NewValidator(v mstakingtypes.Validator, accAddr string, conAddr sdk.ConsAddress) Validator {
	votingPowersJson, _ := v.VotingPowers.MarshalJSON()
	votingPower := int64(0)
	if v.IsBonded() {
		votingPower = v.VotingPower.Int64()
	}
	return Validator{
		AccountId:           accAddr,
		OperatorAddress:     v.OperatorAddress,
		ConsensusAddress:    conAddr.String(),
		Moniker:             v.Description.GetMoniker(),
		Identity:            v.Description.GetIdentity(),
		Website:             v.Description.GetWebsite(),
		Details:             v.Description.GetDetails(),
		CommissionRate:      v.Commission.Rate.String(),
		CommissionMaxRate:   v.Commission.MaxRate.String(),
		CommissionMaxChange: v.Commission.MaxChangeRate.String(),
		Jailed:              v.Jailed,
		IsActive:            v.IsBonded(),
		ConsensusPubkey:     hex.EncodeToString(conAddr),
		VotingPower:         votingPower,
		VotingPowers:        votingPowersJson,
	}
}

type ValidatorBondedTokenChange struct {
	ValidatorAddr string          `json:"validator_address"`
	TxId          string          `json:"transaction_id"`
	BlockHeight   int64           `json:"block_height"`
	Tokens        json.RawMessage `json:"tokens"`
}

type BlockVote int

const (
	// PROPOSE indicates that a validator is proposing the block.
	PROPOSE BlockVote = iota + 1
	// VOTE indicates that a validator is voting for the block.
	VOTE
	// ABSENT indicates that a validator is absent and not participating in voting for the block.
	ABSENT
)

func (v BlockVote) String() string {
	switch v {
	case PROPOSE:
		return "PROPOSE"
	case VOTE:
		return "VOTE"
	case ABSENT:
		return "ABSENT"
	default:
		panic("mismatch blockvote type")
	}
}

type ValidatorCommitSignatures struct {
	OperatorAddress string
	BlockHeight     int64
	Vote            BlockVote
}

func NewValidatorCommitSignatures(operatorAddress string, height int64, vote BlockVote) ValidatorCommitSignatures {
	return ValidatorCommitSignatures{
		OperatorAddress: operatorAddress,
		BlockHeight:     height,
		Vote:            vote,
	}
}

func (v ValidatorCommitSignatures) String() string {
	return fmt.Sprintf("('%s', %d, '%s')", v.OperatorAddress, v.BlockHeight, v.Vote.String())
}

type ValidatorUptime struct {
	Validator string
	VoteCount int
}

func (v ValidatorUptime) String() string {
	return fmt.Sprintf("('%s', %d)", v.Validator, v.VoteCount)
}

type ValidatorHistoricalPower struct {
	ValidatorAddress     string
	Tokens               json.RawMessage
	VotingPower          int64
	HourRoundedTimestamp time.Time
	Timestamp            time.Time
}

func NewValidatorHistoricalPower(v mstakingtypes.Validator, timestamp time.Time) (ValidatorHistoricalPower, error) {
	tokens := v.BondedTokens()
	tokensJson, err := tokens.MarshalJSON()
	if err != nil {
		return ValidatorHistoricalPower{}, err
	}
	return ValidatorHistoricalPower{
		ValidatorAddress:     v.OperatorAddress,
		Tokens:               tokensJson,
		VotingPower:          v.VotingPower.Int64(),
		HourRoundedTimestamp: timestamp.Truncate(time.Hour).UTC(),
		Timestamp:            timestamp.UTC(),
	}, nil
}

func (v ValidatorHistoricalPower) String() string {
	return fmt.Sprintf("('%s', '%s', %d, '%s','%s')", v.ValidatorAddress, string(v.Tokens), v.VotingPower, v.HourRoundedTimestamp.Format(time.RFC3339), v.Timestamp.Format(time.RFC3339))
}

type ValidatorVote struct {
	ValidatorAddress string
	Vote             string
	Height           int64
}
