package flusher

import (
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

type OpinitEventProcessor struct {
	txID string
}

var opinitEventTypes = map[string]bool{
	ophosttypes.EventTypeRecordBatch:             true,
	ophosttypes.EventTypeCreateBridge:            true,
	ophosttypes.EventTypeProposeOutput:           true,
	ophosttypes.EventTypeDeleteOutput:            true,
	ophosttypes.EventTypeInitiateTokenDeposit:    true,
	ophosttypes.EventTypeFinalizeTokenWithdrawal: true,
	ophosttypes.EventTypeUpdateProposer:          true,
	ophosttypes.EventTypeUpdateChallenger:        true,
	ophosttypes.EventTypeUpdateBatchInfo:         true,
	ophosttypes.EventTypeUpdateMetadata:          true,
	ophosttypes.EventTypeUpdateOracle:            true,
}

func newOpinitEventProcessor(txID string) *OpinitEventProcessor {
	return &OpinitEventProcessor{
		txID: txID,
	}
}

func (f *Flusher) processOpinitEvents(txResult *mq.TxResult, height int64, txData *db.Transaction) error {
	processor := newOpinitEventProcessor(db.GetTxID(txResult.Hash, height))
	if err := processor.processTransactionEvents(txResult, txData); err != nil {
		return err
	}
	return nil
}

func (p *OpinitEventProcessor) processTransactionEvents(tx *mq.TxResult, txData *db.Transaction) error {
	for _, event := range tx.ExecTxResults.Events {
		if opinitEventTypes[event.Type] {
			txData.IsOpinit = true
		}
	}
	return nil
}
