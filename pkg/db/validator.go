package db

import (
	"encoding/hex"
	"errors"
	"time"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

func NewGenesisValidator(accAddr string, msg *mstakingtypes.MsgCreateValidator) (Validator, error) {
	pubKey, ok := msg.Pubkey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return Validator{}, errors.New("invalid pubkey")
	}

	consAddr := sdk.GetConsAddress(pubKey)
	return Validator{
		AccountID:           accAddr,
		OperatorAddress:     msg.ValidatorAddress,
		ConsensusAddress:    consAddr.String(),
		Moniker:             msg.Description.GetMoniker(),
		Identity:            msg.Description.GetIdentity(),
		Website:             msg.Description.GetWebsite(),
		Details:             msg.Description.GetDetails(),
		CommissionRate:      msg.Commission.Rate.String(),
		CommissionMaxRate:   msg.Commission.MaxRate.String(),
		CommissionMaxChange: msg.Commission.MaxChangeRate.String(),
		Jailed:              false,
		IsActive:            true,
		ConsensusPubkey:     hex.EncodeToString(consAddr),
		VotingPower:         0,
		VotingPowers:        JSON("{}"),
	}, nil
}

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
