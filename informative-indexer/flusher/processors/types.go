package processors

import (
	stateTracker "github.com/initia-labs/core-indexer/informative-indexer/flusher/state-tracker"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

type Processor interface {
	InitProcessor()
	Name() string
	ProcessTxEvents(txResult *mq.TxResult, height int64, stateUpdateManager *stateTracker.StateUpdateManager, dbTx *db.Transaction) error
}
