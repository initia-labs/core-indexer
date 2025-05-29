package parser

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func ParseCoinAmount(coinStr string) (amount int64, denom string, err error) {
	coin, err := sdk.ParseCoinNormalized(coinStr)
	if err != nil {
		return 0, "", err
	}

	amount = coin.Amount.Int64()
	denom = coin.Denom
	return amount, denom, nil
}
