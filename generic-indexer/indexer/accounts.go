package indexer

import (
	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

func (f *Indexer) getAccountTransactions(txHash string, height int64, events []abci.Event, signer string) (map[string]db.Account, []db.AccountTransaction, error) {
	relatedAccs, err := parser.GrepAddressesFromEvents(events)
	if err != nil {
		logger.Error().Msgf("Error grep addresses from tx: %v", err)
		return nil, nil, err
	}

	accs := make(map[string]db.Account)
	for _, acc := range relatedAccs {
		accs[acc.String()] = db.NewAccountFromSDKAddress(acc)
	}
	accTxs := make([]db.AccountTransaction, 0)
	for acc := range accs {
		accTxs = append(accTxs, db.NewAccountTx(db.GetTxID(txHash, height), height, acc, signer))
	}
	return accs, accTxs, nil
}
