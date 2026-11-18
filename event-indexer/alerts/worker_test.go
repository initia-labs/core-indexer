package alerts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
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

// Set ALERT_TEST_DATABASE_URL to a disposable PostgreSQL database to exercise durable delivery.
func TestDeliveryRetryAndReplay(t *testing.T) {
	dsn := os.Getenv("ALERT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires disposable PostgreSQL: ALERT_TEST_DATABASE_URL")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TEMP TABLE transfer_alerts (id TEXT PRIMARY KEY, message TEXT NOT NULL, sent_at TIMESTAMPTZ)`)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	a := Alert{ID: "test-event", Text: "alert"}
	calls := 0
	send := func(context.Context, string) error {
		calls++
		if calls == 1 {
			return errors.New("offline")
		}
		return nil
	}
	if err := Deliver(ctx, db, a, send); err == nil {
		t.Fatal("send failure was lost")
	}
	if err := Deliver(ctx, db, a, send); err != nil {
		t.Fatal(err)
	}
	if err := Deliver(ctx, db, a, send); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("sent %d times, want failed attempt + success", calls)
	}
}
