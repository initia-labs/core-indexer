package ibc

import (
	"fmt"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/cacher"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

func (p *Processor) InitProcessor(height int64, cacher *cacher.Cacher) {
	p.Height = height
	p.Cacher = cacher
	p.txProcessor = nil
}

func (p *Processor) Name() string {
	return "ibc"
}

func (p *Processor) NewTxProcessor(txData *db.Transaction) {
	p.txProcessor = &TxProcessor{
		txData: txData,
	}
}

func (p *Processor) ProcessTransactionEvents(tx *mq.TxResult) error {
	for _, event := range tx.ExecTxResults.Events {
		if err := p.handleEvent(event); err != nil {
			return fmt.Errorf("failed to handle tx event %s: %w", event.Type, err)
		}
	}
	return nil
}
