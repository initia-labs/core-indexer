package db

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

func GetLatestInformativeBlockHeight(ctx context.Context, dbClient *gorm.DB) (int64, error) {
	var tracking Tracking
	if err := dbClient.WithContext(ctx).First(&tracking).Error; err != nil {
		return 0, err
	}
	return tracking.LatestInformativeBlockHeight, nil
}

func IsTrackingInit(ctx context.Context, dbTx *gorm.DB) (bool, error) {
	var tracking Tracking
	if err := dbTx.WithContext(ctx).First(&tracking).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func InitTracking(ctx context.Context, dbTx *gorm.DB) error {
	tracking := Tracking{
		TxCount:                      0,
		LatestInformativeBlockHeight: 0,
	}
	return dbTx.WithContext(ctx).Create(&tracking).Error
}

func UpdateTxCount(ctx context.Context, dbTx *gorm.DB, txCount int64, height int64) error {
	var tracking Tracking
	if err := dbTx.WithContext(ctx).First(&tracking).Error; err != nil {
		return err
	}
	return dbTx.WithContext(ctx).
		Model(&tracking).
		Where("1 = 1").
		Update("tx_count", gorm.Expr("tx_count + ?", txCount)).
		Update("latest_informative_block_height", height).Error
}

func QueryLatestInformativeBlockHeight(ctx context.Context, dbTx *gorm.DB) (int64, error) {
	var tracking Tracking
	if err := dbTx.WithContext(ctx).First(&tracking).Error; err != nil {
		return 0, err
	}
	return int64(tracking.LatestInformativeBlockHeight), nil
}
