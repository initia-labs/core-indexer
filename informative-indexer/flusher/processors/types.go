package processors

import (
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/initia-labs/initia/app/params"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	statetracker "github.com/initia-labs/core-indexer/informative-indexer/flusher/state-tracker"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

type Processor interface {
	InitProcessor(height int64, validatorMap map[string]mstakingtypes.Validator)
	Name() string
	ProcessSDKMessages(txResult *mq.TxResult, encodingConfig *params.EncodingConfig) error
	ProcessTransactionEvents(txResult *mq.TxResult) error
	ProcessBeginBlockEvents(finalizedBlockEvents *[]abci.Event) error
	ProcessEndBlockEvents(finalizedBlockEvents *[]abci.Event) error
	TrackState(txHash string, stateUpdateManager *statetracker.StateUpdateManager, dbBatchInsert *statetracker.DBBatchInsert) error
}
