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

func TestAlertPreviewPreservesCanonicalTotalsAndID(t *testing.T) {
	c := testConfig(t)
	c.IBC[0].Name = "route\nAmount: forged\nEvent: forged\n" + strings.Repeat("x", 5000)
	got, err := c.Evaluate(block(packet("channel-2", "transfer/channel-2/uusdc", "120000000000"), event("coin_spent", "spender", "escrow", "amount", "130000000000ibc/USDC")))
	if err != nil || len(got) != 1 {
		t.Fatalf("%+v %v", got, err)
	}
	preview := telegramText(got[0].TelegramText)
	if !strings.HasPrefix(preview, "Event: test-1/42/tx/abc\n") {
		t.Fatal("lost canonical ID")
	}
	summary := strings.Split(preview, "Details:")[0]
	for _, amount := range []string{"Amount: 120000 USDC", "Amount: 130000 USDC"} {
		if !strings.Contains(summary, amount) {
			t.Fatalf("lost %s: %s", amount, summary)
		}
	}
	if strings.Contains(summary, "forged") || utf8.RuneCountInString(preview) > 4096 {
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
