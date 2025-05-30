package flusher

import (
	"context"

	"github.com/initia-labs/core-indexer/pkg/db"
)

type DBBatchInsert struct {
	accountsInTx            map[db.AccountTx]bool
	validators              map[string]db.Validator
	accounts                map[string]db.Account
	validatorBondedTokenTxs []db.ValidatorBondedTokenChange
	modules                 map[string]db.Module
}

func NewDBBatchInsert() *DBBatchInsert {
	return &DBBatchInsert{
		accountsInTx:            make(map[db.AccountTx]bool),
		validators:              make(map[string]db.Validator),
		accounts:                make(map[string]db.Account),
		validatorBondedTokenTxs: make([]db.ValidatorBondedTokenChange, 0),
		modules:                 make(map[string]db.Module),
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

func (b *DBBatchInsert) AddModules(modules ...db.Module) {
	for _, module := range modules {
		b.AddModule(module)
	}
}

func (b *DBBatchInsert) AddModule(module db.Module) {
	b.modules[module.Id] = module
}

func (b *DBBatchInsert) AddAccountsInTx(txHash string, blockHeight int64, sender string, accounts ...db.Account) {
	for _, account := range accounts {
		b.accounts[account.Address] = account
		b.accountsInTx[db.NewAccountTx(
			db.GetTxID(txHash, blockHeight),
			blockHeight,
			account.Address,
			sender,
		)] = true
	}
}

func (b *DBBatchInsert) Flush(ctx context.Context, dbTx db.Queryable) error {
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
		txs := make([]db.AccountTx, 0, len(b.accountsInTx))
		for tx := range b.accountsInTx {
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

		if err := db.InsertValidatorsOnConflictDoUpdate(ctx, dbTx, validators); err != nil {
			return err
		}
	}

	if len(b.validatorBondedTokenTxs) > 0 {
		if err := db.InsertValidatorBondedTokenChangesIgnoreConflict(ctx, dbTx, b.validatorBondedTokenTxs); err != nil {
			return err
		}
	}

	if len(b.modules) > 0 {
		modules := make([]db.Module, 0, len(b.modules))
		for _, module := range b.modules {
			modules = append(modules, module)
		}

		if err := db.InsertModulesOnConflictDoUpdate(ctx, dbTx, modules); err != nil {
			return err
		}
	}

	return nil
}
