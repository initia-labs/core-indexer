package account

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	statetracker "github.com/initia-labs/core-indexer/informative-indexer/indexer/state-tracker"
	"github.com/initia-labs/core-indexer/pkg/db"
)

type TxProcessor struct {
	txData      *db.Transaction
	relatedAccs []sdk.AccAddress
	sender      sdk.AccAddress
}

type Processor struct {
	height       int64
	validatorMap map[string]db.ValidatorAddress
	accounts     map[string]db.Account
	accountsInTx map[statetracker.AccountTxKey]db.AccountTransaction

	txProcessor *TxProcessor
}
