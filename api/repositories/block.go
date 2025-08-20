package repositories

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/initia-labs/core-indexer/api/dto"
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
	var record db.Block

	if err := r.db.
		Model(&db.Block{}).
		Select("height").
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "height",
			},
			Desc: true,
		}).
		First(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("GetBlockHeightLatest: failed to fetch latest informative block height")
		return nil, err
	}

	latestHeight := int64(record.Height)

	return &latestHeight, nil
}

func (r *BlockRepository) GetBlockHeightInformativeLatest() (*int64, error) {
	var record db.Tracking

	if err := r.db.
		Model(&db.Tracking{}).
		Select("latest_informative_block_height").
		First(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("GetBlockHeightLatest: failed to fetch latest informative block height")
		return nil, err
	}

	latestHeight := int64(record.LatestInformativeBlockHeight)

	return &latestHeight, nil
}

func (r *BlockRepository) GetBlockTimestamp(latestBlockHeight int64) ([]time.Time, error) {
	var record []db.Block

	if err := r.db.Model(&db.Block{}).
		Select("timestamp").
		Where("height <= ?", latestBlockHeight).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "height",
			},
			Desc: true,
		}).
		Limit(100).Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query block timestamps")
		return nil, err
	}

	timestamps := make([]time.Time, len(record))
	for idx, b := range record {
		timestamps[idx] = b.Timestamp
	}

	return timestamps, nil
}

func (r *BlockRepository) GetBlocks(pagination dto.PaginationQuery) ([]dto.BlockModel, int64, error) {
	record := make([]dto.BlockModel, 0)
	total := int64(0)

	if err := r.db.Model(&db.Block{}).
		Select(`
			blocks.hash,
			blocks.height,
			blocks.timestamp,
			validators.moniker,
			validators.operator_address,
			validators.identity,
			(
				SELECT COUNT(*)
				FROM transactions
				WHERE blocks.height = transactions.block_height
			) AS tx_count
		`).
		Joins("LEFT JOIN validators ON blocks.proposer = validators.operator_address").
		Where("blocks.height >= ?", 1).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "blocks.height",
			},
			Desc: pagination.Reverse,
		}).
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query blocks")
		return nil, 0, err
	}

	if pagination.CountTotal {
		latestHeight, err := r.GetBlockHeightLatest()
		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to get total block count")
			return nil, 0, err
		}

		total = *latestHeight
	}

	return record, total, nil
}

func (r *BlockRepository) GetBlockInfo(height int64) (*dto.BlockInfoModel, error) {
	var record dto.BlockInfoModel

	if err := r.db.Model(&db.Block{}).
		Select(`
			blocks.hash,
			blocks.height,
			blocks.timestamp,
			validators.moniker,
			validators.operator_address,
			validators.identity,
			SUM(transactions.gas_used) AS gas_used,
			SUM(transactions.gas_limit) AS gas_limit
		`).
		Joins("LEFT JOIN validators ON blocks.proposer = validators.operator_address").
		Joins("LEFT JOIN transactions ON blocks.height = transactions.block_height").
		Where("blocks.height = ?", height).
		Clauses(clause.GroupBy{
			Columns: []clause.Column{
				{Name: "blocks.hash"},
				{Name: "blocks.height"},
				{Name: "blocks.timestamp"},
				{Name: "validators.moniker"},
				{Name: "validators.operator_address"},
				{Name: "validators.identity"},
			},
		}).
		First(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query block info")
		return nil, err
	}

	return &record, nil
}

func (r *BlockRepository) GetBlockTxs(pagination dto.PaginationQuery, height int64) ([]dto.BlockTxModel, int64, error) {
	record := make([]dto.BlockTxModel, 0)
	total := int64(0)

	if err := r.db.Model(&db.Transaction{}).
		Select(`
			blocks.height,
			blocks.timestamp,
			accounts.address,
			transactions.hash,
			transactions.success,
			transactions.messages,
			transactions.is_send,
			transactions.is_ibc,
			transactions.is_opinit
		`).
		Joins("LEFT JOIN blocks ON blocks.height = transactions.block_height").
		Joins("LEFT JOIN accounts ON accounts.address = transactions.sender").
		Where("blocks.height = ?", height).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "transactions.block_index",
			},
			Desc: pagination.Reverse,
		}).
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query block transactions")
		return nil, 0, err
	}

	if pagination.CountTotal {
		if err := r.db.Model(&db.Transaction{}).
			Joins("LEFT JOIN blocks ON blocks.height = transactions.block_height").
			Where("blocks.height = ?", height).
			Count(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("Failed to get total block transaction count")
			return nil, 0, err
		}
	}

	return record, total, nil
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
