package validator

import (
	"encoding/json"
	"fmt"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/core-indexer/pkg/db"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

func extractValidatorAndAmount(event abci.Event) (valAddr, coin string) {
	for _, attr := range event.Attributes {
		switch attr.Key {
		case mstakingtypes.AttributeKeyValidator:
			valAddr = attr.Value
		case sdk.AttributeKeyAmount:
			coin = attr.Value
		}
	}
	return valAddr, coin
}

func processStakeChanges(stakeChanges *map[string]int64, txHash string, blockHeight int64) []db.ValidatorBondedTokenChange {
	// Group changes by validator address
	validatorChanges := make(map[string][]map[string]string)

	for key, amount := range *stakeChanges {
		parts := strings.Split(key, ".")
		if len(parts) != 2 {
			panic("invalid stake change key format: must be 'validatorAddr.denom'")
		}

		validatorAddr := parts[0]
		denom := parts[1]

		// Add token change to validator's list
		validatorChanges[validatorAddr] = append(validatorChanges[validatorAddr], map[string]string{
			"amount": fmt.Sprintf("%d", amount),
			"denom":  denom,
		})
	}

	// Convert grouped changes to ValidatorBondedTokenChange
	var changes []db.ValidatorBondedTokenChange
	for validatorAddr, tokens := range validatorChanges {
		tokensJSON, err := json.Marshal(tokens)
		if err != nil {
			panic(fmt.Sprintf("failed to marshal tokens: %v", err))
		}

		changes = append(changes, db.ValidatorBondedTokenChange{
			ValidatorAddress: validatorAddr,
			TransactionID:    db.GetTxID(txHash, blockHeight),
			BlockHeight:      blockHeight,
			Tokens:           tokensJSON,
		})
	}

	return changes
}
