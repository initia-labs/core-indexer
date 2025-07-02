package flusher

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/initia-labs/initia/app/params"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"

	"github.com/initia-labs/core-indexer/informative-indexer/flusher/types"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

// MoveEventProcessor handles the processing of Move-related events in a transaction.
// A single transaction can contain multiple events of different types, which is why
// we use boolean flags to track the presence of each event type.
type MoveEventProcessor struct {
	// Event type flags - a transaction can have multiple event types
	modulesInTx map[vmapi.ModuleInfoResponse]bool

	// Event data collections
	newModules               map[vmapi.ModuleInfoResponse]bool
	modulePublishedEvents    []db.ModuleHistory
	createCollectionEvents   []types.CreateCollectionEvent
	collectionMutationEvents []types.CollectionMutationEvent
	nftMutationEvents        []types.NftMutationEvent
	createdObjects           map[string]bool
	collectionMintEvents     map[string]types.CollectionMintEvent
	collectionBurnEvents     map[string]types.CollectionBurnEvent
	objectOwners             map[string]string
	TxID                     string
}

// newMoveEventProcessor creates a new MoveEventProcessor instance with initialized maps and slices
func newMoveEventProcessor(txID string) *MoveEventProcessor {
	return &MoveEventProcessor{
		modulesInTx:              make(map[vmapi.ModuleInfoResponse]bool),
		newModules:               make(map[vmapi.ModuleInfoResponse]bool),
		modulePublishedEvents:    make([]db.ModuleHistory, 0),
		createCollectionEvents:   make([]types.CreateCollectionEvent, 0),
		collectionMutationEvents: make([]types.CollectionMutationEvent, 0),
		nftMutationEvents:        make([]types.NftMutationEvent, 0),
		createdObjects:           make(map[string]bool),
		collectionMintEvents:     make(map[string]types.CollectionMintEvent),
		collectionBurnEvents:     make(map[string]types.CollectionBurnEvent),
		objectOwners:             make(map[string]string),
		TxID:                     txID,
	}
}

// processMoveEvents processes all Move events in a block, handling multiple transactions
// and their associated events. It maintains state updates and batch inserts for the database.
func (f *Flusher) processMoveEvents(txResult *mq.TxResult, height int64, txData *db.Transaction) error {
	processor := newMoveEventProcessor(db.GetTxID(txResult.Hash, height))
	// Step 1: Process all events in the transaction
	if err := processor.processTransactionEvents(txResult, txData); err != nil {
		return fmt.Errorf("failed to process transaction events: %w", err)
	}

	// Step 2: Process SDK messages to identify entry points
	if err := processor.processSDKMessages(*txResult, f.encodingConfig, txData); err != nil {
		return err
	}

	// Step 3: Update state and database based on processed data
	if err := f.updateStateFromMoveProcessor(processor, height); err != nil {
		return err
	}
	return nil
}

