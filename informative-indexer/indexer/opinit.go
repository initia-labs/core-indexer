package indexer

import (
	"fmt"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/initia/app/params"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

type OpinitEventProcessor struct {
	txID          string
	relatedBridge map[uint64]bool
}

func NewOpinitEventProcessor(txID string) *OpinitEventProcessor {
	return &OpinitEventProcessor{
		txID:          txID,
		relatedBridge: make(map[uint64]bool),
	}
}

// processOpinitEvents processes OPinit events in a transaction
func (f *Indexer) processOpinitEvents(txResult *mq.TxResult, height int64, txData *db.Transaction) error {
	processor := NewOpinitEventProcessor(db.GetTxID(txResult.Hash, height))
	err := processor.processEvents(txResult, f.encodingConfig, txData)
	if err != nil {
		return fmt.Errorf("failed to process OPinit events: %w", err)
	}

	for bridgeID := range processor.relatedBridge {
		f.dbBatchInsert.OpinitTransactions = append(f.dbBatchInsert.OpinitTransactions, db.OpinitTransaction{
			TxID:        db.GetTxID(txResult.Hash, height),
			BridgeID:    int32(bridgeID),
			BlockHeight: int32(height),
		})
	}

	return nil
}

// processEvents processes all messages in a transaction
func (p *OpinitEventProcessor) processEvents(txResult *mq.TxResult, encodingConfig *params.EncodingConfig, txData *db.Transaction) error {
	sdkTx, err := encodingConfig.TxConfig.TxDecoder()(txResult.Tx)
	if err != nil {
		return fmt.Errorf("failed to decode SDK transaction: %w", err)
	}

	for _, msg := range sdkTx.GetMsgs() {
		switch m := msg.(type) {
		case *ophosttypes.MsgRecordBatch:
			txData.IsOpinit = true
			p.relatedBridge[m.BridgeId] = true
		case *ophosttypes.MsgCreateBridge:
			txData.IsOpinit = true
		case *ophosttypes.MsgProposeOutput:
			txData.IsOpinit = true
			p.relatedBridge[m.BridgeId] = true
		case *ophosttypes.MsgDeleteOutput:
			txData.IsOpinit = true
			p.relatedBridge[m.BridgeId] = true
		case *ophosttypes.MsgInitiateTokenDeposit:
			txData.IsOpinit = true
			p.relatedBridge[m.BridgeId] = true
		case *ophosttypes.MsgFinalizeTokenWithdrawal:
			txData.IsOpinit = true
			p.relatedBridge[m.BridgeId] = true
		case *ophosttypes.MsgUpdateProposer:
			txData.IsOpinit = true
			p.relatedBridge[m.BridgeId] = true
		case *ophosttypes.MsgUpdateChallenger:
			txData.IsOpinit = true
			p.relatedBridge[m.BridgeId] = true
		case *ophosttypes.MsgUpdateOracleConfig:
			txData.IsOpinit = true
			p.relatedBridge[m.BridgeId] = true
		case *ophosttypes.MsgUpdateBatchInfo:
			txData.IsOpinit = true
			p.relatedBridge[m.BridgeId] = true
		case *ophosttypes.MsgUpdateMetadata:
			txData.IsOpinit = true
			p.relatedBridge[m.BridgeId] = true
		case *ophosttypes.MsgUpdateParams:
			txData.IsOpinit = true
		}
	}

	return nil
}
