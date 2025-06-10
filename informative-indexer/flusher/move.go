package flusher

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

const (
	ModulePublishedEventKey    = "0x1::code::ModulePublishedEvent"
	CreateCollectionEventKey   = "0x1::collection::CreateCollectionEvent"
	CollectionMutationEventKey = "0x1::collection::MutationEvent"
	ObjectCreatedEventKey      = "0x1::object::CreateEvent"
)

type CreateCollectionEvent struct {
	Collection string `json:"collection"`
	Creator    string `json:"creator"`
	Name       string `json:"name"`
}

type CollectionMutationEvent struct {
	Collection       string `json:"collection"`
	MutatedFieldName string `json:"mutated_field_name"`
	OldValue         string `json:"old_value"`
	NewValue         string `json:"new_value"`
}

type ObjectCreateEvent struct {
	Object string `json:"object"`
	Owner  string `json:"owner"`
}

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
	createCollectionEvents   []CreateCollectionEvent
	collectionMutationEvents []CollectionMutationEvent
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
		createCollectionEvents:   make([]CreateCollectionEvent, 0),
		collectionMutationEvents: make([]CollectionMutationEvent, 0),
	}
}

func (f *Flusher) processMoveEvents(blockResults *mq.BlockResultMsg) error {
	for _, tx := range blockResults.Txs {
		if tx.ExecTxResults.Log == "tx parse error" {
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
				TxID:        db.GetTxID(tx.Hash, blockResults.Height),
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
				TxID:               db.GetTxID(tx.Hash, blockResults.Height),
				NftID:              nil,
			})
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
			case ModulePublishedEventKey:
				p.handlePublishEvent(event)
			case CreateCollectionEventKey:
				p.handleCollectionCreateEvent(event)
			case CollectionMutationEventKey:
				p.handleCollectionMutationEvent(event)
				// case ObjectCreatedEventKey:
				// 	p.handleObjectCreateEvent(event)
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
			e, err := parser.DecodeEvent[CreateCollectionEvent](attr.Value)
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
			e, err := parser.DecodeEvent[CollectionMutationEvent](attr.Value)
			if err != nil {
				continue
			}
			p.collectionMutationEvents = append(p.collectionMutationEvents, e)
		}
	}
}
