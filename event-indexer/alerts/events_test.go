package alerts

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

func event(kind string, attrs ...string) abci.Event {
	e := abci.Event{Type: kind}
	for i := 0; i < len(attrs); i += 2 {
		e.Attributes = append(e.Attributes, abci.EventAttribute{Key: attrs[i], Value: attrs[i+1]})
	}
	return e
}
func packet(channel, denom, amount string) abci.Event {
	return event("send_packet", "packet_src_port", "transfer", "packet_src_channel", channel, "packet_sequence", "7", "packet_data", fmt.Sprintf(`{"denom":%q,"amount":%q,"sender":"alice","receiver":"bob"}`, denom, amount))
}
func testConfig(t *testing.T) Config {
	t.Helper()
	c := Config{Chain: "test-1", Assets: map[string]Asset{
		"USDC": {Denom: "ibc/USDC", PacketDenoms: []string{"transfer/channel-2/uusdc"}, Decimals: 6, Threshold: "100000"},
		"WETH": {Denom: "move/weth", Decimals: 18, Threshold: "100000"},
	}, IBC: []Route{{Name: "Noble", Port: "transfer", Channel: "channel-2", Assets: []string{"USDC"}}},
		Escrows:   []Escrow{{Name: "Rollup escrow", Address: "escrow", Assets: []string{"USDC", "WETH"}}},
		LayerZero: []LayerZero{{Name: "USDC bridge", TypeTag: "0x123::oft_core::OftSent", Asset: "USDC", Destinations: []uint32{30101}}}}
	if err := c.prepare(); err != nil {
		t.Fatal(err)
	}
	return c
}
func block(events ...abci.Event) mq.BlockResultMsg {
	return mq.BlockResultMsg{Height: 42, Txs: []mq.TxResult{{Hash: "abc", ExecTxResults: &abci.ExecTxResult{Events: events}}}}
}

func TestIBCTransactionTotalAndFilters(t *testing.T) {
	c := testConfig(t)
	b := block(packet("channel-2", "transfer/channel-2/uusdc", "100000000000"),
		packet("channel-2", "transfer/channel-2/uusdc", "100000000001"),
		packet("channel-9", "transfer/channel-2/uusdc", "999999999999"),
		packet("channel-2", "USDC", "999999999999"))
	got, err := c.Evaluate(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0].Text, "Amount: 200000.000001 USDC") || got[0].ID != "test-1/42/tx/abc" {
		t.Fatalf("wrong alerts: %+v", got)
	}
	b.Txs[0].ExecTxResults.Code = 5
	got, err = c.Evaluate(b)
	if err != nil || len(got) != 0 {
		t.Fatalf("failed transaction emitted alerts: %+v %v", got, err)
	}
}

func TestHexPacketAndMalformedMonitoredPacket(t *testing.T) {
	c := testConfig(t)
	e := packet("channel-2", "transfer/channel-2/uusdc", "100000000001")
	last := &e.Attributes[len(e.Attributes)-1]
	last.Key = "packet_data_hex"
	last.Value = hex.EncodeToString([]byte(last.Value))
	got, err := c.Evaluate(block(e))
	if err != nil || len(got) != 1 {
		t.Fatalf("hex packet: %v %v", got, err)
	}
	last.Value = "not-hex"
	if _, err := c.Evaluate(block(e)); err == nil {
		t.Fatal("malformed matching packet must fail, not disappear")
	}
}

func TestEscrowOutflowAndLargeInteger(t *testing.T) {
	c := testConfig(t)
	transfer := event("coin_spent", "spender", "escrow", "amount", "100000000001ibc/USDC,100000000000000000000001move/weth")
	b := block(event("coin_received", "receiver", "escrow", "amount", "999999999999ibc/USDC"), transfer, event("transfer", "sender", "escrow", "recipient", "bob", "amount", "100000000001ibc/USDC"))
	got, err := c.Evaluate(b)
	if err != nil || len(got) != 1 {
		t.Fatalf("outflow: %v %v", got, err)
	}
	if !strings.Contains(got[0].Text, "100000.000000000000000001 WETH") {
		t.Fatal(got[0].Text)
	}
	b.Txs = nil
	b.FinalizeBlockEvents = []abci.Event{transfer}
	got, err = c.Evaluate(b)
	if err != nil || len(got) != 1 || !strings.Contains(got[0].ID, "/block/") {
		t.Fatalf("block transfer: %v %v", got, err)
	}
}

func TestLayerZeroUsesDebitedLocalUnits(t *testing.T) {
	c := testConfig(t)
	e := event("move", "type_tag", "0x123::oft_core::OftSent", "data", `{"guid":"0xabc","dst_eid":30101,"from_address":"0x1","amount_sent_ld":"100000000001","amount_received_ld":"99999999999"}`)
	got, err := c.Evaluate(block(e))
	if err != nil || len(got) != 1 {
		t.Fatalf("LZ: %v %v", got, err)
	}
	e.Attributes[0].Value = "0x999::oft_core::OftSent"
	got, err = c.Evaluate(block(e))
	if err != nil || len(got) != 0 {
		t.Fatalf("unmonitored module: %v %v", got, err)
	}
	e.Attributes[0].Value = "0x123::oft_core::OftSent"
	e.Attributes[1].Value = strings.ReplaceAll(e.Attributes[1].Value, "30101", "30102")
	got, err = c.Evaluate(block(e))
	if err != nil || len(got) != 0 {
		t.Fatalf("unmonitored destination: %v %v", got, err)
	}
}

