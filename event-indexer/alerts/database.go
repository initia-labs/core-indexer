package alerts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/initia-labs/core-indexer/pkg/mq"
	_ "github.com/lib/pq"
	"golang.org/x/sync/errgroup"
)

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
	db.SetMaxOpenConns(c.Workers + 1)
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	telegram := Telegram{Token: os.Getenv("TELEGRAM_BOT_TOKEN"), ChatID: os.Getenv("TELEGRAM_CHAT_ID"), Client: &http.Client{Timeout: 15 * time.Second}}
	group, ctx := errgroup.WithContext(ctx)
	for range c.Workers {
		group.Go(func() error {
			for ctx.Err() == nil {
				n, err := scanBatch(ctx, db, c)
				if err != nil {
					return err
				}
				if n == 0 {
					if err := wait(ctx, 2*time.Second); err != nil {
						return err
					}
				}
			}
			return ctx.Err()
		})
	}
	group.Go(func() error {
		for ctx.Err() == nil {
			if _, err := deliverNext(ctx, db, c.Chain, telegram.Send); err != nil {
				return err
			}
			// Pace notifications independently of batch evaluation.
			if err := wait(ctx, 3*time.Second); err != nil {
				return err
			}
		}
		return ctx.Err()
	})
	return group.Wait()
}

func wait(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// scanBatch atomically records both pending alerts and completed evaluations.
// Pending-row selection handles late blocks without relying on a height watermark.
func scanBatch(ctx context.Context, db *sql.DB, c Config) (int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT block_height, transaction_hash, events
		FROM transfer_alert_batches
		WHERE chain_id = $1 AND block_height >= $2 AND evaluated_at IS NULL
		ORDER BY block_height, transaction_hash LIMIT $3 FOR UPDATE SKIP LOCKED`, c.Chain, c.StartHeight, c.BatchSize)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	type batch struct {
		height int64
		hash   string
		events []byte
	}
	var batches []batch
	for rows.Next() {
		var b batch
		if err := rows.Scan(&b.height, &b.hash, &b.events); err != nil {
			return 0, err
		}
		batches = append(batches, b)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	for _, b := range batches {
		var events []abci.Event
		if err := json.Unmarshal(b.events, &events); err != nil {
			return 0, fmt.Errorf("decode indexed events at %d/%s: %w", b.height, b.hash, err)
		}
		block := mq.BlockResultMsg{Height: b.height}
		if b.hash == "" {
			block.FinalizeBlockEvents = events
		} else {
			block.Txs = []mq.TxResult{{Hash: b.hash, ExecTxResults: &abci.ExecTxResult{Events: events}}}
		}
		alerts, err := c.Evaluate(block)
		if err != nil {
			return 0, err
		}
		for _, a := range alerts {
			if _, err := tx.ExecContext(ctx, `INSERT INTO transfer_alerts (id, message) VALUES ($1, $2) ON CONFLICT DO NOTHING`, a.ID, a.Text); err != nil {
				return 0, err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE transfer_alert_batches SET evaluated_at = now()
			WHERE chain_id = $1 AND block_height = $2 AND transaction_hash = $3`, c.Chain, b.height, b.hash); err != nil {
			return 0, err
		}
	}
	return len(batches), tx.Commit()
}

// deliverNext locks one pending alert. A failed send rolls back and stays retryable.
// Telegram has no idempotency key: a crash after delivery but before commit can
// still duplicate a message.
func deliverNext(ctx context.Context, db *sql.DB, chain string, send func(context.Context, string) error) (bool, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var id, message string
	err = tx.QueryRowContext(ctx, `SELECT id, message FROM transfer_alerts
		WHERE sent_at IS NULL AND split_part(id, '/', 1) = $1
		ORDER BY created_at, id LIMIT 1 FOR UPDATE SKIP LOCKED`, chain).Scan(&id, &message)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := send(ctx, message); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE transfer_alerts SET sent_at = now() WHERE id = $1`, id); err != nil {
		return false, err
	}
	return true, tx.Commit()
}
