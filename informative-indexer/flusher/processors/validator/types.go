package validator

import (
	"github.com/initia-labs/core-indexer/pkg/db"
)

type TxProcessor struct {
	txData         *db.Transaction
	txStakeChanges map[string]int64
}

type Processor struct {
	height       int64
	validatorMap map[string]db.ValidatorAddress
	stakeChanges []db.ValidatorBondedTokenChange
	validators   map[string]bool
	slashEvents  []db.ValidatorSlashEvent

	txProcessor *TxProcessor
}
