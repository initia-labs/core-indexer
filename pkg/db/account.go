package db

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
)

func NewAccountFromSDKAddress(address sdk.AccAddress) Account {
	return Account{
		Address:     address.String(),
		VMAddressID: movetypes.ConvertSDKAddressToVMAddress(address).String(),
		Type:        string(BaseAccount),
	}
}

func NewAccountTx(txId string, blockHeight int64, address string, signer string) AccountTransaction {
	return AccountTransaction{
		TransactionID: txId,
		BlockHeight:   blockHeight,
		AccountID:     address,
		IsSigner:      signer == address,
	}
}
