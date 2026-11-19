package alerts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/initia-labs/core-indexer/pkg/mq"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

func Run(ctx context.Context, c Config) error {
	logger := log.With().Str("component", "transfer-alerts").Str("chain", c.Chain).Str("id", os.Getenv("ID")).Logger()
	ctx = logger.WithContext(ctx)
	logger.Info().Int("workers", c.Workers).Int("batch_size", c.BatchSize).Int64("start_height", c.StartHeight).
		Int("assets", len(c.Assets)).Int("ibc_routes", len(c.IBC)).Int("escrows", len(c.Escrows)).Int("layerzero_routes", len(c.LayerZero)).
		Msg("Starting transfer alert worker")
	for _, name := range []string{"DB_CONNECTION_STRING", "TELEGRAM_BOT_TOKEN", "TELEGRAM_CHAT_ID"} {
		if os.Getenv(name) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	db, err := sql.Open("pgx", os.Getenv("DB_CONNECTION_STRING"))
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(c.Workers + 1)
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	logger.Info().Msg("Connected to transfer alert database")
	telegram := Telegram{Token: os.Getenv("TELEGRAM_BOT_TOKEN"), ChatID: os.Getenv("TELEGRAM_CHAT_ID"), Client: &http.Client{Timeout: 15 * time.Second}}
	group, ctx := errgroup.WithContext(ctx)
	for worker := range c.Workers {
		group.Go(func() error {
			lastStatus := time.Now()
			for ctx.Err() == nil {
				n, err := scanBatch(ctx, db, c)
				if err != nil {
					return err
				}
				if worker == 0 && time.Since(lastStatus) >= 30*time.Second {
					logger.Info().Int("groups_in_last_batch", n).Msg("Transfer alert polling active")
					lastStatus = time.Now()
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
	group.Go(func() error { return runDelivery(ctx, db, c.Chain, telegram.Send, wait) })
	err = group.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Error().Err(err).Msg("Transfer alert worker stopped")
	} else {
		logger.Info().Msg("Transfer alert worker stopped")
	}
	return err
}

type deliveryError struct{ error }

func runDelivery(ctx context.Context, db *sql.DB, chain string, send func(context.Context, string) error, pause func(context.Context, time.Duration) error) error {
	delay := 3 * time.Second
	for ctx.Err() == nil {
		_, err := deliverNext(ctx, db, chain, send)
		var sendErr *deliveryError
		if err != nil && !errors.As(err, &sendErr) {
			return err // DB failures still stop the worker.
		}
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Dur("retry_in", delay).Msg("Transfer alert delivery failed; retrying")
		} else {
			delay = 3 * time.Second
		}
		if err := pause(ctx, delay); err != nil {
			return err
		}
		if sendErr != nil {
			delay = min(delay*2, time.Minute)
		}
	}
	return ctx.Err()
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
		WHERE chain_id = $1 AND block_height >= $2 AND evaluated_at IS NULL AND evaluation_error IS NULL
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
	matchingAlerts, failedGroups := 0, 0
	for _, b := range batches {
		var events []abci.Event
		evaluationErr := json.Unmarshal(b.events, &events)
		block := mq.BlockResultMsg{Height: b.height}
		if b.hash == "" {
			block.FinalizeBlockEvents = events
		} else {
			block.Txs = []mq.TxResult{{Hash: b.hash, ExecTxResults: &abci.ExecTxResult{Events: events}}}
		}
		var alerts []Alert
		if evaluationErr == nil {
			alerts, evaluationErr = c.Evaluate(block)
		}
		if evaluationErr != nil {
			if _, err := tx.ExecContext(ctx, `UPDATE transfer_alert_batches SET evaluation_error = $4
				WHERE chain_id = $1 AND block_height = $2 AND transaction_hash = $3`, c.Chain, b.height, b.hash, evaluationErr.Error()); err != nil {
				return 0, err
			}
			failedGroups++
			log.Ctx(ctx).Error().Err(evaluationErr).Int64("height", b.height).Str("transaction_hash", b.hash).Msg("Transfer alert evaluation failed")
			continue
		}
		matchingAlerts += len(alerts)
		for _, a := range alerts {
			if _, err := tx.ExecContext(ctx, `INSERT INTO transfer_alerts (id, chain_id, message, telegram_message) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`, a.ID, c.Chain, a.Text, a.TelegramText); err != nil {
				return 0, err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE transfer_alert_batches SET evaluated_at = now()
			WHERE chain_id = $1 AND block_height = $2 AND transaction_hash = $3`, c.Chain, b.height, b.hash); err != nil {
			return 0, err
		}
	}
	err = tx.Commit()
	if err == nil && len(batches) > 0 {
		log.Ctx(ctx).Info().Int("groups", len(batches)).Int("matching_alerts", matchingAlerts).Int("failed_groups", failedGroups).
			Int64("from_height", batches[0].height).Int64("to_height", batches[len(batches)-1].height).
			Msg("Processed transfer alert batch")
	}
	return len(batches), err
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
	err = tx.QueryRowContext(ctx, `SELECT id, COALESCE(telegram_message, message) FROM transfer_alerts
		WHERE sent_at IS NULL AND chain_id = $1
		ORDER BY created_at, id LIMIT 1 FOR UPDATE SKIP LOCKED`, chain).Scan(&id, &message)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := send(ctx, message); err != nil {
		return false, &deliveryError{err}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE transfer_alerts SET sent_at = now() WHERE id = $1`, id); err != nil {
		return false, err
	}
	err = tx.Commit()
	if err == nil {
		log.Ctx(ctx).Info().Str("alert_id", id).Msg("Sent transfer alert to Telegram")
	}
	return true, err
}
