package statetracker

import (
	"context"
	"fmt"
	"maps"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog"
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

	accounts                   map[string]db.Account
	accountsInTx               map[AccountTxKey]db.AccountTransaction
	proposals                  map[int32]db.Proposal
	ProposalStatusChanges      map[int32]db.Proposal
	ProposalExpeditedChanges   map[int32]bool
	ProposalEmergencyNextTally map[int32]*time.Time
	validators                 map[string]db.Validator
	validatorBondedTokenTxs    []db.ValidatorBondedTokenChange

	modules                    map[string]db.Module
	ModulePublishedEvents      []db.ModuleHistory
	ModuleProposals            []db.ModuleProposal
	Collections                map[string]db.Collection
	CollectionMutationEvents   []db.CollectionMutationEvent
	NftMutationEvents          []db.NftMutationEvent
	CollectionTransactions     []db.CollectionTransaction
	MintedNftTransactions      []db.NftTransaction
	TransferredNftTransactions []db.NftTransaction
	Nfts                       map[string]db.Nft
	ObjectNewOwners            map[string]string
	ModuleTransactions         []db.ModuleTransaction
	BurnedNft                  map[string]bool
	NftBurnTransactions        []db.NftTransaction
	OpinitTransactions         []db.OpinitTransaction
	ProposalDeposits           []db.ProposalDeposit
	TotalDepositChanges        map[int32][]sdk.Coin
	ProposalVotes              []db.ProposalVote
	ValidatorSlashEvents       []db.ValidatorSlashEvent

	logger *zerolog.Logger
}

