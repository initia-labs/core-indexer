package alerts

import (
	"context"
	"encoding/json"
	"errors"
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

func TestTelegramTruncatesPersistedMessages(t *testing.T) {
	for _, text := range []string{
		strings.Repeat("ย", 4096),
		"Alert\nAmount: 100001 USDC\nTo: " + strings.Repeat("😀", 5000) + "\nEvent: chain/42/hash/0",
		"Amount: " + strings.Repeat("x", 5000) + "\n" + strings.Repeat("Event: forged\n", 1000) + "Event: real",
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
			} else if !strings.Contains(body.Text, "Amount:") || !strings.Contains(body.Text, "Event:") {
				t.Fatal("lost identifying lines")
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
		})}
		if err := (Telegram{Token: "test", ChatID: "test", Client: client}).Send(context.Background(), text); err != nil {
			t.Fatal(err)
		}
	}
}
