package db

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/mstaking/types"
)

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

// Validator represents data queried from the database for a validator.
type ValidatorRelation struct {
	// OperatorAddress is the validator's operator address, for example, 'initvaloper'.
	OperatorAddress string
	// ConsensusAddress is the validator's consensus public key, for example, 'initvalconpub'.
	ConsensusAddress string
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

type Validator struct {
	AccountId           string
	OperatorAddress     string
	ConsensusAddress    string
	Moniker             string
	Identity            string
	Website             string
	Details             string
	CommissionRate      string
	CommissionMaxRate   string
	CommissionMaxChange string
	Jailed              bool
	IsActive            bool
	ConsensusPubkey     string
	VotingPower         int64
	VotingPowers        json.RawMessage
}

func NewValidator(v types.Validator, address string, conAddr sdk.ConsAddress) Validator {
	votingPowersJson, _ := v.VotingPowers.MarshalJSON()
	votingPower := int64(0)
	if v.IsBonded() {
		votingPower = v.VotingPower.Int64()
	}
	return Validator{
		AccountId:           address,
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

func NewValidatorHistoricalPower(v types.Validator, timestamp time.Time) (ValidatorHistoricalPower, error) {
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
