package flusher

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/db"
)

// AccountTxKey is a comparable key for AccountTransaction
type AccountTxKey string

// MakeAccountTxKey creates a unique string key from AccountTransaction fields
func MakeAccountTxKey(txID, address string) AccountTxKey {
	return AccountTxKey(fmt.Sprintf("%s:%s", txID, address))
}

type DBBatchInsert struct {
	accountsInTx            map[AccountTxKey]db.AccountTransaction
	validators              map[string]db.Validator
	accounts                map[string]db.Account
	validatorBondedTokenTxs []db.ValidatorBondedTokenChange
}

type AccountInTx struct {
	TxId    string
	Account db.Account
}

func NewDBBatchInsert() *DBBatchInsert {
	return &DBBatchInsert{
		accountsInTx:            make(map[AccountTxKey]db.AccountTransaction),
		validators:              make(map[string]db.Validator),
		accounts:                make(map[string]db.Account),
		validatorBondedTokenTxs: make([]db.ValidatorBondedTokenChange, 0),
	}
}

func (b *DBBatchInsert) AddValidators(validators ...db.Validator) {
	for _, validator := range validators {
		b.validators[validator.OperatorAddress] = validator
	}
}

func (b *DBBatchInsert) AddAccounts(accounts ...db.Account) {
	for _, account := range accounts {
		b.accounts[account.Address] = account
	}
}

func (b *DBBatchInsert) AddValidatorBondedTokenTxs(txs ...db.ValidatorBondedTokenChange) {
	for _, tx := range txs {
		b.validatorBondedTokenTxs = append(b.validatorBondedTokenTxs, tx)
	}
}

func (b *DBBatchInsert) AddAccountsInTx(txHash string, blockHeight int64, sender string, accounts ...db.Account) {
	for _, account := range accounts {
		b.accounts[account.Address] = account

		accountTx := db.NewAccountTx(
			db.GetTxID(txHash, blockHeight),
			blockHeight,
			account.Address,
			sender,
		)
		key := MakeAccountTxKey(accountTx.TransactionID, accountTx.AccountID)

		b.accountsInTx[key] = accountTx
	}
}

func (b *DBBatchInsert) Flush(ctx context.Context, dbTx *gorm.DB) error {
	if len(b.accounts) > 0 {
		accounts := make([]db.Account, 0, len(b.accounts))
		for _, account := range b.accounts {
			accounts = append(accounts, account)
		}

		if err := db.InsertAccountIgnoreConflict(ctx, dbTx, accounts); err != nil {
			return err
		}
	}

	if len(b.accountsInTx) > 0 {
		txs := make([]db.AccountTransaction, 0, len(b.accountsInTx))
		for _, tx := range b.accountsInTx {
			txs = append(txs, tx)
		}

		if err := db.InsertAccountTxsIgnoreConflict(ctx, dbTx, txs); err != nil {
			return err
		}
	}

	if len(b.validators) > 0 {
		validators := make([]db.Validator, 0, len(b.validators))
		for _, validator := range b.validators {
			validators = append(validators, validator)
		}

		if err := db.InsertValidatorIgnoreConflict(ctx, dbTx, validators); err != nil {
			return err
		}
	}

	if len(b.validatorBondedTokenTxs) > 0 {
		if err := db.InsertValidatorBondedTokenChangesIgnoreConflict(ctx, dbTx, b.validatorBondedTokenTxs); err != nil {
			return err
		}
	}

	return nil
}
