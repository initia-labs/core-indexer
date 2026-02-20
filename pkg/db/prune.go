package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

func GetRowCount(ctx context.Context, dbClient *gorm.DB, table string) (int64, error) {
	if !isValidTableName(table) {
		return 0, fmt.Errorf("invalid table name: %s", table)
	}
	var count int64
	result := dbClient.WithContext(ctx).
		Table(table).
		Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to get row count for table %s: %w", table, result.Error)
	}
	return count, nil
}

func BuildPruneQuery(ctx context.Context, dbClient *gorm.DB, table string, threshold int64) (*gorm.DB, error) {
	if !isValidTableName(table) {
		return nil, fmt.Errorf("invalid table name: %s", table)
	}
	query := dbClient.WithContext(ctx).
		Table(table).
		Where("block_height <= ?", threshold)
	return query, nil
}

func DeleteRowsToPrune(ctx context.Context, dbClient *gorm.DB, table string, threshold int64) error {
	if !isValidTableName(table) {
		return fmt.Errorf("invalid table name: %s", table)
	}
	result := dbClient.WithContext(ctx).
		Table(table).
		Where("block_height <= ?", threshold).
		Delete(nil)
	return result.Error
}
