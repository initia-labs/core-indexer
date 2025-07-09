package processors

import (
	"github.com/initia-labs/initia/app/params"

	statetracker "github.com/initia-labs/core-indexer/informative-indexer/flusher/state-tracker"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

type Processor interface {
	InitProcessor()
	Name() string
	ProcessSDKMessages(txResult *mq.TxResult, height int64, encodingConfig *params.EncodingConfig) error
	ProcessTransactionEvents(txResult *mq.TxResult) error
	TrackState(txHash string, blockHeight int64, stateUpdateManager *statetracker.StateUpdateManager) error
}
