package db

import (
	"encoding/hex"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

func NewValidator(v mstakingtypes.Validator, accAddr string, conAddr sdk.ConsAddress) Validator {
	votingPowersJson, _ := v.VotingPowers.MarshalJSON()
	votingPower := int64(0)
	if v.IsBonded() {
		votingPower = v.VotingPower.Int64()
	}
	return Validator{
		AccountID:           accAddr,
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
