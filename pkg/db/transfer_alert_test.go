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
	move := []abci.Event{{Type: "message"}, {Type: "move", Attributes: []abci.EventAttribute{{Key: "type_tag", Value: "0x1::oft_core::OftSent"}}}}
	finalize := []abci.Event{{Type: "unrelated"}, {Type: "coin_spent"}} // No indexed attributes or mode.
	block := &mq.BlockResultMsg{Height: 42, Txs: []mq.TxResult{
		{Hash: "success", ExecTxResults: &abci.ExecTxResult{Events: events}},
		{Hash: "move-only", ExecTxResults: &abci.ExecTxResult{Events: move}},
		{Hash: "failed", ExecTxResults: &abci.ExecTxResult{Code: 1, Events: events}},
		{Hash: "unrelated", ExecTxResults: &abci.ExecTxResult{Events: []abci.Event{{Type: "message"}}}},
	}, FinalizeBlockEvents: finalize}
	got, err := transferAlertBatches("test-1", block)
	if err != nil || len(got) != 3 {
		t.Fatalf("batches: %+v %v", got, err)
	}
	for i, want := range [][]abci.Event{events, move, finalize} {
		var decoded []abci.Event
		if err := json.Unmarshal(got[i].Events, &decoded); err != nil || !reflect.DeepEqual(decoded, want) {
			t.Fatalf("event boundaries changed: %+v %v", decoded, err)
		}
		if got[i].ChainID != "test-1" || got[i].BlockHeight != 42 || got[i].EvaluatedAt != nil {
			t.Fatalf("wrong batch identity/state: %+v", got[i])
		}
	}
	if got[0].TransactionHash != "success" || got[1].TransactionHash != "move-only" || got[2].TransactionHash != "" {
		t.Fatalf("wrong transaction scope: %+v", got)
	}
}

func TestTransferAlertBatchesNormalizeCopies(t *testing.T) {
	events := []abci.Event{{Type: "coin_spent", Attributes: []abci.EventAttribute{{Key: "spender", Value: "escrow"}, {Key: "amount", Value: "1uinit"}, {Key: "note", Value: "a\x00b", Index: true}, {Key: "extra", Value: "c\x00d"}}}}
	b := &mq.BlockResultMsg{Height: 42, Txs: []mq.TxResult{{Hash: "tx", ExecTxResults: &abci.ExecTxResult{Events: events}}}, FinalizeBlockEvents: events}
	got, err := transferAlertBatches("test-1", b)
	if err != nil || len(got) != 2 {
		t.Fatalf("%+v %v", got, err)
	}
	for _, group := range got {
		var decoded []abci.Event
		if err := json.Unmarshal(group.Events, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded[0].Attributes[2].Value != "a�b" || decoded[0].Attributes[3].Value != "c�d" || !decoded[0].Attributes[2].Index {
			t.Fatal("normalization changed event structure")
		}
	}
	if events[0].Attributes[2].Value != "a\x00b" || events[0].Attributes[3].Value != "c\x00d" {
		t.Fatal("shared events were mutated")
	}
}