func NewDBBatchInsert(logger *zerolog.Logger) *DBBatchInsert {
	return &DBBatchInsert{
		transactions:               make([]db.Transaction, 0),
		transactionEvents:          make([]db.TransactionEvent, 0),
		accountsInTx:               make(map[AccountTxKey]db.AccountTransaction),
		accounts:                   make(map[string]db.Account),
		proposals:                  make(map[int32]db.Proposal),
		ProposalStatusChanges:      make(map[int32]db.Proposal),
		ProposalExpeditedChanges:   make(map[int32]bool),
		ProposalEmergencyNextTally: make(map[int32]*time.Time),
		validators:                 make(map[string]db.Validator),
		validatorBondedTokenTxs:    make([]db.ValidatorBondedTokenChange, 0),
		modules:                    make(map[string]db.Module),
		ModulePublishedEvents:      make([]db.ModuleHistory, 0),
		ModuleProposals:            make([]db.ModuleProposal, 0),
		Collections:                make(map[string]db.Collection),
		CollectionMutationEvents:   make([]db.CollectionMutationEvent, 0),
		CollectionTransactions:     make([]db.CollectionTransaction, 0),
		MintedNftTransactions:      make([]db.NftTransaction, 0),
		TransferredNftTransactions: make([]db.NftTransaction, 0),
		Nfts:                       make(map[string]db.Nft),
		ObjectNewOwners:            make(map[string]string),
		ModuleTransactions:         make([]db.ModuleTransaction, 0),
		BurnedNft:                  make(map[string]bool),
		NftBurnTransactions:        make([]db.NftTransaction, 0),
		OpinitTransactions:         make([]db.OpinitTransaction, 0),
		ProposalDeposits:           make([]db.ProposalDeposit, 0),
		ProposalVotes:              make([]db.ProposalVote, 0),
		logger:                     logger,
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

func (b *DBBatchInsert) AddValidatorSlashEvents(slashEvents ...db.ValidatorSlashEvent) {
	b.ValidatorSlashEvents = append(b.ValidatorSlashEvents, slashEvents...)
}

func (b *DBBatchInsert) AddModules(modules ...db.Module) {
	for _, module := range modules {
		b.AddModule(module)
	}
}

func (b *DBBatchInsert) AddModule(module db.Module) {
	b.modules[module.ID] = module
}

func (b *DBBatchInsert) AddAccountsInTx(accounts map[string]db.Account, accountsInTx map[AccountTxKey]db.AccountTransaction) {
	maps.Copy(b.accounts, accounts)
	maps.Copy(b.accountsInTx, accountsInTx)
}

func (b *DBBatchInsert) Flush(ctx context.Context, dbTx *gorm.DB, height int64) error {
	if err := db.UpdateTxCount(ctx, dbTx, int64(len(b.transactions)), height); err != nil {
		b.logger.Error().Msgf("Error updating tracking table: %v", err)
		return err
	}

	if len(b.accounts) > 0 {
		accounts := make([]db.Account, 0, len(b.accounts))
		vmAddresses := make([]db.VMAddress, len(b.accounts))
		for _, account := range b.accounts {
			accounts = append(accounts, account)
			vmAddresses = append(vmAddresses, db.VMAddress{VMAddress: account.VMAddressID})
		}

		if err := db.InsertVMAddressesIgnoreConflict(ctx, dbTx, vmAddresses); err != nil {
			b.logger.Error().Msgf("Error inserting vm addresses: %v", err)
			return err
		}

		if err := db.InsertAccountIgnoreConflict(ctx, dbTx, accounts); err != nil {
			return err
		}
	}

	if len(b.transactions) > 0 {
		if err := db.UpsertTransactions(ctx, dbTx, b.transactions); err != nil {
			b.logger.Error().Msgf("Error inserting transactions: %v", err)
			return err
		}
	}

	if len(b.ValidatorSlashEvents) > 0 {
		if err := db.InsertValidatorSlashEvents(ctx, dbTx, b.ValidatorSlashEvents); err != nil {
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

	if len(b.ProposalStatusChanges) > 0 {
		proposals := make([]db.Proposal, 0, len(b.ProposalStatusChanges))
		for _, proposal := range b.ProposalStatusChanges {
			proposals = append(proposals, proposal)
		}
		if err := db.UpdateProposalStatus(ctx, dbTx, proposals); err != nil {
			return err
		}
	}

	if len(b.ProposalExpeditedChanges) > 0 {
		proposalIDs := make([]int32, 0, len(b.ProposalExpeditedChanges))
		for proposalID := range b.ProposalExpeditedChanges {
			proposalIDs = append(proposalIDs, proposalID)
			if err := db.UpdateProposalExpedited(ctx, dbTx, proposalIDs); err != nil {
				return err
			}
		}
	}

	if len(b.ProposalDeposits) > 0 {
		if err := db.InsertProposalDeposits(ctx, dbTx, b.ProposalDeposits); err != nil {
			return err
		}
		if err := db.UpdateProposalTotalDeposit(ctx, dbTx, b.TotalDepositChanges); err != nil {
			return err
		}
	}

	if len(b.ProposalVotes) > 0 {
		// TODO: cache validator addresses
		validatorAddresses, err := db.QueryValidatorAddresses(ctx, dbTx)
		if err != nil {
			return err
		}

		for idx := range b.ProposalVotes {
			if validatorAddress, ok := validatorAddresses[b.ProposalVotes[idx].Voter]; ok {
				b.ProposalVotes[idx].IsValidator = true
				b.ProposalVotes[idx].ValidatorAddress = &validatorAddress.OperatorAddress
			}
		}

		if err := db.UpsertProposalVotes(ctx, dbTx, b.ProposalVotes); err != nil {
			return err
		}
	}

	if len(b.ProposalEmergencyNextTally) > 0 {
		if err := db.UpdateProposalEmergencyNextTally(ctx, dbTx, b.ProposalEmergencyNextTally); err != nil {
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

	if len(b.ModulePublishedEvents) > 0 {
		if err := db.InsertModuleHistories(ctx, dbTx, b.ModulePublishedEvents); err != nil {
			return err
		}
	}

	if len(b.ModuleTransactions) > 0 {
		if err := db.InsertModuleTransactions(ctx, dbTx, b.ModuleTransactions); err != nil {
			return err
		}
	}

	if len(b.ModuleProposals) > 0 {
		if err := db.InsertModuleProposalsOnConflictDoUpdate(ctx, dbTx, b.ModuleProposals); err != nil {
			return err
		}
	}

	err := b.FlushCollectionAndNftRelated(ctx, dbTx)
	if err != nil {
		return err
	}

	if len(b.OpinitTransactions) > 0 {
		if err := db.InsertOpinitTransactions(ctx, dbTx, b.OpinitTransactions); err != nil {
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
	if len(b.Collections) > 0 {
		collections := make([]db.Collection, 0, len(b.Collections))
		for _, collection := range b.Collections {
			collections = append(collections, collection)
		}
		if err := db.UpsertCollection(ctx, dbTx, collections); err != nil {
			return err
		}
	}
	return nil
}

func (b *DBBatchInsert) FlushNft(ctx context.Context, dbTx *gorm.DB) error {
	if len(b.Nfts) > 0 {
		nfts := make([]*db.Nft, 0, len(b.Nfts))
		for _, nft := range b.Nfts {
			nfts = append(nfts, &nft)
		}
		if err := db.InsertNftsOnConflictDoUpdate(ctx, dbTx, nfts); err != nil {
			return err
		}
	}
	return nil
}

func (b *DBBatchInsert) FlushCollectionTransactions(ctx context.Context, dbTx *gorm.DB) error {
	if len(b.CollectionTransactions) > 0 {
		if err := db.InsertCollectionTransactions(ctx, dbTx, b.CollectionTransactions); err != nil {
			return err
		}
	}
	return nil
}

func (b *DBBatchInsert) FlushTransferredNft(ctx context.Context, dbTx *gorm.DB) error {
	if len(b.ObjectNewOwners) > 0 {
		ids := make([]string, 0, len(b.ObjectNewOwners))
		for object := range b.ObjectNewOwners {
			ids = append(ids, object)
		}
		nfts, err := db.GetNftsByIDs(ctx, dbTx, ids)
		if err != nil {
			return err
		}
		existingNfts := make(map[string]*db.Nft)
		for _, nft := range nfts {
			existingNfts[nft.ID] = nft
			nft.Owner = b.ObjectNewOwners[nft.ID]
		}
		if err := db.InsertNftsOnConflictDoUpdate(ctx, dbTx, nfts); err != nil {
			return err
		}

		nftTxs := make([]db.NftTransaction, 0)
		nftHistories := make([]db.NftHistory, 0)
		collectionTransactions := make([]db.CollectionTransaction, 0)
		for _, tx := range b.TransferredNftTransactions {
			if nft, ok := existingNfts[tx.NftID]; ok {
				nftTxs = append(nftTxs, tx)
				collectionTransactions = append(collectionTransactions, db.CollectionTransaction{
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
					To:          b.ObjectNewOwners[tx.NftID],
					Remark:      db.JSON("{}"),
				})
				existingNfts[tx.NftID].Owner = b.ObjectNewOwners[tx.NftID]
			}
		}

		if err := db.InsertCollectionTransactions(ctx, dbTx, collectionTransactions); err != nil {
			return err
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
	if len(b.MintedNftTransactions) > 0 {
		if err := db.InsertNftTransactions(ctx, dbTx, b.MintedNftTransactions); err != nil {
			return err
		}
	}

	return nil
}

func (b *DBBatchInsert) FlushBurnedNft(ctx context.Context, dbTx *gorm.DB) error {
	if len(b.NftBurnTransactions) > 0 {
		nftIDs := make([]string, 0, len(b.NftBurnTransactions))
		for _, tx := range b.NftBurnTransactions {
			nftIDs = append(nftIDs, tx.NftID)
		}

		if err := db.InsertNftTransactions(ctx, dbTx, b.NftBurnTransactions); err != nil {
			return err
		}

		if err := db.UpdateBurnedNftsOnConflictDoUpdate(ctx, dbTx, nftIDs); err != nil {
			return err
		}
	}

	return nil
}

func (b *DBBatchInsert) FlushCollectionMutationEvents(ctx context.Context, dbTx *gorm.DB) error {
	if len(b.CollectionMutationEvents) > 0 {
		if err := db.InsertCollectionMutationEvents(ctx, dbTx, b.CollectionMutationEvents); err != nil {
			return err
		}
		for _, event := range b.CollectionMutationEvents {
			switch event.MutatedFieldName {
			case "uri":
				if err := db.UpdateCollectionURI(ctx, dbTx, event.CollectionID, event.NewValue); err != nil {
					return err
				}
			case "description":
				if err := db.UpdateCollectionDescription(ctx, dbTx, event.CollectionID, event.NewValue); err != nil {
					return err
				}
			case "name":
				if err := db.UpdateCollectionName(ctx, dbTx, event.CollectionID, event.NewValue); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (b *DBBatchInsert) FlushNftMutationEvents(ctx context.Context, dbTx *gorm.DB) error {
	if len(b.NftMutationEvents) > 0 {
		if err := db.InsertNftMutationEvents(ctx, dbTx, b.NftMutationEvents); err != nil {
			return err
		}
		for _, event := range b.NftMutationEvents {
			switch event.MutatedFieldName {
			case "uri":
				if err := db.UpdateNftURI(ctx, dbTx, event.NftID, event.NewValue); err != nil {
					return err
				}
			case "description":
				if err := db.UpdateNftDescription(ctx, dbTx, event.NftID, event.NewValue); err != nil {
					return err
				}
			}
		}

	}
	return nil
}
