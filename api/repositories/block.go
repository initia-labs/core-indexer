package repositories

import (
	"time"

	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

type blockRepository struct {
	db *gorm.DB
}

func NewBlockRepository(db *gorm.DB) BlockRepository {
	return &blockRepository{
		db: db,
	}
}

func (r *blockRepository) GetBlockHeightLatest() (*int32, error) {
	var record db.Tracking

	err := r.db.
		Table(db.TableNameTracking).
		Select("latest_informative_block_height").
		First(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("GetBlockHeightLatest: failed to fetch latest informative block height")
		return nil, err
	}

	return &record.LatestInformativeBlockHeight, nil
}

func (r *blockRepository) GetBlockTimestamp(latestBlockHeight *int32) ([]time.Time, error) {
	var record []db.Block

	err := r.db.Table(db.TableNameBlock).
		Select("timestamp").
		Where("height <= ?", latestBlockHeight).
		Order("height DESC").
		Limit(100).Find(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query block timestamps")
		return nil, err
	}

	timestamps := make([]time.Time, len(record))
	for idx, b := range record {
		timestamps[idx] = b.Timestamp
	}

	return timestamps, nil
}
