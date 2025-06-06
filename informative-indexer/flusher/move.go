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
	ModulePublishedEvent = "0x1::code::ModulePublishedEvent"
)

type MoveEventProcessor struct {
	modulesInTx        map[vmapi.ModuleInfoResponse]bool
	isPublish          bool
	isMoveExecuteEvent bool
	isMoveExecute      bool
	isMoveScript       bool

	newModules map[vmapi.ModuleInfoResponse]bool
}

func newMoveEventProcessor() *MoveEventProcessor {
	return &MoveEventProcessor{
		modulesInTx:        make(map[vmapi.ModuleInfoResponse]bool),
		isPublish:          false,
		isMoveExecuteEvent: false,
		isMoveExecute:      false,
		isMoveScript:       false,
		newModules:         make(map[vmapi.ModuleInfoResponse]bool),
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
			case ModulePublishedEvent:
				p.handlePublishEvent(event)
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
