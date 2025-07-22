package db

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	sdk "github.com/cosmos/cosmos-sdk/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
)

func GetTxID(hash string, blockHeight int64) string {
	return fmt.Sprintf("%s/%d", strings.ToUpper(hash), blockHeight)
}

func NormalizeEscapeString(str string) string {
	return strings.ReplaceAll(str, "\x00", "\uFFFD")
}

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

func InsertVMAddressesAndAccountsIgnoreConflict(ctx context.Context, dbTx *gorm.DB, accountsMap map[string]Account) error {
	accounts := make([]Account, 0, len(accountsMap))
	vmAddresses := make([]VMAddress, len(accountsMap))
	for _, account := range accountsMap {
		accounts = append(accounts, account)
		vmAddresses = append(vmAddresses, VMAddress{VMAddress: account.VMAddressID})
	}

	if err := InsertVMAddressesIgnoreConflict(ctx, dbTx, vmAddresses); err != nil {
		return fmt.Errorf("error inserting vm addresses: %v", err)
	}
	if err := InsertAccountsIgnoreConflict(ctx, dbTx, accounts); err != nil {
		return fmt.Errorf("error inserting accounts: %v", err)
	}

	return nil
}
