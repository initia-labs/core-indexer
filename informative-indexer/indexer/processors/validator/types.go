package validator

import (
	"github.com/initia-labs/core-indexer/informative-indexer/indexer/processors"
	"github.com/initia-labs/core-indexer/pkg/db"
)

var _ processors.Processor = &Processor{}

type TxProcessor struct {
	txData         *db.Transaction
	txStakeChanges map[string]int64
}

type Processor struct {
	processors.BaseProcessor
	stakeChanges []db.ValidatorBondedTokenChange
	validators   map[string]bool
	slashEvents  []db.ValidatorSlashEvent

	txProcessor *TxProcessor
}
