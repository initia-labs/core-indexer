package flusher

import (
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

func (f *Flusher) processAccounts(blockResults *mq.BlockResultMsg) error {
	for _, tx := range blockResults.Txs {
		if tx.ExecTxResults.Log == TxParseError {
			continue
		}

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
			blockResults.Height,
			sender.String(),
			accounts...,
		)
	}
	return nil
}