// updateStateFromMoveProcessor updates state and database based on processed event data
func (f *Flusher) updateStateFromMoveProcessor(processor *MoveEventProcessor, height int64) error {
	// Update modules state
	for module := range processor.newModules {
		if _, ok := f.stateUpdateManager.modules[module]; !ok {
			txID := processor.TxID
			f.stateUpdateManager.modules[module] = &txID
		}
	}

	// Update module transactions
	for module, isEntry := range processor.modulesInTx {
		// use for test only
		if _, ok := f.stateUpdateManager.modules[module]; !ok {
			txID := processor.TxID
			f.stateUpdateManager.modules[module] = &txID
		}
		f.dbBatchInsert.moduleTransactions = append(f.dbBatchInsert.moduleTransactions, db.ModuleTransaction{
			IsEntry:     isEntry,
			BlockHeight: int32(height),
			TxID:        processor.TxID,
			ModuleID:    db.GetModuleID(module),
		})
	}

	// Update collections
	for _, event := range processor.createCollectionEvents {
		f.stateUpdateManager.collectionsToUpdate[event.Collection] = true
		f.dbBatchInsert.collections[event.Collection] = db.Collection{
			ID:          event.Collection,
			Creator:     event.Creator,
			Name:        event.Name,
			BlockHeight: int32(height),
			URI:         "",
			Description: "",
		}

		// TODO: improve this
		f.dbBatchInsert.collectionTransactions = append(f.dbBatchInsert.collectionTransactions, db.CollectionTransaction{
			CollectionID:       event.Collection,
			IsCollectionCreate: true,
			BlockHeight:        int32(height),
			TxID:               processor.TxID,
			NftID:              nil,
		})
	}

	// Update NFTs
	for _, mintedNft := range processor.collectionMintEvents {
		f.stateUpdateManager.nftsToUpdate[mintedNft.Nft] = true
		txID := processor.TxID
		f.dbBatchInsert.nfts[mintedNft.Nft] = db.Nft{
			TokenID:    mintedNft.TokenID,
			Remark:     db.JSON("{}"),
			ProposalID: nil,
			TxID:       &txID,
			ID:         mintedNft.Nft,
			Collection: mintedNft.Collection,
			IsBurned:   false,
			Owner:       processor.objectOwners[mintedNft.Nft],

			// temporary fields
			Description: "",
			URI:         "",
		}
		f.dbBatchInsert.mintedNftTransactions = append(
			f.dbBatchInsert.mintedNftTransactions,
			db.NewNftMintTransaction(mintedNft.Nft, processor.TxID, int32(height)),
		)

		// TODO: improve this
		f.dbBatchInsert.collectionTransactions = append(f.dbBatchInsert.collectionTransactions, db.CollectionTransaction{
			CollectionID: mintedNft.Collection,
			IsNftMint:    true,
			BlockHeight:  int32(height),
			TxID:         processor.TxID,
			NftID:        &mintedNft.Nft,
		})
	}

	for _, burnEvent := range processor.collectionBurnEvents {
		f.dbBatchInsert.burnedNft[burnEvent.Nft] = true
		f.dbBatchInsert.collectionTransactions = append(f.dbBatchInsert.collectionTransactions, db.CollectionTransaction{
			CollectionID: burnEvent.Collection,
			IsNftBurn:    true,
			BlockHeight:  int32(height),
			TxID:         processor.TxID,
			NftID:        &burnEvent.Nft,
		})
		f.dbBatchInsert.nftBurnTransactions = append(f.dbBatchInsert.nftBurnTransactions, db.NewNftBurnTransaction(burnEvent.Nft, processor.TxID, int32(height)))
	}

	// Update object transfers
	for object, owner := range processor.objectOwners {
		f.dbBatchInsert.objectNewOwners[object] = owner
		f.dbBatchInsert.transferredNftTransactions = append(
			f.dbBatchInsert.transferredNftTransactions,
			db.NewNftTransferTransaction(object, processor.TxID, int32(height)),
		)
	}

	for _, event := range processor.collectionMutationEvents {
		f.dbBatchInsert.collectionMutationEvents = append(f.dbBatchInsert.collectionMutationEvents, db.CollectionMutationEvent{
			CollectionID:     event.Collection,
			MutatedFieldName: event.MutatedFieldName,
			OldValue:         event.OldValue,
			NewValue:         event.NewValue,
			Remark:           db.JSON("{}"),
			TxID:             processor.TxID,
			BlockHeight:      int32(height),
		})
	}
	for _, event := range processor.nftMutationEvents {
		f.dbBatchInsert.nftMutationEvents = append(f.dbBatchInsert.nftMutationEvents, db.NftMutationEvent{
			NftID:            event.Nft,
			MutatedFieldName: event.MutatedFieldName,
			OldValue:         event.OldValue,
			NewValue:         event.NewValue,
			Remark:           db.JSON("{}"),
			TxID:             processor.TxID,
			BlockHeight:      int32(height),
		})
	}

	f.dbBatchInsert.modulePublishedEvents = append(f.dbBatchInsert.modulePublishedEvents, processor.modulePublishedEvents...)
	return nil
}

