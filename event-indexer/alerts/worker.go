package alerts

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	_ "github.com/lib/pq"

	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/storage"
)

type Telegram struct {
	Token, ChatID string
	Client        *http.Client
}

func (t Telegram) Send(ctx context.Context, text string) error {
	body, err := json.Marshal(map[string]any{"chat_id": t.ChatID, "text": text, "link_preview_options": map[string]bool{"is_disabled": true}})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.telegram.org/bot"+t.Token+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create Telegram request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.Client.Do(req)
	// HTTP errors include the URL, which contains the bot token.
	if err != nil {
		return fmt.Errorf("Telegram request failed")
	}
	defer resp.Body.Close()
	var result struct {
		OK        bool `json:"ok"`
		ErrorCode int  `json:"error_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("Telegram returned invalid JSON (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK || !result.OK {
		return fmt.Errorf("Telegram rejected message (HTTP %d, code %d)", resp.StatusCode, result.ErrorCode)
	}
	return nil
}

// Deliver persists pending messages before sending. A failed send remains retryable.
// Telegram has no idempotency key: a crash after delivery but before commit can duplicate a message.
func Deliver(ctx context.Context, db *sql.DB, a Alert, send func(context.Context, string) error) error {
	if _, err := db.ExecContext(ctx, `INSERT INTO transfer_alerts (id, message) VALUES ($1, $2) ON CONFLICT DO NOTHING`, a.ID, a.Text); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var message string
	var sent sql.NullTime
	if err := tx.QueryRowContext(ctx, `SELECT message, sent_at FROM transfer_alerts WHERE id = $1 FOR UPDATE`, a.ID).Scan(&message, &sent); err != nil {
		return err
	}
	if !sent.Valid {
		if err := send(ctx, message); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE transfer_alerts SET sent_at = now() WHERE id = $1`, a.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func Run(ctx context.Context, c Config) error {
	for _, name := range []string{"DB_CONNECTION_STRING", "TELEGRAM_BOT_TOKEN", "TELEGRAM_CHAT_ID"} {
		if os.Getenv(name) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	db, err := sql.Open("postgres", os.Getenv("DB_CONNECTION_STRING"))
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	t := Telegram{Token: os.Getenv("TELEGRAM_BOT_TOKEN"), ChatID: os.Getenv("TELEGRAM_CHAT_ID"), Client: &http.Client{Timeout: 15 * time.Second}}
	process := func(ctx context.Context, b mq.BlockResultMsg) error {
		alerts, err := c.Evaluate(b)
		if err != nil {
			return err
		}
		for _, a := range alerts {
			if err := Deliver(ctx, db, a, t.Send); err != nil {
				return err
			}
		}
		return nil
	}
	if c.RPC != "" {
		return pollRPC(ctx, c, db, process)
	}
	for _, name := range []string{"PUBSUB_PROJECT_ID", "BLOCK_RESULTS_SUBSCRIPTION", "BLOCK_RESULTS_CLAIM_CHECK_BUCKET"} {
		if os.Getenv(name) == "" {
			return fmt.Errorf("%s is required in Pub/Sub mode", name)
		}
	}
	consumer, err := mq.NewConsumer(ctx, os.Getenv("PUBSUB_PROJECT_ID"), os.Getenv("BLOCK_RESULTS_SUBSCRIPTION"), os.Getenv("PUBSUB_ENDPOINT"))
	if err != nil {
		return err
	}
	defer consumer.Close()
	gcs, err := storage.NewGCSClient()
	if err != nil {
		return err
	}
	return consumer.Receive(ctx, func(ctx context.Context, m *mq.Message) error {
		data := m.Data
		if strings.HasPrefix(m.Attributes["key"], mq.NEW_BLOCK_RESULTS_CLAIM_CHECK_MESSAGE_KEY) {
			var claim mq.ClaimCheckMsg
			if err := json.Unmarshal(data, &claim); err != nil {
				return err
			}
			var err error
			data, err = gcs.ReadFile(os.Getenv("BLOCK_RESULTS_CLAIM_CHECK_BUCKET"), claim.ObjectPath)
			if err != nil {
				return err
			}
		}
		var b mq.BlockResultMsg
		if err := json.Unmarshal(data, &b); err != nil {
			return err
		}
		return process(ctx, b)
	})
}

// Run one polling worker per chain. Advance its cursor only after all alerts succeed.
func pollRPC(ctx context.Context, c Config, db *sql.DB, process func(context.Context, mq.BlockResultMsg) error) error {
	rpc, err := rpchttp.NewWithClient(c.RPC, "/websocket", &http.Client{Timeout: 15 * time.Second})
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO transfer_alert_cursors (chain_id, height) VALUES ($1, $2) ON CONFLICT DO NOTHING`, c.Chain, c.StartHeight-1); err != nil {
		return err
	}
	var height int64
	if err := db.QueryRowContext(ctx, `SELECT height FROM transfer_alert_cursors WHERE chain_id = $1`, c.Chain).Scan(&height); err != nil {
		return err
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		status, err := rpc.Status(ctx)
		if err != nil {
			return err
		}
		for height < status.SyncInfo.LatestBlockHeight {
			next := height + 1
			block, err := rpc.Block(ctx, &next)
			if err != nil {
				return err
			}
			results, err := rpc.BlockResults(ctx, &next)
			if err != nil {
				return err
			}
			data, err := mq.NewBlockResultMsgBytes(block, results)
			if err != nil {
				return err
			}
			var b mq.BlockResultMsg
			if err := json.Unmarshal(data, &b); err != nil {
				return err
			}
			if err := process(ctx, b); err != nil {
				return err
			}
			if _, err := db.ExecContext(ctx, `UPDATE transfer_alert_cursors SET height = $2, updated_at = now() WHERE chain_id = $1`, c.Chain, next); err != nil {
				return err
			}
			height = next
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
