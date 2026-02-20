package db

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func InsertGenesisBlock(ctx context.Context, dbTx *gorm.DB, timestamp time.Time) error {
	err := dbTx.WithContext(ctx).Exec(`
		INSERT INTO blocks (height, hash, timestamp, proposer) 
		VALUES (?, ?, ?, ?) 
		ON CONFLICT DO NOTHING
	`, 0, []byte("GENESIS"), timestamp, nil).Error
	if err != nil {
		return err
	}
	return nil
}

func InsertBlockIgnoreConflict(ctx context.Context, dbTx *gorm.DB, block Block) error {
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		Create(&block)
	return result.Error
}

func UpsertBlock(ctx context.Context, dbTx *gorm.DB, block Block) error {
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "height"}},
			DoUpdates: clause.AssignmentColumns([]string{"proposer"}),
		}).
		Create(&block)
	return result.Error
}
