package flusher

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"

	"github.com/initia-labs/core-indexer/informative-indexer/flusher/types"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

type MoveEventProcessor struct {
	modulesInTx        map[vmapi.ModuleInfoResponse]bool
	isPublish          bool
	isMoveExecuteEvent bool
	isMoveExecute      bool
	isMoveScript       bool
	isCollectionCreate bool
	isNftTransfer      bool
	isNftMint          bool
	isNftBurn          bool

	newModules               map[vmapi.ModuleInfoResponse]bool
	createCollectionEvents   []types.CreateCollectionEvent
	collectionMutationEvents []types.CollectionMutationEvent
	createdObjects           map[string]bool
	collectionMintEvents     map[string]types.CollectionMintEvent
	objectOwners             map[string]string
}

func newMoveEventProcessor() *MoveEventProcessor {
	return &MoveEventProcessor{
		modulesInTx:              make(map[vmapi.ModuleInfoResponse]bool),
		isPublish:                false,
		isMoveExecuteEvent:       false,
		isMoveExecute:            false,
		isMoveScript:             false,
		isCollectionCreate:       false,
		isNftTransfer:            false,
		isNftMint:                false,
		isNftBurn:                false,
		newModules:               make(map[vmapi.ModuleInfoResponse]bool),
		createCollectionEvents:   make([]types.CreateCollectionEvent, 0),
		collectionMutationEvents: make([]types.CollectionMutationEvent, 0),
		createdObjects:           make(map[string]bool),
		collectionMintEvents:     make(map[string]types.CollectionMintEvent),
		objectOwners:             make(map[string]string),
	}
}

func (f *Flusher) processMoveEvents(blockResults *mq.BlockResultMsg) error {
	for _, tx := range blockResults.Txs {
		if tx.ExecTxResults.Log == TxParseError {
			continue
		}

		processor := newMoveEventProcessor()
		if err := processor.processTransactionEvents(&tx); err != nil {
			return fmt.Errorf("failed to process transaction events: %w", err)
		}

		for module := range processor.newModules {
			// Only add the module if it hasn't been seen before to avoid overwriting
			// the original transaction hash that published the module
			if _, ok := f.stateUpdateManager.modules[module]; !ok {
				txID := db.GetTxID(tx.Hash, blockResults.Height)
				f.stateUpdateManager.modules[module] = &txID
			}
		}

		txID := db.GetTxID(tx.Hash, blockResults.Height)
		sdkTx, err := f.encodingConfig.TxConfig.TxDecoder()(tx.Tx)
		if err != nil {
			logger.Error().Msgf("Error decoding sdk tx: %v", err)
			return err
		}
		for _, msg := range sdkTx.GetMsgs() {
			switch msg := msg.(type) {
			case *movetypes.MsgExecute:
				processor.handleMoveExecuteEventIsEntry(msg.ModuleAddress, msg.ModuleName)
			case *movetypes.MsgExecuteJSON:
				processor.handleMoveExecuteEventIsEntry(msg.ModuleAddress, msg.ModuleName)
			}
		}

		for module, isEntry := range processor.modulesInTx {
			f.dbBatchInsert.moduleTransactions = append(f.dbBatchInsert.moduleTransactions, db.ModuleTransaction{
				IsEntry:     isEntry,
				BlockHeight: int32(blockResults.Height),
				TxID:        txID,
				ModuleID:    db.GetModuleID(module),
			})
		}

		for _, event := range processor.createCollectionEvents {
			f.stateUpdateManager.collectionsToUpdate[event.Collection] = true
			f.dbBatchInsert.collections[event.Collection] = db.Collection{
				ID:          event.Collection,
				Creator:     event.Creator,
				Name:        event.Name,
				BlockHeight: int32(blockResults.Height),
				URI:         "",
				Description: "",
			}
			f.dbBatchInsert.collectionTransactions = append(f.dbBatchInsert.collectionTransactions, db.CollectionTransaction{
				CollectionID:       event.Collection,
				IsCollectionCreate: true,
				BlockHeight:        int32(blockResults.Height),
				TxID:               txID,
				NftID:              nil,
			})
		}

		for _, mintedNft := range processor.collectionMintEvents {
			f.stateUpdateManager.nftsToUpdate[mintedNft.Nft] = true
			f.dbBatchInsert.nfts[mintedNft.Nft] = db.Nft{
				URI:         "",
				Description: "",
				TokenID:     mintedNft.TokenID,
				Remark:      db.JSON("{}"),
				ProposalID:  nil,
				TxID:        &txID,
				Owner:       mintedNft.Nft, // temporary owner will be updated in the next step
				ID:          mintedNft.Nft,
				Collection:  mintedNft.Collection,
				IsBurned:    false,
			}
			f.dbBatchInsert.mintedNftTransactions = append(f.dbBatchInsert.mintedNftTransactions, db.NewNftMintTransaction(mintedNft.Nft, txID, int32(blockResults.Height)))
		}

		for object, owner := range processor.objectOwners {
			f.dbBatchInsert.objectNewOwners[object] = owner
			f.dbBatchInsert.transferredNftTransactions = append(f.dbBatchInsert.transferredNftTransactions, db.NewNftTransferTransaction(object, txID, int32(blockResults.Height)))
		}
	}
	return nil
}

