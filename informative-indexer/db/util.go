package db

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	MaxPostgresParams = math.MaxUint16 // Max PostgreSQL limit
)

func getColumns(t interface{}) []string {
	var fieldNames []string
	tType := reflect.TypeOf(t)
	if tType.Kind() == reflect.Ptr {
		tType = tType.Elem()
	}
	if tType.Kind() != reflect.Struct {
		panic(fmt.Errorf("provided type is not a struct or pointer to a struct"))
	}
	for i := 0; i < tType.NumField(); i++ {
		field := tType.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			fieldNames = append(fieldNames, jsonTag)
		}
	}
	return fieldNames
}

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

func QueryRowWithTimeout(parentCtx context.Context, dbClient Queryable, query string, args ...interface{}) pgx.Row {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()
	result := dbClient.QueryRow(ctx, query, args...)
	return result
}

func ExecWithTimeout(parentCtx context.Context, dbClient Queryable, query string, args ...interface{}) (pgconn.CommandTag, error) {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	result, err := dbClient.Exec(ctx, query, args...)
	return result, err
}

func BulkInsert(parentCtx context.Context, dbTx Queryable, tableName string, columns []string, values [][]interface{}, extraArgs string) error {
	if len(values) == 0 || len(columns) == 0 {
		return nil
	}

	if len(columns) != len(values[0]) {
		return ErrorLengthMismatch
	}

	maxRowsPerBatch := MaxPostgresParams / len(columns)
	for start := 0; start < len(values); start += maxRowsPerBatch {
		end := start + maxRowsPerBatch
		if end > len(values) {
			end = len(values)
		}

		batchValues := values[start:end]
		query := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES %s %s",
			tableName, strings.Join(columns, ", "), generatePlaceholders(batchValues), extraArgs,
		)

		_, err := ExecWithTimeout(parentCtx, dbTx, query, flattenValues(batchValues)...)
		if err != nil {
			return err
		}
	}

	return nil
}

func QueryRowsWithTimeout(parentCtx context.Context, dbClient Queryable, query string, args ...interface{}) (pgx.Rows, error) {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	results, err := dbClient.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return results, err
}
