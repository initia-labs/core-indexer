package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func flattenValues(values [][]interface{}) []interface{} {
	flatValues := make([]interface{}, 0)
	for _, value := range values {
		flatValues = append(flatValues, value...)
	}
	return flatValues
}

func generatePlaceholders(values [][]interface{}) string {
	placeholders := make([]string, 0)
	valueIdx := 1
	for i := range values {
		placeholder := make([]string, 0)
		for range values[i] {
			placeholder = append(placeholder, fmt.Sprintf("$%d", valueIdx))
			valueIdx++
		}
		placeholders = append(placeholders, fmt.Sprintf("(%s)", strings.Join(placeholder, ", ")))
	}
	return strings.Join(placeholders, ", ")
}

func ExecWithTimeout(parentCtx context.Context, dbClient Queryable, query string, args ...interface{}) (pgconn.CommandTag, error) {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	result, err := dbClient.Exec(ctx, query, args...)
	return result, err
}

func QueryRowWithTimeout(parentCtx context.Context, dbClient Queryable, query string, args ...interface{}) pgx.Row {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()
	result := dbClient.QueryRow(ctx, query, args...)
	return result
}

func QueryWithTimeout(parentCtx context.Context, dbClient Queryable, query string, args ...interface{}) (pgx.Rows, error, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	result, err := dbClient.Query(ctx, query, args...)
	return result, err, cancel
}

func BulkInsert(parentCtx context.Context, dbTx Queryable, tableName string, columns []string, values [][]interface{}, extraArgs string) error {
	if len(values) == 0 || len(columns) == 0 {
		return nil
	}

	if len(columns) != len(values[0]) {
		return ErrorLengthMismatch
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s %s", tableName, strings.Join(columns, ", "), generatePlaceholders(values), extraArgs)
	_, err := ExecWithTimeout(parentCtx, dbTx, query, flattenValues(values)...)
	if err != nil {
		return err
	}

	return nil
}