// processSDKMessages processes SDK transaction messages to identify entry points
func (processor *MoveEventProcessor) processSDKMessages(tx mq.TxResult, encodingConfig *params.EncodingConfig, txData *db.Transaction) error {
	sdkTx, err := encodingConfig.TxConfig.TxDecoder()(tx.Tx)
	if err != nil {
		return fmt.Errorf("failed to decode SDK transaction: %w", err)
	}

	for _, msg := range sdkTx.GetMsgs() {
		switch msg := msg.(type) {
		case *movetypes.MsgExecute:
			txData.IsMoveExecute = true
			if err := processor.handleMoveExecuteEventIsEntry(msg.ModuleAddress, msg.ModuleName); err != nil && tx.ExecTxResults.IsOK() {
				return fmt.Errorf("failed to process MsgExecute: %w", err)
			}
		case *movetypes.MsgExecuteJSON:
			txData.IsMoveExecute = true
			if err := processor.handleMoveExecuteEventIsEntry(msg.ModuleAddress, msg.ModuleName); err != nil && tx.ExecTxResults.IsOK() {
				return fmt.Errorf("failed to process MsgExecuteJSON: %w", err)
			}
		case *movetypes.MsgScript:
			txData.IsMoveScript = true
		}
	}

	return nil
}

// processTransactionEvents processes all events in a transaction, routing them to appropriate handlers
func (p *MoveEventProcessor) processTransactionEvents(tx *mq.TxResult, txData *db.Transaction) error {
	for _, event := range tx.ExecTxResults.Events {
		if err := p.handleEvent(event, txData); err != nil {
			return fmt.Errorf("failed to handle event: %w", err)
		}
	}
	return nil
}

// handleEvent routes events to appropriate handlers based on event type
func (p *MoveEventProcessor) handleEvent(event abci.Event, txData *db.Transaction) error {
	switch event.Type {
	case movetypes.EventTypeMove:
		return p.handleMoveEvent(event, txData)
	case movetypes.EventTypeExecute:
		return p.handleMoveExecuteEvent(event, txData)
	default:
		return nil
	}
}

// handleMoveEvent processes Move-specific events, routing them to appropriate handlers
func (p *MoveEventProcessor) handleMoveEvent(event abci.Event, txData *db.Transaction) error {
	if value, found := findAttribute(event.Attributes, movetypes.AttributeKeyTypeTag); found {
		switch value {
		case types.ModulePublishedEventKey:
			return p.handlePublishEvent(event, txData)
		case types.CreateCollectionEventKey:
			return p.handleCollectionCreateEvent(event, txData)
		case types.CollectionMutationEventKey:
			return p.handleCollectionMutationEvent(event, txData)
		case types.NftMutationEventKey:
			return p.handleNftMutationEvent(event, txData)
		case types.CollectionMintEventKey:
			return p.handleCollectionMintEvent(event, txData)
		case types.ObjectTransferEventKey:
			return p.handleObjectTransferEvent(event, txData)
		case types.CollectionBurnEventKey:
			return p.handleCollectionBurnEvent(event, txData)
		case types.ObjectCreateEventKey:
			return p.handleObjectCreateEvent(event, txData)
		}
	}
	return nil
}

// handlePublishEvent processes module publish events, recording new modules
func (p *MoveEventProcessor) handlePublishEvent(event abci.Event, txData *db.Transaction) error {
	txData.IsMovePublish = true
	if value, found := findAttribute(event.Attributes, movetypes.AttributeKeyData); found {
		module, upgradePolicy, err := parser.DecodePublishModuleData(value)
		if err != nil {
			return fmt.Errorf("failed to decode publish module data: %w", err)
		}
		p.newModules[module] = true
		p.modulePublishedEvents = append(p.modulePublishedEvents, db.ModuleHistory{
			ModuleID:      db.GetModuleID(module),
			BlockHeight:   int32(txData.BlockHeight),
			Remark:        db.JSON("{}"),
			ProposalID:    nil,
			TxID:          &txData.ID,
			UpgradePolicy: db.GetUpgradePolicy(upgradePolicy),
		})
	}
	return nil
}

