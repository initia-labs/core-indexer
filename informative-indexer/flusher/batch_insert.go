package flusher

import (
	"context"

	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/db"
)

type DBBatchInsert struct {
	validators              map[string]db.Validator
	accounts                map[string]db.Account
	validatorBondedTokenTxs []db.ValidatorBondedTokenChange
}

func NewDBBatchInsert() *DBBatchInsert {
	return &DBBatchInsert{
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
