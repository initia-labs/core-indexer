package account

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/cacher"
	statetracker "github.com/initia-labs/core-indexer/informative-indexer/indexer/state-tracker"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

func (p *Processor) InitProcessor(height int64, cacher *cacher.Cacher) {
	p.Height = height
	p.Cacher = cacher
	p.accounts = make(map[string]db.Account)
	p.accountsInTx = make(map[statetracker.AccountTxKey]db.AccountTransaction)
}

func (p *Processor) Name() string {
	return "account"
}

func (p *Processor) NewTxProcessor(txData *db.Transaction) {
	p.txProcessor = &TxProcessor{
		txData:      txData,
		relatedAccs: make([]sdk.AccAddress, 0),
	}
}

func (p *Processor) ProcessTransactionEvents(tx *mq.TxResult) error {
	relatedAccs, err := parser.GrepAddressesFromEvents(tx.ExecTxResults.Events)
	if err != nil {
		return err
	}
	p.txProcessor.relatedAccs = relatedAccs

	return nil
}

func (p *Processor) ResolveTxProcessor() error {
	for _, acc := range p.txProcessor.relatedAccs {
		account := db.NewAccountFromSDKAddress(acc)
		p.accounts[account.Address] = account

		accountTx := db.NewAccountTx(
			p.txProcessor.txData.ID,
			p.Height,
			account.Address,
			p.txProcessor.txData.Sender,
		)
		key := statetracker.MakeAccountTxKey(accountTx.TransactionID, accountTx.AccountID)
		p.accountsInTx[key] = accountTx
	}
	return nil
}

func (p *Processor) TrackState(stateUpdateManager *statetracker.StateUpdateManager, dbBatchInsert *statetracker.DBBatchInsert) error {
	dbBatchInsert.AddAccountsInTx(p.accounts, p.accountsInTx)
	return nil
}
