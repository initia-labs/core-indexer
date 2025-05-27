package db

import (
	"encoding/hex"
	"encoding/json"

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
