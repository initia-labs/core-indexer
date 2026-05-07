package move

import (
	"fmt"

	"github.com/initia-labs/initia/app/params"
	vmapi "github.com/initia-labs/movevm/api"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/cacher"
	statetracker "github.com/initia-labs/core-indexer/informative-indexer/indexer/state-tracker"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

func (p *Processor) InitProcessor(height int64, cacher *cacher.Cacher) {
	p.Height = height
	p.Cacher = cacher
	p.newModules = make(map[vmapi.ModuleInfoResponse]string)
	p.moduleTransactions = make([]db.ModuleTransaction, 0)
	p.newCollections = make(map[string]db.Collection)
	p.collectionTransactions = make([]db.CollectionTransaction, 0)
	p.newNfts = make(map[string]db.Nft)
	p.mintedNftTransactions = make([]db.NftTransaction, 0)
	p.burnedNftTransactions = make([]db.NftTransaction, 0)
	p.objectOwners = make(map[string]string)

	p.modulePublishedEvents = make([]db.ModuleHistory, 0)
	p.collectionMutationEvents = make([]db.CollectionMutationEvent, 0)
	p.nftMutationEvents = make([]db.NftMutationEvent, 0)
}

func (p *Processor) Name() string {
	return "account"
}

func (p *Processor) NewTxProcessor(txData *db.Transaction) {
	p.txProcessor = &TxProcessor{
		txData:      txData,
		modulesInTx: make(map[vmapi.ModuleInfoResponse]bool),
		nftsMap:     make(map[string]string),
	}
}

// ProcessSDKMessages processes SDK transaction messages to identify entry points
func (p *Processor) ProcessSDKMessages(tx *mq.TxResult, encodingConfig *params.EncodingConfig) error {
	if !tx.ExecTxResults.IsOK() {
		return nil
	}

	sdkTx, err := encodingConfig.TxConfig.TxDecoder()(tx.Tx)
	if err != nil {
		return fmt.Errorf("failed to decode SDK transaction: %w", err)
	}

	for _, msg := range sdkTx.GetMsgs() {
		p.handleMsg(msg, tx.ExecTxResults.IsOK())
	}

	return nil
}

func (p *Processor) ProcessTransactionEvents(tx *mq.TxResult) error {
	for _, event := range tx.ExecTxResults.Events {
		if err := p.handleEvent(event); err != nil {
			return fmt.Errorf("failed to handle tx event %s: %w", event.Type, err)
		}
	}
	return nil
}

func (p *Processor) ResolveTxProcessor() error {
	// Update module transactions
	for module, isEntry := range p.txProcessor.modulesInTx {
		p.moduleTransactions = append(p.moduleTransactions, db.ModuleTransaction{
			IsEntry:     isEntry,
			BlockHeight: p.Height,
			TxID:        p.txProcessor.txData.ID,
			ModuleID:    db.GetModuleID(module),
		})
	}

	return nil
}

func (p *Processor) TrackState(stateUpdateManager *statetracker.StateUpdateManager, dbBatchInsert *statetracker.DBBatchInsert) error {
	// Update modules state
	for module, txID := range p.newModules {
		stateUpdateManager.Modules[module] = &txID
	}

	dbBatchInsert.ModuleTransactions = append(dbBatchInsert.ModuleTransactions, p.moduleTransactions...)

	// Update collections
	for _, collection := range p.newCollections {
		stateUpdateManager.CollectionsToUpdate[collection.ID] = true
		dbBatchInsert.Collections[collection.ID] = collection
	}
	dbBatchInsert.CollectionTransactions = append(dbBatchInsert.CollectionTransactions, p.collectionTransactions...)

	// Update NFTs
	for _, nft := range p.newNfts {
		stateUpdateManager.NftsToUpdate[nft.ID] = true
		dbBatchInsert.Nfts[nft.ID] = nft
	}
	dbBatchInsert.MintedNftTransactions = append(dbBatchInsert.MintedNftTransactions, p.mintedNftTransactions...)

	for _, nft := range p.burnedNftTransactions {
		dbBatchInsert.BurnedNft[nft.NftID] = true
	}
	dbBatchInsert.BurnedNftTransactions = append(dbBatchInsert.BurnedNftTransactions, p.burnedNftTransactions...)

	// Update object transfers
	for object, owner := range p.objectOwners {
		dbBatchInsert.ObjectNewOwners[object] = owner
		dbBatchInsert.TransferredNftTransactions = append(
			dbBatchInsert.TransferredNftTransactions,
			db.NewNftTransferTransaction(object, p.txProcessor.txData.ID, p.Height),
		)
	}

	dbBatchInsert.ModulePublishedEvents = append(dbBatchInsert.ModulePublishedEvents, p.modulePublishedEvents...)
	dbBatchInsert.CollectionMutationEvents = append(dbBatchInsert.CollectionMutationEvents, p.collectionMutationEvents...)
	dbBatchInsert.NftMutationEvents = append(dbBatchInsert.NftMutationEvents, p.nftMutationEvents...)

	return nil
}
