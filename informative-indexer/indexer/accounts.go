package indexer

import (
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

func (f *Indexer) processAccounts(tx *mq.TxResult, height int64, _ *db.Transaction) error {
	relatedAccs, err := parser.GrepAddressesFromEvents(tx.ExecTxResults.Events)
	if err != nil {
		return err
	}

	sender, err := parser.GrepSenderFromEvents(tx.ExecTxResults.Events)
	if err != nil {
		return err
	}

	accounts := make([]db.Account, 0, len(relatedAccs))
	for _, acc := range relatedAccs {
		accounts = append(accounts, db.NewAccountFromSDKAddress(acc))
	}

	f.dbBatchInsert.AddAccountsInTx(
		tx.Hash,
		height,
		sender.String(),
		accounts...,
	)
	return nil
}
