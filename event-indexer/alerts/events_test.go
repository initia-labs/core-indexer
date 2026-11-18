package alerts

import (
	"encoding/hex"
	"fmt"
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

func TestIBCThresholdAndEventBoundaries(t *testing.T) {
	c := testConfig(t)
	b := block(packet("channel-2", "transfer/channel-2/uusdc", "100000000000"),
		packet("channel-2", "transfer/channel-2/uusdc", "100000000001"),
		packet("channel-9", "transfer/channel-2/uusdc", "999999999999"),
		packet("channel-2", "USDC", "999999999999"))
	got, err := c.Evaluate(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0].Text, "100000.000001 USDC") || !strings.Contains(got[0].ID, "/1/") {
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
	if err != nil || len(got) != 2 {
		t.Fatalf("outflow: %v %v", got, err)
	}
	if !strings.Contains(got[1].Text, "100000.000000000000000001 WETH") {
		t.Fatal(got[1].Text)
	}
	b.Txs = nil
	b.FinalizeBlockEvents = []abci.Event{transfer}
	got, err = c.Evaluate(b)
	if err != nil || len(got) != 2 || !strings.Contains(got[0].ID, "/block/") {
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
