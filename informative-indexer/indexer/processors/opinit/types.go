package opinit

import (
	"github.com/initia-labs/core-indexer/informative-indexer/indexer/processors"
	"github.com/initia-labs/core-indexer/pkg/db"
)

var _ processors.Processor = &Processor{}

type TxProcessor struct {
	txData        *db.Transaction
	relatedBridge map[uint64]bool
}

type Processor struct {
	processors.BaseProcessor
	opinitTransactions []db.OpinitTransaction

	txProcessor *TxProcessor
}
