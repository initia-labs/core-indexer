package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	QueryTimeout = 5 * time.Minute

	ErrorNonRetryable   = errors.New("non-retryable error")
	ErrorLengthMismatch = errors.New("length mismatch")
)

func NewClient(databaseURL string) (*pgxpool.Pool, error) {
	return pgxpool.New(context.Background(), databaseURL)
}

type Queryable interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
	Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
}

func GetLatestBlockHeight(ctx context.Context, dbClient Queryable) (int64, error) {
	var height int64
	err := QueryRowWithTimeout(ctx, dbClient, "SELECT block_height FROM transaction_events ORDER BY block_height DESC LIMIT 1").Scan(&height)
	if err != nil {
		// Handle no rows found in `transaction_events`
		if errors.Is(err, pgx.ErrNoRows) {
			// Fallback to the latest indexed height from `blocks`
			err = QueryRowWithTimeout(ctx, dbClient, "SELECT height FROM blocks ORDER BY height DESC LIMIT 1").Scan(&height)
			if err == nil {
				return height, nil
			}
		}
		return 0, fmt.Errorf("failed to get latest block height: %w", err)
	}

	return height, nil
}

func InsertTransactionEventsIgnoreConflict(ctx context.Context, dbTx Queryable, txEvents []*TransactionEvent) error {
	span := sentry.StartSpan(ctx, "InsertTransactionEvents")
	span.Description = "Bulk insert transaction_events into the database"
	defer span.Finish()

	if len(txEvents) == 0 {
		return nil
	}

	columns := getColumns(txEvents[0])
	var values [][]interface{}
	for _, txEvent := range txEvents {
		values = append(values, []interface{}{
			txEvent.TransactionHash,
			txEvent.BlockHeight,
			txEvent.EventKey,
			txEvent.EventValue,
			txEvent.EventIndex,
		})
	}

	return BulkInsert(ctx, dbTx, "transaction_events", columns, values, "ON CONFLICT DO NOTHING")
}

func InsertFinalizeBlockEventsIgnoreConflict(ctx context.Context, dbTx Queryable, blockEvents []*FinalizeBlockEvent) error {
	span := sentry.StartSpan(ctx, "InsertFinalizeBlockEvents")
	span.Description = "Bulk insert finalize_block_events into the database"
	defer span.Finish()

	if len(blockEvents) == 0 {
		return nil
	}

	columns := getColumns(blockEvents[0])
	var values [][]interface{}
	for _, blockEvent := range blockEvents {
		values = append(values, []interface{}{
			blockEvent.BlockHeight,
			blockEvent.EventKey,
			blockEvent.EventValue,
			blockEvent.EventIndex,
			blockEvent.Mode,
		})
	}

	return BulkInsert(ctx, dbTx, "finalize_block_events", columns, values, "ON CONFLICT DO NOTHING")
}

func GetRowCount(ctx context.Context, dbClient Queryable) (int64, error) {
	var count int64
	err := QueryRowWithTimeout(ctx, dbClient, "SELECT COUNT(*) FROM transaction_events").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get row count: %w", err)
	}
	return count, nil
}

func GetRowsToPrune(ctx context.Context, dbClient Queryable, threshold int64) (pgx.Rows, error) {
	query := `SELECT transaction_hash, block_height, event_key, event_value, event_index FROM transaction_events WHERE block_height <= $1`
	rows, err := QueryRowsWithTimeout(ctx, dbClient, query, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows to prunig: %w", err)
	}
	return rows, err
}

func DeleteRowsToPrune(ctx context.Context, dbClient Queryable, threshold int64) error {
	query := `DELETE FROM transaction_events WHERE block_height <= $1`
	_, err := ExecWithTimeout(ctx, dbClient, query, threshold)
	if err != nil {
		return err
	}
	return nil
}
