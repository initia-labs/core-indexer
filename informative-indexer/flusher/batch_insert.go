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

	accounts                map[string]db.Account
	accountsInTx            map[AccountTxKey]db.AccountTransaction
	proposals               map[int32]db.Proposal
	validators              map[string]db.Validator
	validatorBondedTokenTxs []db.ValidatorBondedTokenChange

	modules                    map[string]db.Module
	modulePublishedEvents      []db.ModuleHistory
	collections                map[string]db.Collection
	collectionMutationEvents   []db.CollectionMutationEvent
	nftMutationEvents          []db.NftMutationEvent
	collectionTransactions     []db.CollectionTransaction
	mintedNftTransactions      []db.NftTransaction
	transferredNftTransactions []db.NftTransaction
	nfts                       map[string]db.Nft
	objectNewOwners            map[string]string
	moduleTransactions         []db.ModuleTransaction
	burnedNft                  map[string]bool
	nftBurnTransactions        []db.NftTransaction
	opinitTransactions         []db.OpinitTransaction
}

func NewDBBatchInsert() *DBBatchInsert {
	return &DBBatchInsert{
		transactions:               make([]db.Transaction, 0),
		transactionEvents:          make([]db.TransactionEvent, 0),
		accountsInTx:               make(map[AccountTxKey]db.AccountTransaction),
		accounts:                   make(map[string]db.Account),
		proposals:                  make(map[int32]db.Proposal),
		validators:                 make(map[string]db.Validator),
		validatorBondedTokenTxs:    make([]db.ValidatorBondedTokenChange, 0),
		modules:                    make(map[string]db.Module),
		modulePublishedEvents:      make([]db.ModuleHistory, 0),
		collections:                make(map[string]db.Collection),
		collectionMutationEvents:   make([]db.CollectionMutationEvent, 0),
		collectionTransactions:     make([]db.CollectionTransaction, 0),
		mintedNftTransactions:      make([]db.NftTransaction, 0),
		transferredNftTransactions: make([]db.NftTransaction, 0),
		nfts:                       make(map[string]db.Nft),
		objectNewOwners:            make(map[string]string),
		moduleTransactions:         make([]db.ModuleTransaction, 0),
		burnedNft:                  make(map[string]bool),
		nftBurnTransactions:        make([]db.NftTransaction, 0),
		opinitTransactions:         make([]db.OpinitTransaction, 0),
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

	if len(b.proposals) > 0 {
		proposals := make([]db.Proposal, 0, len(b.proposals))
		for _, proposal := range b.proposals {
			proposals = append(proposals, proposal)
		}

		if err := db.InsertProposalsIgnoreConflict(ctx, dbTx, proposals); err != nil {
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

	if len(b.modulePublishedEvents) > 0 {
		if err := db.InsertModuleHistories(ctx, dbTx, b.modulePublishedEvents); err != nil {
			return err
		}
	}

	if len(b.moduleTransactions) > 0 {
		if err := db.InsertModuleTransactions(ctx, dbTx, b.moduleTransactions); err != nil {
			return err
		}
	}

	err := b.FlushCollectionAndNftRelated(ctx, dbTx)
	if err != nil {
		return err
	}

	if len(b.opinitTransactions) > 0 {
		if err := db.InsertOpinitTransactions(ctx, dbTx, b.opinitTransactions); err != nil {
			return err
		}
	}

	return nil
}

func (b *DBBatchInsert) FlushCollectionAndNftRelated(ctx context.Context, dbTx *gorm.DB) error {
	// First flush collections since NFTs have foreign key relationships to collections
	err := b.FlushCollection(ctx, dbTx)
	if err != nil {
		return err
	}

	// Then flush NFTs which depend on collections
	err = b.FlushNft(ctx, dbTx)
	if err != nil {
		return err
	}

	// Flush collection transactions after both collections and NFTs are in place
	// since it has foreign key relationships to both tables
	err = b.FlushCollectionTransactions(ctx, dbTx)
	if err != nil {
		return err
	}

	// Finally flush all NFT-related transactions (mint, transfer, burn)
	// These operations depend on NFTs being in place
	err = b.FlushMintedNft(ctx, dbTx)
	if err != nil {
		return err
	}

	err = b.FlushTransferredNft(ctx, dbTx)
	if err != nil {
		return err
	}

	err = b.FlushBurnedNft(ctx, dbTx)
	if err != nil {
		return err
	}

	err = b.FlushCollectionMutationEvents(ctx, dbTx)
	if err != nil {
		return err
	}

	err = b.FlushNftMutationEvents(ctx, dbTx)
	if err != nil {
		return err
	}
	return nil
}

func (b *DBBatchInsert) FlushCollection(ctx context.Context, dbTx *gorm.DB) error {
	if len(b.collections) > 0 {
		collections := make([]db.Collection, 0, len(b.collections))
		for _, collection := range b.collections {
			collections = append(collections, collection)
		}
		if err := db.UpsertCollection(ctx, dbTx, collections); err != nil {
			return err
		}
	}
	return nil
}

func (b *DBBatchInsert) FlushNft(ctx context.Context, dbTx *gorm.DB) error {
	if len(b.nfts) > 0 {
		nfts := make([]*db.Nft, 0, len(b.nfts))
		for _, nft := range b.nfts {
			nfts = append(nfts, &nft)
		}
		if err := db.InsertNftsOnConflictDoUpdate(ctx, dbTx, nfts); err != nil {
			return err
		}
	}
	return nil
}

func (b *DBBatchInsert) FlushCollectionTransactions(ctx context.Context, dbTx *gorm.DB) error {
	if len(b.collectionTransactions) > 0 {
		if err := db.InsertCollectionTransactions(ctx, dbTx, b.collectionTransactions); err != nil {
			return err
		}
	}
	return nil
}

func (b *DBBatchInsert) FlushTransferredNft(ctx context.Context, dbTx *gorm.DB) error {
	if len(b.objectNewOwners) > 0 {
		ids := make([]string, 0, len(b.objectNewOwners))
		for object := range b.objectNewOwners {
			ids = append(ids, object)
		}
		nfts, err := db.GetNftsByIDs(ctx, dbTx, ids)
		if err != nil {
			return err
		}
		existingNfts := make(map[string]*db.Nft)
		for _, nft := range nfts {
			existingNfts[nft.ID] = nft
			nft.Owner = b.objectNewOwners[nft.ID]
		}
		if err := db.InsertNftsOnConflictDoUpdate(ctx, dbTx, nfts); err != nil {
			return err
		}

		nftTxs := make([]db.NftTransaction, 0)
		nftHistories := make([]db.NftHistory, 0)
		for _, tx := range b.transferredNftTransactions {
			if nft, ok := existingNfts[tx.NftID]; ok {
				nftTxs = append(nftTxs, tx)
				b.collectionTransactions = append(b.collectionTransactions, db.CollectionTransaction{
					IsNftTransfer: true,
					TxID:          tx.TxID,
					NftID:         &tx.NftID,
					CollectionID:  nft.Collection,
					BlockHeight:   tx.BlockHeight,
				})
				nftHistories = append(nftHistories, db.NftHistory{
					NftID:       tx.NftID,
					TxID:        tx.TxID,
					BlockHeight: tx.BlockHeight,
					From:        nft.Owner,
					To:          b.objectNewOwners[tx.NftID],
					Remark:      db.JSON("{}"),
				})
				existingNfts[tx.NftID].Owner = b.objectNewOwners[tx.NftID]
			}
		}
		nfts = make([]*db.Nft, 0, len(existingNfts))
		for _, nft := range existingNfts {
			nfts = append(nfts, nft)
		}

		if err := db.InsertNftsOnConflictDoUpdate(ctx, dbTx, nfts); err != nil {
			return err
		}

		if err := db.InsertNftTransactions(ctx, dbTx, nftTxs); err != nil {
			return err
		}
		if err := db.InsertNftHistories(ctx, dbTx, nftHistories); err != nil {
			return err
		}

	}
	return nil
}

func (b *DBBatchInsert) FlushMintedNft(ctx context.Context, dbTx *gorm.DB) error {
	if len(b.mintedNftTransactions) > 0 {
		if err := db.InsertNftTransactions(ctx, dbTx, b.mintedNftTransactions); err != nil {
			return err
		}
	}

	return nil

}
func (b *DBBatchInsert) FlushBurnedNft(ctx context.Context, dbTx *gorm.DB) error {
	if len(b.nftBurnTransactions) > 0 {
		nftIDs := make([]string, 0, len(b.nftBurnTransactions))
		for _, tx := range b.nftBurnTransactions {
			nftIDs = append(nftIDs, tx.NftID)
		}

		if err := db.InsertNftTransactions(ctx, dbTx, b.nftBurnTransactions); err != nil {
			return err
		}

		if err := db.UpdateBurnedNftsOnConflictDoUpdate(ctx, dbTx, nftIDs); err != nil {
			return err
		}
	}

	return nil
}

func (b *DBBatchInsert) FlushCollectionMutationEvents(ctx context.Context, dbTx *gorm.DB) error {
	if len(b.collectionMutationEvents) > 0 {
		if err := db.InsertCollectionMutationEvents(ctx, dbTx, b.collectionMutationEvents); err != nil {
			return err
		}
	}
	return nil
}

func (b *DBBatchInsert) FlushNftMutationEvents(ctx context.Context, dbTx *gorm.DB) error {
	if len(b.nftMutationEvents) > 0 {
		if err := db.InsertNftMutationEvents(ctx, dbTx, b.nftMutationEvents); err != nil {
			return err
		}
	}
	return nil
}
