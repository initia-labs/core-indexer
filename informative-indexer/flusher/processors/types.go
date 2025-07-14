package processors

import (
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/initia/app/params"

	statetracker "github.com/initia-labs/core-indexer/informative-indexer/flusher/state-tracker"
)

type Processor interface {
	InitProcessor(height int64, validatorMap map[string]db.ValidatorAddress)
	Name() string
	ProcessBeginBlockEvents(finalizeBlockEvents *[]abci.Event) error
	ProcessEndBlockEvents(finalizeBlockEvents *[]abci.Event) error
	NewTxProcessor(txData *db.Transaction)
	ProcessSDKMessages(tx *mq.TxResult, encodingConfig *params.EncodingConfig) error
	ProcessTransactionEvents(tx *mq.TxResult) error
	ResolveTxProcessor() error
	TrackState(stateUpdateManager *statetracker.StateUpdateManager, dbBatchInsert *statetracker.DBBatchInsert) error
}
