package account

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/app/params"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/processors"
	statetracker "github.com/initia-labs/core-indexer/informative-indexer/indexer/state-tracker"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

var _ processors.Processor = &Processor{}

func (p *Processor) InitProcessor(height int64, validatorMap map[string]db.ValidatorAddress) {
	p.height = height
	p.validatorMap = validatorMap
	p.accounts = make(map[string]db.Account)
	p.accountsInTx = make(map[statetracker.AccountTxKey]db.AccountTransaction)
}

func (p *Processor) Name() string {
	return "account"
}

func (p *Processor) ProcessBeginBlockEvents(finalizeBlockEvents *[]abci.Event) error {
	return nil
}

func (p *Processor) ProcessEndBlockEvents(finalizeBlockEvents *[]abci.Event) error {
	return nil
}

func (p *Processor) NewTxProcessor(txData *db.Transaction) {
	p.txProcessor = &TxProcessor{
		txData:      txData,
		relatedAccs: make([]types.AccAddress, 0),
		sender:      nil,
	}
}

func (p *Processor) ProcessSDKMessages(tx *mq.TxResult, encodingConfig *params.EncodingConfig) error {
	return nil
}

func (p *Processor) ProcessTransactionEvents(tx *mq.TxResult) error {
	for _, event := range tx.ExecTxResults.Events {
		if err := p.handleEvent(event); err != nil {
			return fmt.Errorf("failed to handle tx event %s: %w", event.Type, err)
		}
	}
	return nil
}

func (p *Processor) ResolveTxProcessor() error {
	if p.txProcessor.sender == nil {
		return fmt.Errorf("sender not found")
	}

	for _, acc := range p.txProcessor.relatedAccs {
		account := db.NewAccountFromSDKAddress(acc)
		p.accounts[account.Address] = account

		accountTx := db.NewAccountTx(
			p.txProcessor.txData.ID,
			p.height,
			account.Address,
			p.txProcessor.sender.String(),
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
