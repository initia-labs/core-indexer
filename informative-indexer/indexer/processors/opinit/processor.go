package opinit

import (
	"fmt"

	"github.com/initia-labs/initia/app/params"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/cacher"
	statetracker "github.com/initia-labs/core-indexer/informative-indexer/indexer/state-tracker"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

func (p *Processor) InitProcessor(height int64, cacher *cacher.Cacher) {
	p.Height = height
	p.Cacher = cacher
	p.txProcessor = nil
	p.opinitTransactions = make([]db.OpinitTransaction, 0)
}

func (p *Processor) Name() string {
	return "opinit"
}

func (p *Processor) NewTxProcessor(txData *db.Transaction) {
	p.txProcessor = &TxProcessor{
		txData:        txData,
		relatedBridge: make(map[uint64]bool),
	}
}

// processOpinitEvents processes OPinit events in a transaction
func (p *Processor) ProcessSDKMessages(tx *mq.TxResult, encodingConfig *params.EncodingConfig) error {
	sdkTx, err := encodingConfig.TxConfig.TxDecoder()(tx.Tx)
	if err != nil {
		return fmt.Errorf("failed to decode SDK transaction: %w", err)
	}

	for _, msg := range sdkTx.GetMsgs() {
		p.handleMsg(msg)
	}

	return nil
}

func (p *Processor) ResolveTxProcessor() error {
	for bridgeID := range p.txProcessor.relatedBridge {
		p.opinitTransactions = append(p.opinitTransactions, db.OpinitTransaction{
			TxID:        p.txProcessor.txData.ID,
			BridgeID:    int32(bridgeID),
			BlockHeight: p.Height,
		})
	}
	return nil
}

func (p *Processor) TrackState(stateUpdateManager *statetracker.StateUpdateManager, dbBatchInsert *statetracker.DBBatchInsert) error {
	dbBatchInsert.OpinitTransactions = append(dbBatchInsert.OpinitTransactions, p.opinitTransactions...)
	return nil
}
