package repositories

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

var _ BlockRepositoryI = &BlockRepository{}

type BlockRepository struct {
	db *gorm.DB
}

func NewBlockRepository(db *gorm.DB) *BlockRepository {
	return &BlockRepository{
		db: db,
	}
}

func (r *BlockRepository) GetBlockHeightLatest() (*int64, error) {
	var record db.Tracking

	err := r.db.
		Model(&db.Tracking{}).
		Select("latest_informative_block_height").
		First(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("GetBlockHeightLatest: failed to fetch latest informative block height")
		return nil, err
	}

	latestHeight := int64(record.LatestInformativeBlockHeight)

	return &latestHeight, nil
}

func (r *BlockRepository) GetBlockTimestamp(latestBlockHeight int64) ([]time.Time, error) {
	var record []db.Block

	err := r.db.Model(&db.Block{}).
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

func (r *BlockRepository) GetLatestBlock() (*db.Block, error) {
	var block db.Block

	if err := r.db.Model(&db.Block{}).
		Limit(1).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "height",
			},
			Desc: true,
		}).
		First(&block).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query latest block")
		return nil, err
	}

	return &block, nil
}
