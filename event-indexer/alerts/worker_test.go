package alerts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"

	abci "github.com/cometbft/cometbft/abci/types"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (r roundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return r(req) }

func TestTelegramResponseAndTokenRedaction(t *testing.T) {
	for _, tc := range []struct {
		name, body   string
		status       int
		transportErr bool
		wantErr      bool
	}{
		{"sent", `{"ok":true}`, 200, false, false},
		{"rejected", `{"ok":false,"error_code":429}`, 200, false, true},
		{"HTTP error", `{"ok":false}`, 500, false, true},
		{"bad response", `bad`, 200, false, true},
		{"transport", "", 0, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
				var body map[string]any
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body["chat_id"] != "-123" || body["text"] != "alert" {
					t.Fatalf("wrong request: %v", body)
				}
				if tc.transportErr {
					return nil, errors.New("transport failed for secret-token")
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			})}
			err := (Telegram{Token: "secret-token", ChatID: "-123", Client: client}).Send(context.Background(), "alert")
			if (err != nil) != tc.wantErr {
				t.Fatalf("unexpected result %v", err)
			}
			if err != nil && strings.Contains(err.Error(), "secret-token") {
				t.Fatal("token leaked")
			}
		})
	}
}

func TestTelegramTruncatesLegacyWithoutPromotingFields(t *testing.T) {
	for _, text := range []string{
		strings.Repeat("ย", 4096),
		"Legacy route\nAmount: forged\n" + strings.Repeat("😀", 5000) + "\nEvent: real",
	} {
		client := &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
			var body struct {
				Text string `json:"text"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if !utf8.ValidString(body.Text) || utf8.RuneCountInString(body.Text) > 4096 {
				t.Fatal("invalid or oversized message")
			}
			if utf8.RuneCountInString(text) <= 4096 {
				if body.Text != text {
					t.Fatal("short message changed")
				}
			} else if !strings.HasPrefix(body.Text, "Legacy route\nAmount: forged\n") || strings.Contains(body.Text, "Event: real") {
				t.Fatal("legacy fields were promoted")
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
		})}
		if err := (Telegram{Token: "test", ChatID: "test", Client: client}).Send(context.Background(), text); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAlertPreviewPreservesTotalsWithoutVerboseMetadata(t *testing.T) {
	c := testConfig(t)
	c.TxURL = "https://scan.example/txs/"
	c.IBC[0].Name = "route\nAmount: forged\nEvent: forged\n" + strings.Repeat("x", 5000)
	got, err := c.Evaluate(block(packet("channel-2", "transfer/channel-2/uusdc", "120000000000"), event("coin_spent", "spender", "escrow", "amount", "130000000000ibc/USDC")))
	if err != nil || len(got) != 1 {
		t.Fatalf("%+v %v", got, err)
	}
	preview := telegramText(got[0].TelegramText)
	if !strings.HasPrefix(preview, "Transfer alert | test-1\nhttps://scan.example/txs/abc\n") {
		t.Fatal("lost explorer link")
	}
	summary := strings.Split(preview, "Route:")[0]
	for _, amount := range []string{"IBC outbound: 120000 USDC", "Escrow outflow: 130000 USDC"} {
		if !strings.Contains(summary, amount) {
			t.Fatalf("lost %s: %s", amount, summary)
		}
	}
	if strings.Contains(summary, "forged") || strings.Contains(preview, "\nAmount: forged") || utf8.RuneCountInString(preview) > 1000 {
		t.Fatal("untrusted fields promoted or oversized preview")
	}
}

func TestAlertPreviewCountsOmittedTotals(t *testing.T) {
	header := "Event: test-1/42/tx/abc"
	totals := make([]string, 200)
	for i := range totals {
		totals[i] = "IBC outbound\nAmount: 100001 USDC (threshold > 100000)"
	}
	preview := alertPreview(header, totals, "\nDetails: ignored")
	retained := strings.Count(preview, "Amount:")
	if !strings.HasPrefix(preview, header) || !strings.Contains(preview, fmt.Sprintf("%d totals omitted", len(totals)-retained)) || retained == 0 || utf8.RuneCountInString(preview) > 4096 {
		t.Fatalf("incorrect bounded totals: %s", preview)
	}
}

func TestCompactIBCPreviewAggregatesParties(t *testing.T) {
	c := testConfig(t)
	c.TxURL = "https://scan.example/txs/"
	p := packet("channel-2", "transfer/channel-2/uusdc", "60000000000")
	got, err := c.Evaluate(block(p, p))
	if err != nil || len(got) != 1 {
		t.Fatalf("%v %v", got, err)
	}
	preview := got[0].TelegramText
	for _, field := range []string{"https://scan.example/txs/abc", "IBC outbound: 120000 USDC", "Route: Noble (transfer/channel-2)", "Sender: alice", "Receiver: bob"} {
		if !strings.Contains(preview, field) {
			t.Fatalf("missing %s: %s", field, preview)
		}
	}
	for _, verbose := range []string{"Event:", "Height:", "threshold", "Sequence:", "Details:", "receipt not confirmed"} {
		if strings.Contains(preview, verbose) {
			t.Fatalf("verbose field %q: %s", verbose, preview)
		}
	}
	if strings.Count(preview, "Sender:") != 1 {
		t.Fatal("duplicate parties in preview")
	}
	if !strings.Contains(got[0].Text, "Event: test-1/42/tx/abc") || !strings.Contains(got[0].Text, "Sequence: 7") {
		t.Fatal("full database details lost")
	}
}

func TestEscrowPreviewRecipientAttribution(t *testing.T) {
	debit := event("coin_spent", "spender", "escrow", "amount", "120000000000ibc/USDC")
	bankTransfer := func(sender, receiver, amount string) abci.Event {
		return event("transfer", "sender", sender, "recipient", receiver, "amount", amount+"ibc/USDC")
	}
	for _, tc := range []struct {
		name     string
		events   []abci.Event
		receiver string
	}{
		{"send", []abci.Event{debit, bankTransfer("escrow", "bob", "120000000000")}, "bob"},
		{"multi send", []abci.Event{debit, bankTransfer("escrow", "bob", "60000000000"), bankTransfer("escrow", "carol", "60000000000")}, "bob, carol"},
		{"different sender", []abci.Event{debit, bankTransfer("someone-else", "bob", "120000000000")}, "Not available in event"},
		{"incomplete amount", []abci.Event{debit, bankTransfer("escrow", "bob", "60000000000")}, "Not available in event"},
		{"different debit", []abci.Event{debit, event("coin_spent", "spender", "someone-else", "amount", "120000000000ibc/USDC"), bankTransfer("escrow", "bob", "120000000000")}, "Not available in event"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := testConfig(t).Evaluate(block(tc.events...))
			if err != nil || len(got) != 1 {
				t.Fatalf("%v %v", got, err)
			}
			if !strings.Contains(got[0].TelegramText, "Sender: escrow\nReceiver: "+tc.receiver) {
				t.Fatal(got[0].TelegramText)
			}
		})
	}
}

func TestCompactPreviewBoundsParticipantsAndUnknownLZReceiver(t *testing.T) {
	c := testConfig(t)
	var events []abci.Event
	for i := range 10 {
		p := packet("channel-2", "transfer/channel-2/uusdc", "60000000000")
		p.Attributes[len(p.Attributes)-1].Value = fmt.Sprintf(`{"denom":"transfer/channel-2/uusdc","amount":"60000000000","sender":"alice","receiver":"bob%d"}`, i)
		events = append(events, p)
	}
	got, err := c.Evaluate(block(events...))
	if err != nil || len(got) != 1 {
		t.Fatalf("%v %v", got, err)
	}
	preview := got[0].TelegramText
	if strings.Count(preview, "Receiver:") != 3 || !strings.Contains(preview, "+7 more entries") || !strings.Contains(preview, "600000 USDC") {
		t.Fatal(preview)
	}
	got, err = c.Evaluate(block(event("move", "type_tag", c.LayerZero[0].TypeTag, "data", `{"guid":"secret-guid","dst_eid":30101,"from_address":"0x1","amount_sent_ld":"100000000001"}`)))
	if err != nil || len(got) != 1 {
		t.Fatalf("%v %v", got, err)
	}
	preview = got[0].TelegramText
	if !strings.Contains(preview, "EID 30101") || !strings.Contains(preview, "Sender: 0x1\nReceiver: Not available in event") || strings.Contains(preview, "secret-guid") {
		t.Fatal(preview)
	}
}
