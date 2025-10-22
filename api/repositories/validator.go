package repositories

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/utils"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

var _ ValidatorRepositoryI = &ValidatorRepository{}

// ValidatorRepository implements ValidatorRepositoryI
type ValidatorRepository struct {
	db                *gorm.DB
	countQueryTimeout time.Duration
}

func NewValidatorRepository(db *gorm.DB, countQueryTimeout time.Duration) *ValidatorRepository {
	return &ValidatorRepository{
		db:                db,
		countQueryTimeout: countQueryTimeout,
	}
}

func (r *ValidatorRepository) GetValidators(ctx context.Context, pagination dto.PaginationQuery, isActive bool, sortBy, search string) ([]dto.ValidatorWithVoteCountModel, int64, error) {
	record := make([]dto.ValidatorWithVoteCountModel, 0)
	total := int64(0)

	query := r.db.Model(&db.Validator{}).
		Select("validators.*, validator_vote_counts.last_100 AS last_100").
		Where("is_active = ?", isActive).
		Joins("LEFT JOIN validator_vote_counts ON validators.operator_address = validator_vote_counts.validator_address")

	if search != "" {
		query = query.Where("moniker ILIKE ? OR operator_address = ?", "%"+search+"%", search)
	}

	orders := make([]clause.OrderByColumn, 0)

	if sortBy == "uptime" && isActive {
		if pagination.Reverse {
			orders = append(orders, clause.OrderByColumn{
				Column: clause.Column{Name: "validator_vote_counts.last_100 DESC NULLS LAST", Raw: true},
			})
		} else {
			orders = append(orders, clause.OrderByColumn{
				Column: clause.Column{Name: "validator_vote_counts.last_100 ASC NULLS FIRST", Raw: true},
			})
		}

		orders = append(orders, clause.OrderByColumn{
			Column: clause.Column{Name: "voting_power"},
			Desc:   true,
		})
		orders = append(orders, clause.OrderByColumn{
			Column: clause.Column{Name: "moniker"},
			Desc:   false,
		})
	} else if sortBy == "commission" {
		orders = append(
			orders,
			clause.OrderByColumn{
				Column: clause.Column{Name: "commission_rate"},
				Desc:   pagination.Reverse,
			},
			clause.OrderByColumn{
				Column: clause.Column{Name: "voting_power"},
				Desc:   true,
			},
			clause.OrderByColumn{
				Column: clause.Column{Name: "moniker"},
				Desc:   false,
			},
		)
	} else if sortBy == "moniker" {
		orders = append(
			orders,
			clause.OrderByColumn{
				Column: clause.Column{Name: "moniker"},
				Desc:   pagination.Reverse,
			},
			clause.OrderByColumn{
				Column: clause.Column{Name: "voting_power"},
				Desc:   true,
			},
		)
	} else {
		orders = append(
			orders,
			clause.OrderByColumn{
				Column: clause.Column{Name: "voting_power"},
				Desc:   pagination.Reverse,
			},
			clause.OrderByColumn{
				Column: clause.Column{Name: "moniker"},
				Desc:   false,
			},
		)
	}

	query = query.Clauses(clause.OrderBy{
		Columns: orders,
	})

	// Fetch paginated results
	if err := query.
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query validators")
		return nil, 0, err
	}

	if pagination.CountTotal {
		countQuery := r.db.Model(&db.Validator{}).Where("is_active = ?", isActive)
		if search != "" {
			countQuery = countQuery.Where("moniker ILIKE ? OR operator_address = ?", "%"+search+"%", search)
		}

		var err error
		total, err = utils.CountWithTimeout(ctx, countQuery, r.countQueryTimeout)
		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count validators")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *ValidatorRepository) GetValidatorsByPower(ctx context.Context, pagination *dto.PaginationQuery, onlyActive bool) ([]db.Validator, error) {
	record := make([]db.Validator, 0)

	query := r.db.Model(&db.Validator{})
	if onlyActive {
		query = query.Where("is_active = true")
	}
	if pagination != nil {
		query = query.
			Limit(pagination.Limit).
			Offset(pagination.Offset)
	}

	if err := query.
		Order(clause.OrderBy{
			Columns: []clause.OrderByColumn{
				{
					Column: clause.Column{Name: "voting_power"},
					Desc:   true,
				},
				{
					Column: clause.Column{Name: "moniker"},
					Desc:   false,
				},
			},
		}).
		Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query validators by power")
		return nil, err
	}

	return record, nil
}

