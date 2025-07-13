package validator

import (
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	"github.com/initia-labs/core-indexer/pkg/db"
)

type TxProcessor struct {
	txData         *db.Transaction
	txStakeChanges map[string]int64
}

type Processor struct {
	height       int64
	validatorMap map[string]mstakingtypes.Validator
	stakeChanges []db.ValidatorBondedTokenChange
	validators   map[string]bool
	slashEvents  []db.ValidatorSlashEvent

	txProcessor *TxProcessor
}
