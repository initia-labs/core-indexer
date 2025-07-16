package account

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/processors"
	statetracker "github.com/initia-labs/core-indexer/informative-indexer/indexer/state-tracker"
	"github.com/initia-labs/core-indexer/pkg/db"
)

var _ processors.Processor = &Processor{}

type TxProcessor struct {
	txData      *db.Transaction
	relatedAccs []sdk.AccAddress
	sender      sdk.AccAddress
}

type Processor struct {
	processors.BaseProcessor
	accounts     map[string]db.Account
	accountsInTx map[statetracker.AccountTxKey]db.AccountTransaction

	txProcessor *TxProcessor
}