func (r *ValidatorRepository) GetValidatorRow(ctx context.Context, operatorAddr string) (*db.Validator, error) {
	var record db.Validator

	if err := r.db.Model(&db.Validator{}).
		Where("operator_address = ?", operatorAddr).
		First(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to find validator of operator address: %s", operatorAddr)
		return nil, err
	}

	return &record, nil
}

func (r *ValidatorRepository) GetValidatorBlockVoteByBlockLimit(ctx context.Context, minHeight, maxHeight int64) ([]dto.ValidatorBlockVoteModel, error) {
	var record []dto.ValidatorBlockVoteModel

	if err := r.db.Model(&db.ValidatorCommitSignature{}).
		Select("block_height as height, vote").
		Where("block_height <= ? AND block_height >= ? AND vote = ?", maxHeight, minHeight, "PROPOSE").
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "block_height",
			},
			Desc: true,
		}).Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query validator proposed blocks")
		return nil, err
	}

	return record, nil
}

func (r *ValidatorRepository) GetValidatorCommitSignatures(ctx context.Context, operatorAddr string, minHeight, maxHeight int64) ([]dto.ValidatorBlockVoteModel, error) {
	var record []dto.ValidatorBlockVoteModel

	if err := r.db.Model(&db.ValidatorCommitSignature{}).
		Select("block_height as height, vote").
		Where("validator_address = ? AND block_height <= ? AND block_height >= ?", operatorAddr, maxHeight, minHeight).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "block_height",
			},
			Desc: true,
		}).Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query validator commit signature for %s", operatorAddr)
		return nil, err
	}

	return record, nil
}

func (r *ValidatorRepository) GetValidatorSlashEvents(ctx context.Context, operatorAddr string, minTimestamp time.Time) ([]dto.ValidatorUptimeEventModel, error) {
	var record []dto.ValidatorUptimeEventModel

	if err := r.db.Model(&db.ValidatorSlashEvent{}).
		Select("blocks.height as height, blocks.timestamp as timestamp, validator_slash_events.type as type").
		Joins("JOIN blocks ON validator_slash_events.block_height = blocks.height").
		Where("validator_slash_events.validator_address = ? AND blocks.timestamp >= ?",
			operatorAddr, minTimestamp).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "block_height",
			},
			Desc: true,
		}).
		Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query validator slash events for %s", operatorAddr)
		return nil, err
	}

	return record, nil
}

func (r *ValidatorRepository) GetValidatorUptimeInfo(ctx context.Context, operatorAddr string) (*dto.ValidatorWithVoteCountModel, error) {
	var record dto.ValidatorWithVoteCountModel

	if err := r.db.Model(&db.Validator{}).
		Select("validators.*, validator_vote_counts.last_100 as last_100").
		Joins("LEFT JOIN validator_vote_counts ON validators.operator_address = validator_vote_counts.validator_address").
		Where("validators.operator_address = ?", operatorAddr).
		First(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query validator uptime for %s", operatorAddr)
		return nil, err
	}

	return &record, nil
}

