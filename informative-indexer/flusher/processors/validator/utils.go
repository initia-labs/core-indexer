package validator

import (
	"encoding/json"
	"fmt"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	"github.com/initia-labs/core-indexer/informative-indexer/flusher/utils"
	"github.com/initia-labs/core-indexer/pkg/db"
)

func extractValidatorAndAmount(event abci.Event) (string, string, error) {
	valAddr, found := utils.FindAttribute(event.Attributes, mstakingtypes.AttributeKeyValidator)
	if !found {
		return "", "", fmt.Errorf("failed to find validator address in %s", event.Type)
	}
	coin, found := utils.FindAttribute(event.Attributes, sdk.AttributeKeyAmount)
	if !found {
		return "", "", fmt.Errorf("failed to find amount in %s", event.Type)
	}

	return valAddr, coin, nil
}

func processStakeChanges(stakeChanges map[string]int64, txID string, blockHeight int64) ([]db.ValidatorBondedTokenChange, error) {
	// Group changes by validator address
	validatorChanges := make(map[string][]map[string]string)

	for key, amount := range stakeChanges {
		parts := strings.Split(key, ".")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid stake change key format: must be 'validatorAddr.denom'")
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
			return nil, fmt.Errorf("failed to marshal tokens: %w", err)
		}

		changes = append(changes, db.ValidatorBondedTokenChange{
			ValidatorAddress: validatorAddr,
			TransactionID:    txID,
			BlockHeight:      blockHeight,
			Tokens:           tokensJSON,
		})
	}

	return changes, nil
}