func (p *MoveEventProcessor) processTransactionEvents(tx *mq.TxResult) error {
	for _, event := range tx.ExecTxResults.Events {
		p.handleEvent(event)
	}
	return nil
}

func (p *MoveEventProcessor) handleEvent(event abci.Event) {
	switch event.Type {
	case sdk.EventTypeMessage:
		p.handleMessageEvent(event)
	case movetypes.EventTypeMove:
		p.handleMoveEvent(event)
	case movetypes.EventTypeExecute:
		p.handleMoveExecuteEvent(event)
	}
}

func (p *MoveEventProcessor) handleMessageEvent(event abci.Event) {
	for _, attr := range event.Attributes {
		if attr.Key != sdk.AttributeKeyAction {
			continue
		}
		p.setMessageTypeFlags(attr.Value)
	}
}

func (p *MoveEventProcessor) setMessageTypeFlags(msgTypeURL string) {
	switch msgTypeURL {
	case sdk.MsgTypeURL(&movetypes.MsgPublish{}):
		p.isPublish = true
	case sdk.MsgTypeURL(&movetypes.MsgExecute{}):
		p.isMoveExecute = true
	case sdk.MsgTypeURL(&movetypes.MsgScript{}):
		p.isMoveScript = true
	}
}

func (p *MoveEventProcessor) handleMoveEvent(event abci.Event) {
	for _, attr := range event.Attributes {
		if attr.Key == movetypes.AttributeKeyTypeTag {
			switch attr.Value {
			case types.ModulePublishedEventKey:
				p.handlePublishEvent(event)
			case types.CreateCollectionEventKey:
				p.handleCollectionCreateEvent(event)
			case types.CollectionMutationEventKey:
				p.handleCollectionMutationEvent(event)
			case types.CollectionMintEventKey:
				p.handleCollectionMintEvent(event)
			case types.ObjectTransferEventKey:
				p.handleObjectTransferEvent(event)
			}
		}
	}
}

func (p *MoveEventProcessor) handlePublishEvent(event abci.Event) {
	p.isPublish = true
	for _, attr := range event.Attributes {
		if attr.Key == movetypes.AttributeKeyData {
			module, _, err := parser.DecodePublishModuleData(attr.Value)
			if err != nil {
				continue
			}
			p.newModules[module] = true
		}
	}
}

func (p *MoveEventProcessor) handleMoveExecuteEvent(event abci.Event) {
	moduleAddrs := make([]string, 0)
	moduleNames := make([]string, 0)
	for _, attr := range event.Attributes {
		if attr.Key == movetypes.AttributeKeyModuleAddr {
			moduleAddrs = append(moduleAddrs, attr.Value)
		}
		if attr.Key == movetypes.AttributeKeyModuleName {
			moduleNames = append(moduleNames, attr.Value)
		}
	}

	for idx, moduleAddr := range moduleAddrs {
		addr, _ := vmtypes.NewAccountAddress(moduleAddr)
		p.modulesInTx[vmapi.ModuleInfoResponse{Address: addr, Name: moduleNames[idx]}] = false
	}
}

func (p *MoveEventProcessor) handleMoveExecuteEventIsEntry(moduleAddress, moduleName string) error {
	vmAddr, err := vmtypes.NewAccountAddress(moduleAddress)
	if err != nil {
		logger.Error().Msgf("Error converting module address: %v", err)
		return err
	}
	p.modulesInTx[vmapi.ModuleInfoResponse{Address: vmAddr, Name: moduleName}] = true
	return nil
}

func (p *MoveEventProcessor) handleCollectionCreateEvent(event abci.Event) {
	p.isCollectionCreate = true
	for _, attr := range event.Attributes {
		if attr.Key == movetypes.AttributeKeyData {
			e, err := parser.DecodeEvent[types.CreateCollectionEvent](attr.Value)
			if err != nil {
				continue
			}
			p.createCollectionEvents = append(p.createCollectionEvents, e)
		}
	}
}

func (p *MoveEventProcessor) handleCollectionMutationEvent(event abci.Event) {
	for _, attr := range event.Attributes {
		if attr.Key == movetypes.AttributeKeyData {
			e, err := parser.DecodeEvent[types.CollectionMutationEvent](attr.Value)
			if err != nil {
				continue
			}
			p.collectionMutationEvents = append(p.collectionMutationEvents, e)
		}
	}
}

func (p *MoveEventProcessor) handleCollectionMintEvent(event abci.Event) {
	for _, attr := range event.Attributes {
		if attr.Key == movetypes.AttributeKeyData {
			e, err := parser.DecodeEvent[types.CollectionMintEvent](attr.Value)
			if err != nil {
				continue
			}
			p.collectionMintEvents[e.Nft] = e
		}
	}
}

func (p *MoveEventProcessor) handleObjectTransferEvent(event abci.Event) {
	for _, attr := range event.Attributes {
		if attr.Key == movetypes.AttributeKeyData {
			e, err := parser.DecodeEvent[types.ObjectTransferEvent](attr.Value)
			if err != nil {
				continue
			}
			p.objectOwners[e.Object] = e.From
		}
	}
}