func (r *ValidatorRepository) GetValidatorBondedTokenChanges(ctx context.Context, pagination dto.PaginationQuery, operatorAddr string) ([]db.ValidatorBondedTokenChange, int64, error) {
	var record []db.ValidatorBondedTokenChange
	var total int64

	if err := r.db.Model(&db.ValidatorBondedTokenChange{}).
		Preload("Transaction").
		Preload("Block").
		Where("validator_address = ?", operatorAddr).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "block_height",
			},
			Desc: pagination.Reverse,
		}).
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query validator bonded token changes for %s", operatorAddr)
		return nil, 0, err
	}

	if pagination.CountTotal {
		var err error
		total, err = utils.CountWithTimeout(ctx, r.db.Model(&db.ValidatorBondedTokenChange{}).Where("validator_address = ?", operatorAddr), r.countQueryTimeout)
		if err != nil {
			logger.Get().Error().Err(err).Msgf("Failed to count validator bonded token changes for %s", operatorAddr)
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *ValidatorRepository) GetValidatorProposedBlocks(ctx context.Context, pagination dto.PaginationQuery, operatorAddr string) ([]dto.ValidatorProposedBlockModel, int64, error) {
	var record []struct {
		Hash              []byte    `gorm:"column:hash"`
		Height            int32     `gorm:"column:height"`
		Timestamp         time.Time `gorm:"column:timestamp"`
		TransactionCount  int64     `gorm:"column:transaction_count"`
		ValidatorMoniker  string    `gorm:"column:moniker"`
		ValidatorIdentity string    `gorm:"column:identity"`
		ValidatorAddress  string    `gorm:"column:operator_address"`
	}
	var total int64

	since := time.Now().AddDate(0, 0, -30)
	if err := r.db.Model(&db.Block{}).
		Select("blocks.hash, blocks.height, blocks.timestamp, COUNT(transactions.id) as transaction_count, validators.moniker, validators.identity, validators.operator_address").
		Joins("LEFT JOIN transactions ON transactions.block_height = blocks.height").
		Joins("JOIN validators ON blocks.proposer = validators.operator_address").
		Where("blocks.proposer = ? AND blocks.timestamp >= ?", operatorAddr, since).
		Group("blocks.height, blocks.hash, blocks.timestamp, validators.moniker, validators.identity, validators.operator_address").
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "blocks.height",
			},
			Desc: pagination.Reverse,
		}).
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query proposed blocks for %s", operatorAddr)
		return nil, 0, err
	}

	if pagination.CountTotal {
		var err error
		total, err = utils.CountWithTimeout(ctx, r.db.Model(&db.Block{}).Where("proposer = ? AND timestamp >= ?", operatorAddr, since), r.countQueryTimeout)
		if err != nil {
			logger.Get().Error().Err(err).Msgf("Failed to count proposed blocks for %s", operatorAddr)
			return nil, 0, err
		}
	}

	result := make([]dto.ValidatorProposedBlockModel, len(record))
	for idx, block := range record {
		result[idx] = dto.ValidatorProposedBlockModel{
			Hash:             fmt.Sprintf("%x", block.Hash),
			Height:           int(block.Height),
			Timestamp:        block.Timestamp,
			TransactionCount: int(block.TransactionCount),
			Validator: dto.BlockProposer{
				Identity:        block.ValidatorIdentity,
				Moniker:         block.ValidatorMoniker,
				OperatorAddress: block.ValidatorAddress,
			},
		}
	}

	return result, total, nil
}

func (r *ValidatorRepository) GetValidatorHistoricalPowers(ctx context.Context, operatorAddr string) ([]dto.ValidatorHistoricalPowerModel, int64, error) {
	var record []dto.ValidatorHistoricalPowerModel

	since := time.Now().AddDate(0, 0, -90)
	if err := r.db.Model(db.ValidatorHistoricalPower{}).
		Select("hour_rounded_timestamp, timestamp, voting_power").
		Where("validator_address = ? AND timestamp >= ?", operatorAddr, since).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "timestamp",
			},
			Desc: false,
		}).Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query historical powers for %s", operatorAddr)
		return nil, 0, err
	}

	return record, int64(len(record)), nil
}
