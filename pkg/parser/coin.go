package parser

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func ParseCoinAmount(coinStr string) (int64, string, error) {
	coin, err := sdk.ParseCoinNormalized(coinStr)
	if err != nil {
		return 0, "", err
	}

	return coin.Amount.Int64(), coin.Denom, nil
}
