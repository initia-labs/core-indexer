package alerts

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

type Alert struct{ ID, Text string }

type transferKey struct {
	kind, asset, escrow string
}

type transfer struct {
	transferKey
	route, details string
	amount         *big.Int
}

// Evaluate totals each successful transaction independently. Block-level events
// have no transaction boundary, so each of those events is evaluated separately.
func (c Config) Evaluate(b mq.BlockResultMsg) ([]Alert, error) {
	var alerts []Alert
	for _, tx := range b.Txs {
		if tx.ExecTxResults == nil {
			return nil, fmt.Errorf("tx %s: missing execution result", tx.Hash)
		}
		if tx.ExecTxResults.Code != 0 {
			continue
		}
		id := fmt.Sprintf("%s/%d/tx/%s", c.Chain, b.Height, tx.Hash)
		found, err := c.evaluateEvents(b.Height, tx.Hash, id, tx.ExecTxResults.Events)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, found...)
	}
	for i, event := range b.FinalizeBlockEvents {
		id := fmt.Sprintf("%s/%d/block/%d", c.Chain, b.Height, i)
		found, err := c.evaluateEvents(b.Height, "", id, []abci.Event{event})
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, found...)
	}
	return alerts, nil
}

func (c Config) evaluateEvents(height int64, tx, id string, events []abci.Event) ([]Alert, error) {
	type total struct {
		transferKey
		amount     big.Int
		routes     map[string]*big.Int
		routeOrder []string
		details    []string
	}
	byKey := make(map[transferKey]*total)
	var totals []*total // Preserve event order for deterministic messages on replay.
	for i, event := range events {
		transfers, err := c.parseEvent(event)
		if err != nil {
			return nil, fmt.Errorf("%s event %d: %w", id, i, err)
		}
		for _, tr := range transfers {
			group := byKey[tr.transferKey]
			if group == nil {
				group = &total{transferKey: tr.transferKey, routes: make(map[string]*big.Int)}
				byKey[tr.transferKey] = group
				totals = append(totals, group)
			}
			group.amount.Add(&group.amount, tr.amount)
			if group.routes[tr.route] == nil {
				group.routes[tr.route] = new(big.Int)
				group.routeOrder = append(group.routeOrder, tr.route)
			}
			group.routes[tr.route].Add(group.routes[tr.route], tr.amount)
			group.details = append(group.details, fmt.Sprintf("Event #%d | %s | %s\nTransfer: %s %s\n%s", i, tr.kind, tr.route, display(tr.amount, c.Assets[tr.asset].Decimals), tr.asset, tr.details))
		}
	}
	var summaries, details []string
	for _, group := range totals {
		a := c.Assets[group.asset]
		if group.amount.Cmp(a.limit) <= 0 {
			continue
		}
		summary := fmt.Sprintf("%s\nAmount: %s %s (threshold > %s)", group.kind, display(&group.amount, a.Decimals), group.asset, a.Threshold)
		if group.escrow != "" {
			summary += "\nSpender: " + group.escrow
		}
		for _, route := range group.routeOrder {
			summary += fmt.Sprintf("\nRoute: %s — %s %s", route, display(group.routes[route], a.Decimals), group.asset)
		}
		summaries = append(summaries, summary)
		details = append(details, group.details...)
	}
	if len(summaries) == 0 {
		return nil, nil
	}
	text := fmt.Sprintf("Transfer alert | %s\nHeight: %d\nEvent: %s", c.Chain, height, id)
	if tx != "" {
		text += "\nTx: " + c.TxURL + tx
	}
	// Put every matching total before the verbose event details.
	text += "\n\n" + strings.Join(summaries, "\n\n") + "\n\nDetails:\n" + strings.Join(details, "\n\n")
	return []Alert{{ID: id, Text: text}}, nil
}

func (c Config) parseEvent(event abci.Event) ([]transfer, error) {
	attrs := make(map[string]string, len(event.Attributes))
	for _, a := range event.Attributes {
		attrs[a.Key] = a.Value
	}
	var out []transfer
	emit := func(kind, route, asset, escrow, amount, details string) error {
		n, err := unsigned(amount)
		if err != nil {
			return err
		}
		key := transferKey{kind: kind, asset: asset, escrow: escrow}
		// Overlapping rules must not count the same physical event twice.
		for _, existing := range out {
			if existing.transferKey == key {
				return nil
			}
		}
		out = append(out, transfer{transferKey: key, route: route, amount: n, details: details})
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
				if err := emit("IBC outbound", fmt.Sprintf("%s (%s/%s)", r.Name, r.Port, r.Channel), id, "", p.Amount, details); err != nil {
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
					if err := emit("Escrow outflow", e.Name, id, e.Address, coin[:n], "Spender: "+e.Address); err != nil {
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
			if err := emit("LZ outbound", fmt.Sprintf("%s (EID %d)", l.Name, p.Destination), l.Asset, "", p.Amount, details); err != nil {
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
