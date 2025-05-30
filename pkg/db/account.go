package db

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
)

type Account struct {
	Address   string `json:"address"`
	VMAddress string `json:"vm_address_id"`
}

func NewAccountFromSDKAddress(address sdk.AccAddress) Account {
	return Account{
		Address:   address.String(),
		VMAddress: movetypes.ConvertSDKAddressToVMAddress(address).String(),
	}
}

type AccountTx struct {
	TxId        string `json:"transaction_id"`
	BlockHeight int64  `json:"block_height"`
	Account     string `json:"account_id"`
	IsSigner    bool   `json:"is_signer"`
}

func NewAccountTx(txId string, blockHeight int64, address string, signer string) AccountTx {
	return AccountTx{
		TxId:        txId,
		BlockHeight: blockHeight,
		Account:     address,
		IsSigner:    signer == address,
	}
}
