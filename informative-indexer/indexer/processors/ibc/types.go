package ibc

import (
	"github.com/initia-labs/core-indexer/informative-indexer/indexer/processors"
	"github.com/initia-labs/core-indexer/pkg/db"
)

var _ processors.Processor = &Processor{}

type TxProcessor struct {
	txData *db.Transaction
}

type Processor struct {
	processors.BaseProcessor

	txProcessor *TxProcessor
}