func TestFractionalThreshold(t *testing.T) {
	n, err := baseUnits("0.000001", 6)
	if err != nil || n.String() != "1" {
		t.Fatalf("%v %v", n, err)
	}
	for _, s := range []string{"1.0000001", "-1", "NaN", "1e5", "1/2"} {
		if _, err := baseUnits(s, 6); err == nil {
			t.Fatalf("accepted %s", s)
		}
	}
}

func TestIBCReceiverControlsAreEscaped(t *testing.T) {
	c := testConfig(t)
	e := packet("channel-2", "transfer/channel-2/uusdc", "100000000001")
	e.Attributes[len(e.Attributes)-1].Value = `{"denom":"transfer/channel-2/uusdc","amount":"100000000001","sender":"alice","receiver":"bob\u0000\nEvent: fake"}`
	got, err := c.Evaluate(block(e))
	if err != nil || len(got) != 1 {
		t.Fatalf("%v %v", got, err)
	}
	if strings.ContainsRune(got[0].Text, 0) || !strings.Contains(got[0].Text, `To: bob\x00\nEvent: fake`) {
		t.Fatalf("unsafe receiver display: %q", got[0].Text)
	}
	if !strings.Contains(got[0].Text, "From: alice\n") {
		t.Fatal("normal address changed")
	}
}

func TestTransactionAggregation(t *testing.T) {
	ibc := func(channel, amount string) abci.Event {
		return packet(channel, "transfer/channel-2/uusdc", amount)
	}
	escrow := func(address, amount string) abci.Event {
		return event("coin_spent", "spender", address, "amount", amount)
	}
	lz := func(amount string) abci.Event {
		return event("move", "type_tag", "0x123::oft_core::OftSent", "data", fmt.Sprintf(`{"guid":"0xabc","dst_eid":30101,"from_address":"0x1","amount_sent_ld":%q}`, amount))
	}
	for _, tc := range []struct {
		name   string
		events []abci.Event
		want   []string
	}{
		{"split IBC across channels", []abci.Event{ibc("channel-2", "60000000000"), ibc("channel-3", "60000000000")}, []string{
			"Amount: 120000 USDC", "Route: Noble (transfer/channel-2) — 60000 USDC", "Route: Injective (transfer/channel-3) — 60000 USDC",
		}},
		{"exact threshold", []abci.Event{ibc("channel-2", "50000000000"), ibc("channel-2", "50000000000")}, nil},
		{"one base unit over", []abci.Event{ibc("channel-2", "50000000000"), ibc("channel-2", "50000000001")}, []string{"Amount: 100000.000001 USDC"}},
		{"overlapping rules do not duplicate", []abci.Event{ibc("channel-2", "60000000000"), lz("60000000000"), escrow("escrow", "60000000000ibc/USDC")}, nil},
		{"different assets stay separate", []abci.Event{escrow("escrow", "60000000000ibc/USDC,60000000000000000000000move/weth")}, nil},
		{"different escrows stay separate", []abci.Event{escrow("escrow", "60000000000ibc/USDC"), escrow("other", "60000000000ibc/USDC")}, nil},
		{"split escrow debits", []abci.Event{escrow("escrow", "60000000000ibc/USDC"), escrow("escrow", "60000000000ibc/USDC")}, []string{"Escrow outflow\nAmount: 120000 USDC", "Spender: escrow"}},
		{"split LayerZero sends", []abci.Event{lz("60000000000"), lz("60000000000")}, []string{"LZ outbound\nAmount: 120000 USDC"}},
		{"one message for multiple types", []abci.Event{
			ibc("channel-2", "120000000000"), lz("120000000000"), escrow("escrow", "120000000000ibc/USDC"),
		}, []string{"IBC outbound\nAmount: 120000 USDC", "LZ outbound\nAmount: 120000 USDC", "Escrow outflow\nAmount: 120000 USDC"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := testConfig(t)
			// Duplicate rules must not inflate any of the totals.
			c.IBC = append(c.IBC, c.IBC[0], Route{Name: "Injective", Port: "transfer", Channel: "channel-3", Assets: []string{"USDC"}})
			c.Escrows = append(c.Escrows, c.Escrows[0], Escrow{Name: "Other escrow", Address: "other", Assets: []string{"USDC"}})
			c.LayerZero = append(c.LayerZero, c.LayerZero[0])
			b := block(tc.events...)
			got, err := c.Evaluate(b)
			if err != nil {
				t.Fatal(err)
			}
			if len(tc.want) == 0 {
				if len(got) != 0 {
					t.Fatalf("unexpected alert: %+v", got)
				}
				return
			}
			if len(got) != 1 || got[0].ID != "test-1/42/tx/abc" {
				t.Fatalf("want one transaction alert: %+v", got)
			}
			for _, want := range tc.want {
				if !strings.Contains(got[0].Text, want) {
					t.Fatalf("missing %q in %s", want, got[0].Text)
				}
			}
			replay, err := c.Evaluate(b)
			if err != nil || !reflect.DeepEqual(got, replay) {
				t.Fatalf("replay changed alert: %+v, %v", replay, err)
			}
		})
	}
}

func TestAggregationDoesNotCrossTransactionsOrBlockEvents(t *testing.T) {
	c := testConfig(t)
	b := block(packet("channel-2", "transfer/channel-2/uusdc", "60000000000"))
	second := b.Txs[0]
	second.Hash = "def"
	b.Txs = append(b.Txs, second)
	b.FinalizeBlockEvents = []abci.Event{
		event("coin_spent", "spender", "escrow", "amount", "60000000000ibc/USDC"),
		event("coin_spent", "spender", "escrow", "amount", "60000000000ibc/USDC"),
	}
	got, err := c.Evaluate(b)
	if err != nil || len(got) != 0 {
		t.Fatalf("summed across transaction or block-event boundaries: %+v %v", got, err)
	}
}
