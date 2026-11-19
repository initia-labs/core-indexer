package db

import (
	"context"
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InsertTransferAlertBatches shares the caller's transaction with either indexed
// events or the RPC reader checkpoint. Conflicts preserve prior evaluation state.
func InsertTransferAlertBatches(ctx context.Context, tx *gorm.DB, chain string, block *mq.BlockResultMsg) error {
	batches, err := transferAlertBatches(chain, block)
	if err != nil || len(batches) == 0 {
		return err
	}
	return tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(batches, BatchSize).Error
}

func transferAlertBatches(chain string, block *mq.BlockResultMsg) ([]TransferAlertBatch, error) {
	var batches []TransferAlertBatch
	appendEvents := func(hash string, events []abci.Event) error {
		for _, event := range events {
			if event.Type != "send_packet" && event.Type != "coin_spent" && event.Type != "move" {
				continue
			}
			// Keep the full group to preserve event indexes, including finalize-block
			// events omitted by the search index. No rules or thresholds run here.
			normalized := make([]abci.Event, len(events))
			for i, event := range events {
				normalized[i] = event
				normalized[i].Attributes = append([]abci.EventAttribute(nil), event.Attributes...)
				for j := range normalized[i].Attributes {
					a := &normalized[i].Attributes[j]
					a.Key = NormalizeEscapeString(a.Key)
					a.Value = NormalizeEscapeString(a.Value)
				}
			}
			data, err := json.Marshal(normalized)
			if err != nil {
				return err
			}
			batches = append(batches, TransferAlertBatch{ChainID: chain, BlockHeight: block.Height, TransactionHash: hash, Events: JSONB(data)})
			break
		}
		return nil
	}
	for _, tx := range block.Txs {
		if tx.ExecTxResults == nil {
			return nil, fmt.Errorf("tx %s: missing execution result", tx.Hash)
		}
		if tx.ExecTxResults.Code != 0 {
			continue
		}
		if err := appendEvents(tx.Hash, tx.ExecTxResults.Events); err != nil {
			return nil, err
		}
	}
	// An empty hash identifies events outside a transaction.
	if err := appendEvents("", block.FinalizeBlockEvents); err != nil {
		return nil, err
	}
	return batches, nil
}
