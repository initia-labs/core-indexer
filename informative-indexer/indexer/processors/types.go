package processors

import (
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/initia/app/params"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/cacher"
	statetracker "github.com/initia-labs/core-indexer/informative-indexer/indexer/state-tracker"
)

type Processor interface {
	InitProcessor(height int64, cacher *cacher.Cacher)
	Name() string
	ProcessBeginBlockEvents(finalizeBlockEvents *[]abci.Event) error
	ProcessEndBlockEvents(finalizeBlockEvents *[]abci.Event) error
	NewTxProcessor(txData *db.Transaction)
	ProcessSDKMessages(tx *mq.TxResult, encodingConfig *params.EncodingConfig) error
	ProcessTransactionEvents(tx *mq.TxResult) error
	ResolveTxProcessor() error
	TrackState(stateUpdateManager *statetracker.StateUpdateManager, dbBatchInsert *statetracker.DBBatchInsert) error
}

var _ Processor = &BaseProcessor{}

type BaseProcessor struct {
	Height int64
	Cacher *cacher.Cacher
}

func (p *BaseProcessor) InitProcessor(height int64, cacher *cacher.Cacher) {
	p.Height = height
	p.Cacher = cacher
}

func (p *BaseProcessor) Name() string {
	return "base"
}

func (p *BaseProcessor) ProcessBeginBlockEvents(finalizeBlockEvents *[]abci.Event) error {
	return nil
}

func (p *BaseProcessor) ProcessEndBlockEvents(finalizeBlockEvents *[]abci.Event) error {
	return nil
}

func (p *BaseProcessor) NewTxProcessor(txData *db.Transaction) {
}

func (p *BaseProcessor) ProcessSDKMessages(tx *mq.TxResult, encodingConfig *params.EncodingConfig) error {
	return nil
}

func (p *BaseProcessor) ProcessTransactionEvents(tx *mq.TxResult) error {
	return nil
}

func (p *BaseProcessor) ResolveTxProcessor() error {
	return nil
}

func (p *BaseProcessor) TrackState(stateUpdateManager *statetracker.StateUpdateManager, dbBatchInsert *statetracker.DBBatchInsert) error {
	return nil
}
