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

type Alert struct{ ID, Text, TelegramText string }

type transferKey struct {
	kind, asset, escrow string
}

type transfer struct {
	transferKey
	route, details   string
	sender, receiver string
	amount           *big.Int
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
		parties    []transfer
	}
	byKey := make(map[transferKey]*total)
	var totals []*total // Preserve event order for deterministic messages on replay.
	for i, event := range events {
		transfers, err := c.parseEvent(event)
		if err != nil {
			return nil, fmt.Errorf("%s event %d: %w", id, i, err)
		}
		for _, tr := range transfers {
			if tr.escrow != "" {
				tr.receiver = escrowReceiver(events[i+1:], tr, c.Assets[tr.asset].Denom)
			}
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
			if !slices.ContainsFunc(group.parties, func(p transfer) bool {
				return p.route == tr.route && p.sender == tr.sender && p.receiver == tr.receiver
			}) {
				group.parties = append(group.parties, tr)
			}
		}
	}
	var summaries, details []string
	var telegramTotals, telegramParties []string
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
			details = append(details, fmt.Sprintf("Route: %s — %s %s", route, display(group.routes[route], a.Decimals), group.asset))
		}
		summaries = append(summaries, summary)
		details = append(details, group.details...)
		telegramTotals = append(telegramTotals, fmt.Sprintf("%s: %s %s", group.kind, display(&group.amount, a.Decimals), group.asset))
		for _, p := range group.parties[:min(3, len(group.parties))] {
			telegramParties = append(telegramParties, fmt.Sprintf("Route: %s\nSender: %s\nReceiver: %s", previewField(p.route), previewField(p.sender), previewField(p.receiver)))
		}
		if len(group.parties) > 3 {
			telegramParties = append(telegramParties, fmt.Sprintf("+%d more entries; see explorer", len(group.parties)-3))
		}
	}
	if len(summaries) == 0 {
		return nil, nil
	}
	header := fmt.Sprintf("Event: %s\nTransfer alert | %s\nHeight: %d", id, c.Chain, height)
	if tx != "" {
		header += "\nTx: " + c.TxURL + tx
	}
	// Put every matching total before the verbose event details.
	detailText := "\n\nDetails:\n" + strings.Join(details, "\n\n")
	text := header + "\n\n" + strings.Join(summaries, "\n\n") + detailText
	telegramHeader := "Transfer alert | " + c.Chain
	if tx != "" {
		telegramHeader += "\n" + c.TxURL + tx
	} else {
		telegramHeader += fmt.Sprintf("\nBlock: %d", height)
	}
	preview := alertPreview(telegramHeader, telegramTotals, "\n\n"+strings.Join(telegramParties, "\n\n"))
	return []Alert{{ID: id, Text: text, TelegramText: preview}}, nil
}

func (c Config) parseEvent(event abci.Event) ([]transfer, error) {
	attrs := make(map[string]string, len(event.Attributes))
	for _, a := range event.Attributes {
		// Move puts its canonical envelope first, then appends scalar JSON fields.
		if _, exists := attrs[a.Key]; exists && event.Type == "move" && (a.Key == "type_tag" || a.Key == "data") {
			continue
		}
		attrs[a.Key] = a.Value
	}
	var out []transfer
	emit := func(tr transfer, amount string) error {
		n, err := unsigned(amount)
		if err != nil {
			return err
		}
		// Overlapping rules must not count the same physical event twice.
		for _, existing := range out {
			if existing.transferKey == tr.transferKey {
				return nil
			}
		}
		tr.amount = n
		out = append(out, tr)
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
				if err := emit(transfer{transferKey: transferKey{kind: "IBC outbound", asset: id}, route: fmt.Sprintf("%s (%s/%s)", r.Name, r.Port, r.Channel), details: details, sender: p.Sender, receiver: p.Receiver}, p.Amount); err != nil {
					return nil, err
				}
			}
		}
	case "coin_spent":
		for _, e := range c.Escrows {
			if attrs["spender"] != e.Address {
				continue
			}
			amount, present := attrs["amount"]
			if !present {
				return nil, fmt.Errorf("missing escrow transfer amount")
			}
			if amount == "" {
				continue
			}
			for _, coin := range strings.Split(amount, ",") {
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
					if err := emit(transfer{transferKey: transferKey{kind: "Escrow outflow", asset: id, escrow: e.Address}, route: e.Name, details: "Spender: " + e.Address, sender: e.Address}, coin[:n]); err != nil {
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
			if err := emit(transfer{transferKey: transferKey{kind: "LZ outbound", asset: l.Asset}, route: fmt.Sprintf("%s (EID %d)", l.Name, p.Destination), details: details, sender: p.Sender}, p.Amount); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// Bank sends emit coin_spent before their transfer events. Restrict attribution
// to that debit and require the outputs to account for its full asset amount.
func escrowReceiver(events []abci.Event, debit transfer, denom string) string {
	var total big.Int
	var receivers []string
	for _, event := range events {
		if event.Type == "coin_spent" {
			break
		}
		if event.Type != "transfer" {
			continue
		}
		attrs := make(map[string]string, len(event.Attributes))
		for _, a := range event.Attributes {
			attrs[a.Key] = a.Value
		}
		if attrs["sender"] != debit.escrow || attrs["recipient"] == "" {
			continue
		}
		for _, coin := range strings.Split(attrs["amount"], ",") {
			n := strings.IndexFunc(coin, func(r rune) bool { return r < '0' || r > '9' })
			if n <= 0 || coin[n:] != denom {
				continue
			}
			amount, err := unsigned(coin[:n])
			if err != nil {
				continue
			}
			total.Add(&total, amount)
			if !slices.Contains(receivers, attrs["recipient"]) {
				receivers = append(receivers, attrs["recipient"])
			}
		}
	}
	if total.Cmp(debit.amount) != 0 {
		return ""
	}
	return strings.Join(receivers, ", ")
}

func previewField(value string) string {
	if value == "" {
		return "Not available in event"
	}
	runes := []rune(displayAddress(value))
	if len(runes) > 160 {
		return string(runes[:157]) + "..."
	}
	return string(runes)
}

// ICS-20 receiver strings can contain control characters, including NUL, which
// PostgreSQL TEXT cannot store. Quote them without changing ordinary addresses.
func displayAddress(address string) string {
	quoted := strconv.Quote(address)
	return quoted[1 : len(quoted)-1]
}
