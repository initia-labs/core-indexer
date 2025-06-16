package flusher

import (
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

type IbcEventProcessor struct {
	txID string
}

func newIbcEventProcessor(txID string) *IbcEventProcessor {
	return &IbcEventProcessor{
		txID: txID,
	}
}

// processMoveEvents processes all Move events in a block, handling multiple transactions
// and their associated events. It maintains state updates and batch inserts for the database.
func (f *Flusher) processIbcEvents(txResult *mq.TxResult, height int64, txData *db.Transaction) error {
	processor := newIbcEventProcessor(db.GetTxID(txResult.Hash, height))
	// Step 1: Process all events in the transaction
	if err := processor.processTransactionEvents(txResult, txData); err != nil {
		return err
	}

	// // Step 3: Update state and database based on processed data
	// if err := f.updateStateFromMoveProcessor(processor, blockResults.Height); err != nil {
	// 	return err
	// }
	return nil
}

func (p *IbcEventProcessor) processTransactionEvents(tx *mq.TxResult, txData *db.Transaction) error {
	for _, event := range tx.ExecTxResults.Events {
		switch event.Type {
		case sdk.EventTypeMessage:
			p.handleMessageEvents(event, txData)
		case ibcchanneltypes.EventTypeSendPacket:
			txData.IsIbc = true
		}
	}
	return nil
}

func (p *IbcEventProcessor) handleMessageEvents(event abci.Event, txData *db.Transaction) {
	for _, attribute := range event.Attributes {
		if attribute.Key == sdk.AttributeKeyAction {
			if strings.HasPrefix(attribute.Value, "/ibc") {
				txData.IsIbc = true
			}
		}
	}
}
