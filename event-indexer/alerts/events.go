package alerts

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

type Alert struct{ ID, Text string }

// Evaluate keeps event boundaries intact, including transactions with several packets/transfers.
func (c Config) Evaluate(b mq.BlockResultMsg) ([]Alert, error) {
	var alerts []Alert
	for _, tx := range b.Txs {
		if tx.ExecTxResults == nil {
			return nil, fmt.Errorf("tx %s: missing execution result", tx.Hash)
		}
		if tx.ExecTxResults.Code != 0 {
			continue
		}
		for i, event := range tx.ExecTxResults.Events {
			found, err := c.evaluateEvent(b.Height, tx.Hash, i, event)
			if err != nil {
				return nil, fmt.Errorf("height %d tx %s event %d: %w", b.Height, tx.Hash, i, err)
			}
			alerts = append(alerts, found...)
		}
	}
	// Module transfers can also execute outside a transaction (e.g. end block).
	for i, event := range b.FinalizeBlockEvents {
		found, err := c.evaluateEvent(b.Height, "block", i, event)
		if err != nil {
			return nil, fmt.Errorf("height %d block event %d: %w", b.Height, i, err)
		}
		alerts = append(alerts, found...)
	}
	return alerts, nil
}

func (c Config) evaluateEvent(height int64, tx string, index int, event abci.Event) ([]Alert, error) {
	attrs := make(map[string]string, len(event.Attributes))
	for _, a := range event.Attributes {
		attrs[a.Key] = a.Value
	}
	var out []Alert
	emit := func(kind, route, asset, amount, details string) error {
		a := c.Assets[asset]
		n, err := unsigned(amount)
		if err != nil {
			return err
		}
		if n.Cmp(a.limit) <= 0 {
			return nil
		}
		id := fmt.Sprintf("%s/%d/%s/%d/%s/%s", c.Chain, height, tx, index, kind, asset)
		text := fmt.Sprintf("%s | %s\nRoute: %s\nAmount: %s %s (threshold > %s)\n%s\nHeight: %d\nEvent: %s", kind, c.Chain, route, display(n, a.Decimals), asset, a.Threshold, details, height, id)
		if tx != "block" {
			text += "\nTx: " + c.TxURL + tx
		}
		// Repeated route entries must not duplicate one physical event.
		for _, existing := range out {
			if existing.ID == id {
				return nil
			}
		}
		out = append(out, Alert{ID: id, Text: text})
		return nil
	}
	switch event.Type {
	case "send_packet":
		for _, r := range c.IBC {
			if attrs["packet_src_port"] != r.Port || attrs["packet_src_channel"] != r.Channel {
				continue
			}
			data := []byte(attrs["packet_data"])
			if len(data) == 0 {
				var err error
				data, err = hex.DecodeString(attrs["packet_data_hex"])
				if err != nil {
					return nil, fmt.Errorf("decode packet: %w", err)
				}
			}
			var p struct{ Denom, Amount, Sender, Receiver string }
			if err := json.Unmarshal(data, &p); err != nil {
				return nil, fmt.Errorf("decode ICS-20 packet: %w", err)
			}
			for _, id := range r.Assets {
				if !slices.Contains(c.Assets[id].PacketDenoms, p.Denom) {
					continue
				}
				details := fmt.Sprintf("Status: sent (destination receipt not confirmed)\nChannel: %s/%s -> %s/%s\nSequence: %s\nFrom: %s\nTo: %s", r.Port, r.Channel, attrs["packet_dst_port"], attrs["packet_dst_channel"], attrs["packet_sequence"], displayAddress(p.Sender), displayAddress(p.Receiver))
				if err := emit("IBC outbound", r.Name, id, p.Amount, details); err != nil {
					return nil, err
				}
			}
		}
	case "coin_spent":
		for _, e := range c.Escrows {
			if attrs["spender"] != e.Address {
				continue
			}
			for _, coin := range strings.Split(attrs["amount"], ",") {
				n := 0
				for n < len(coin) && coin[n] >= '0' && coin[n] <= '9' {
					n++
				}
				if n == 0 || n == len(coin) {
					return nil, fmt.Errorf("invalid escrow transfer coin")
				}
				for _, id := range e.Assets {
					if coin[n:] != c.Assets[id].Denom {
						continue
					}
					if err := emit("Escrow outflow", e.Name, id, coin[:n], "Spender: "+e.Address); err != nil {
						return nil, err
					}
				}
			}
		}
	case "move":
		for _, l := range c.LayerZero {
			if attrs["type_tag"] != l.TypeTag {
				continue
			}
			// LayerZero devtools examples/oft-initia/sources/shared_oft/oft_core.move.
			var p struct {
				GUID        json.RawMessage `json:"guid"`
				Destination uint32          `json:"dst_eid"`
				Sender      string          `json:"from_address"`
				Amount      string          `json:"amount_sent_ld"`
			}
			if err := json.Unmarshal([]byte(attrs["data"]), &p); err != nil {
				return nil, fmt.Errorf("decode OftSent: %w", err)
			}
			if len(l.Destinations) != 0 && !slices.Contains(l.Destinations, p.Destination) {
				continue
			}
			details := "Status: sent (destination receipt not confirmed)\nDestination EID: " + strconv.FormatUint(uint64(p.Destination), 10) + "\nFrom: " + p.Sender + "\nGUID: " + string(p.GUID)
			if err := emit("LZ outbound", l.Name, l.Asset, p.Amount, details); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// ICS-20 receiver strings can contain control characters, including NUL, which
// PostgreSQL TEXT cannot store. Quote them without changing ordinary addresses.
func displayAddress(address string) string {
	quoted := strconv.Quote(address)
	return quoted[1 : len(quoted)-1]
}