// handleMoveExecuteEvent processes Move execution events, recording module executions
func (p *MoveEventProcessor) handleMoveExecuteEvent(event abci.Event, _ *db.Transaction) error {
	moduleAddrs := make([]string, 0)
	moduleNames := make([]string, 0)

	for _, attr := range event.Attributes {
		switch attr.Key {
		case movetypes.AttributeKeyModuleAddr:
			moduleAddrs = append(moduleAddrs, attr.Value)
		case movetypes.AttributeKeyModuleName:
			moduleNames = append(moduleNames, attr.Value)
		}
	}

	for idx, moduleAddr := range moduleAddrs {
		addr, err := vmtypes.NewAccountAddress(moduleAddr)
		if err != nil {
			return fmt.Errorf("invalid module address: %w", err)
		}
		p.modulesInTx[vmapi.ModuleInfoResponse{Address: addr, Name: moduleNames[idx]}] = false
	}
	return nil
}

// handleMoveExecuteEventIsEntry processes entry point execution events,
// marking modules as entry points when they are executed directly
func (p *MoveEventProcessor) handleMoveExecuteEventIsEntry(moduleAddress, moduleName string) error {
	vmAddr, err := vmtypes.NewAccountAddress(moduleAddress)
	if err != nil {
		return fmt.Errorf("invalid module address: %w", err)
	}
	p.modulesInTx[vmapi.ModuleInfoResponse{Address: vmAddr, Name: moduleName}] = true
	return nil
}

// handleCollectionCreateEvent processes collection creation events
func (p *MoveEventProcessor) handleCollectionCreateEvent(event abci.Event, txData *db.Transaction) error {
	return handleEventWithKey(event, movetypes.AttributeKeyData, &txData.IsCollectionCreate, func(e types.CreateCollectionEvent) {
		p.createCollectionEvents = append(p.createCollectionEvents, e)
	})
}

// handleCollectionMutationEvent processes collection mutation events
func (p *MoveEventProcessor) handleCollectionMutationEvent(event abci.Event, _ *db.Transaction) error {
	return handleEventWithKey(event, movetypes.AttributeKeyData, nil, func(e types.CollectionMutationEvent) {
		p.collectionMutationEvents = append(p.collectionMutationEvents, e)
	})
}

func (p *MoveEventProcessor) handleNftMutationEvent(event abci.Event, _ *db.Transaction) error {
	return handleEventWithKey(event, movetypes.AttributeKeyData, nil, func(e types.NftMutationEvent) {
		p.nftMutationEvents = append(p.nftMutationEvents, e)
	})
}

// handleCollectionMintEvent processes NFT minting events
func (p *MoveEventProcessor) handleCollectionMintEvent(event abci.Event, txData *db.Transaction) error {
	return handleEventWithKey(event, movetypes.AttributeKeyData, &txData.IsNftMint, func(e types.CollectionMintEvent) {
		p.collectionMintEvents[e.Nft] = e
	})
}

// handleObjectTransferEvent processes object transfer events
func (p *MoveEventProcessor) handleObjectTransferEvent(event abci.Event, txData *db.Transaction) error {
	return handleEventWithKey(event, movetypes.AttributeKeyData, &txData.IsNftTransfer, func(e types.ObjectTransferEvent) {
		p.objectOwners[e.Object] = e.To
	})
}

func (p *MoveEventProcessor) handleCollectionBurnEvent(event abci.Event, txData *db.Transaction) error {
	return handleEventWithKey(event, movetypes.AttributeKeyData, &txData.IsNftBurn, func(e types.CollectionBurnEvent) {
		p.collectionBurnEvents[e.Nft] = e
	})
}

func (p *MoveEventProcessor) handleObjectCreateEvent(event abci.Event, _ *db.Transaction) error {
	return handleEventWithKey(event, movetypes.AttributeKeyData, nil, func(e types.ObjectCreateEvent) {
		p.objectOwners[e.Object] = e.Owner
	})
}
