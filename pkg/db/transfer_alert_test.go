package db

import (
	"encoding/json"
	"reflect"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

func TestTransferAlertBatchesPreserveSuccessfulEventGroups(t *testing.T) {
	events := []abci.Event{{Type: "message"}, {Type: "send_packet", Attributes: []abci.EventAttribute{{Key: "packet_data", Value: `{"amount":"1"}`}}}, {Type: "coin_spent"}}
	finalize := []abci.Event{{Type: "unrelated"}, {Type: "coin_spent"}} // No indexed attributes or mode.
	block := &mq.BlockResultMsg{Height: 42, Txs: []mq.TxResult{
		{Hash: "success", ExecTxResults: &abci.ExecTxResult{Events: events}},
		{Hash: "failed", ExecTxResults: &abci.ExecTxResult{Code: 1, Events: events}},
		{Hash: "unrelated", ExecTxResults: &abci.ExecTxResult{Events: []abci.Event{{Type: "message"}}}},
	}, FinalizeBlockEvents: finalize}
	got, err := transferAlertBatches("test-1", block)
	if err != nil || len(got) != 2 {
		t.Fatalf("batches: %+v %v", got, err)
	}
	for i, want := range [][]abci.Event{events, finalize} {
		var decoded []abci.Event
		if err := json.Unmarshal(got[i].Events, &decoded); err != nil || !reflect.DeepEqual(decoded, want) {
			t.Fatalf("event boundaries changed: %+v %v", decoded, err)
		}
		if got[i].ChainID != "test-1" || got[i].BlockHeight != 42 || got[i].EvaluatedAt != nil {
			t.Fatalf("wrong batch identity/state: %+v", got[i])
		}
	}
	if got[0].TransactionHash != "success" || got[1].TransactionHash != "" {
		t.Fatalf("wrong transaction scope: %+v", got)
	}
}
