package db

import (
	"context"
	"errors"
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
		return 0, err
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
