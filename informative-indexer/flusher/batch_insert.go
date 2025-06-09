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
	transactions      []db.Transaction
	transactionEvents []db.TransactionEvent

	accountsInTx            map[AccountTxKey]db.AccountTransaction
	validators              map[string]db.Validator
	accounts                map[string]db.Account
	validatorBondedTokenTxs []db.ValidatorBondedTokenChange

	modules                  map[string]db.Module
	collections              map[string]db.Collection
	collectionMutationEvents []db.CollectionMutationEvent
	collectionTransactions   []db.CollectionTransaction
	nftTransactions          []db.NftTransaction
	nfts                     map[string]db.Nft
}

func NewDBBatchInsert() *DBBatchInsert {
	return &DBBatchInsert{
		transactions:             make([]db.Transaction, 0),
		transactionEvents:        make([]db.TransactionEvent, 0),
		accountsInTx:             make(map[AccountTxKey]db.AccountTransaction),
		validators:               make(map[string]db.Validator),
		accounts:                 make(map[string]db.Account),
		validatorBondedTokenTxs:  make([]db.ValidatorBondedTokenChange, 0),
		modules:                  make(map[string]db.Module),
		collections:              make(map[string]db.Collection),
		collectionMutationEvents: make([]db.CollectionMutationEvent, 0),
		collectionTransactions:   make([]db.CollectionTransaction, 0),
		nftTransactions:          make([]db.NftTransaction, 0),
		nfts:                     make(map[string]db.Nft),
	}
}

func (b *DBBatchInsert) AddTransaction(transaction db.Transaction) {
	b.transactions = append(b.transactions, transaction)
}

func (b *DBBatchInsert) AddTransactionEvent(transactionEvent db.TransactionEvent) {
	b.transactionEvents = append(b.transactionEvents, transactionEvent)
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
	b.validatorBondedTokenTxs = append(b.validatorBondedTokenTxs, txs...)
}

func (b *DBBatchInsert) AddModules(modules ...db.Module) {
	for _, module := range modules {
		b.AddModule(module)
	}
}

func (b *DBBatchInsert) AddModule(module db.Module) {
	b.modules[module.ID] = module
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
		vmAddresses := make([]db.VMAddress, len(b.accounts))
		for _, account := range b.accounts {
			accounts = append(accounts, account)
			vmAddresses = append(vmAddresses, db.VMAddress{VMAddress: account.VMAddressID})
		}

		if err := db.InsertVMAddressesIgnoreConflict(ctx, dbTx, vmAddresses); err != nil {
			logger.Error().Msgf("Error inserting vm addresses: %v", err)
			return err
		}

		if err := db.InsertAccountIgnoreConflict(ctx, dbTx, accounts); err != nil {
			return err
		}
	}

	if len(b.transactions) > 0 {
		if err := db.InsertTransactionIgnoreConflict(ctx, dbTx, b.transactions); err != nil {
			logger.Error().Msgf("Error inserting transactions: %v", err)
			return err
		}
	}

	if len(b.transactionEvents) > 0 {
		if err := db.InsertTransactionEventsIgnoreConflict(ctx, dbTx, b.transactionEvents); err != nil {
			logger.Error().Msgf("Error inserting transaction_events: %v", err)
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

		if err := db.UpsertValidators(ctx, dbTx, validators); err != nil {
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

		if err := db.UpsertModules(ctx, dbTx, modules); err != nil {
			return err
		}
	}

	if len(b.collections) > 0 {
		collections := make([]db.Collection, 0, len(b.collections))
		for _, collection := range b.collections {
			fmt.Println("collection", collection)
			collections = append(collections, collection)
		}
		fmt.Println("collections", len(collections))
		if err := db.UpsertCollection(ctx, dbTx, collections); err != nil {
			return err
		}

	}

	if len(b.collectionTransactions) > 0 {
		if err := db.InsertCollectionTransactions(ctx, dbTx, b.collectionTransactions); err != nil {
			return err
		}
	}
	if len(b.nfts) > 0 {
		nfts := make([]db.Nft, 0, len(b.nfts))
		for _, nft := range b.nfts {
			nfts = append(nfts, nft)
		}
		if err := db.InsertNftsOnConflictDoUpdate(ctx, dbTx, nfts); err != nil {
			return err
		}
	}
	if len(b.nftTransactions) > 0 {
		if err := db.InsertNftTransactions(ctx, dbTx, b.nftTransactions); err != nil {
			return err
		}
	}

	return nil
}
