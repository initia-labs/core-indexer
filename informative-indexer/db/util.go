package db

import (
	"context"
	"github.com/jackc/pgx/v5"
)

func QueryRowWithTimeout(parentCtx context.Context, dbClient Queryable, query string, args ...interface{}) pgx.Row {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()
	result := dbClient.QueryRow(ctx, query, args...)
	return result
}
