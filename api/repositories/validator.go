package repositories

import (
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

var _ ValidatorRepositoryI = &ValidatorRepository{}

// ValidatorRepository implements ValidatorRepositoryI
type ValidatorRepository struct {
	db *gorm.DB
}

func NewValidatorRepository(db *gorm.DB) *ValidatorRepository {
	return &ValidatorRepository{
		db: db,
	}
}

func (r *ValidatorRepository) GetValidators(pagination dto.PaginationQuery, isActive bool, sortBy, search string) ([]dto.ValidatorWithVoteCountModel, int64, error) {
	validators := make([]dto.ValidatorWithVoteCountModel, 0)
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
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset)).
		Find(&validators).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query validators")
		return nil, 0, err
	}

	if pagination.CountTotal {
		countQuery := r.db.Model(&db.Validator{}).Where("is_active = ?", isActive)
		if search != "" {
			countQuery = countQuery.Where("moniker ILIKE ? OR operator_address = ?", "%"+search+"%", search)
		}

		if err := countQuery.Count(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count validators")
			return nil, 0, err
		}
	}

	return validators, total, nil
}

func (r *ValidatorRepository) GetValidatorsByPower(pagination *dto.PaginationQuery, onlyActive bool) ([]db.Validator, error) {
	validators := make([]db.Validator, 0)

	query := r.db.Model(&db.Validator{})
	if onlyActive {
		query = query.Where("is_active = true")
	}
	if pagination != nil {
		query = query.
			Limit(int(pagination.Limit)).
			Offset(int(pagination.Offset))
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
		Find(&validators).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query validators by power")
		return nil, err
	}

	return validators, nil
}

func (r *ValidatorRepository) GetValidatorRow(operatorAddr string) (*db.Validator, error) {
	var validator db.Validator

	if err := r.db.Model(&db.Validator{}).
		Where("operator_address = ?", operatorAddr).
		First(&validator).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to find validator of operator address: %s", operatorAddr)
		return nil, err
	}

	return &validator, nil
}

func (r *ValidatorRepository) GetValidatorBlockVoteByBlockLimit(minHeight, maxHeight int64) ([]dto.ValidatorBlockVote, error) {
	var proposedBlocks []dto.ValidatorBlockVote

	if err := r.db.Model(&db.ValidatorCommitSignature{}).
		Select("block_height as height, vote").
		Where("block_height <= ? AND block_height >= ? AND vote = ?", maxHeight, minHeight, "PROPOSE").
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "block_height",
			},
			Desc: true,
		}).Find(&proposedBlocks).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query validator proposed blocks")
		return nil, err
	}

	return proposedBlocks, nil
}

func (r *ValidatorRepository) GetValidatorCommitSignatures(operatorAddr string, minHeight, maxHeight int64) ([]dto.ValidatorBlockVote, error) {
	var signatures []dto.ValidatorBlockVote

	if err := r.db.Model(&db.ValidatorCommitSignature{}).
		Select("block_height as height, vote").
		Where("validator_address = ? AND block_height <= ? AND block_height >= ?", operatorAddr, maxHeight, minHeight).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "block_height",
			},
			Desc: true,
		}).Find(&signatures).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query validator commit signature for %s", operatorAddr)
		return nil, err
	}

	return signatures, nil
}

func (r *ValidatorRepository) GetValidatorSlashEvents(operatorAddr string, minTimestamp time.Time) ([]dto.ValidatorUptimeEvent, error) {
	var events []dto.ValidatorUptimeEvent

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
		Find(&events).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query validator slash events for %s", operatorAddr)
		return nil, err
	}

	return events, nil
}

func (r *ValidatorRepository) GetValidatorUptimeInfo(operatorAddr string) (*dto.ValidatorWithVoteCountModel, error) {
	var validatorInfo dto.ValidatorWithVoteCountModel

	if err := r.db.Model(&db.Validator{}).
		Select("validators.*, validator_vote_counts.last_100 as last_100").
		Joins("LEFT JOIN validator_vote_counts ON validators.operator_address = validator_vote_counts.validator_address").
		Where("validators.operator_address = ?", operatorAddr).
		First(&validatorInfo).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query validator uptime for %s", operatorAddr)
		return nil, err
	}

	return &validatorInfo, nil
}

func (r *ValidatorRepository) GetValidatorBondedTokenChanges(pagination dto.PaginationQuery, operatorAddr string) ([]db.ValidatorBondedTokenChange, int64, error) {
	var tokenChanges []db.ValidatorBondedTokenChange
	var total int64

	if err := r.db.Model(&db.ValidatorBondedTokenChange{}).
		Preload("Transaction").
		Preload("Block").
		Where("validator_address = ?", operatorAddr).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "block_height",
			},
			Desc: true,
		}).
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset)).
		Find(&tokenChanges).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query validator bonded token changes for %s", operatorAddr)
		return nil, 0, err
	}

	if err := r.db.Model(&db.ValidatorBondedTokenChange{}).
		Where("validator_address = ?", operatorAddr).
		Count(&total).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to count validator bonded token changes for %s", operatorAddr)
		return nil, 0, err
	}

	return tokenChanges, total, nil
}

func (r *ValidatorRepository) GetValidatorProposedBlocks(pagination dto.PaginationQuery, operatorAddr string) ([]dto.ValidatorProposedBlock, int64, error) {
	var blocks []struct {
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
			Desc: true,
		}).
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset)).
		Find(&blocks).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query proposed blocks for %s", operatorAddr)
		return nil, 0, err
	}

	if err := r.db.Model(&db.Block{}).
		Where("proposer = ? AND timestamp >= ?", operatorAddr, since).
		Count(&total).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to count proposed blocks for %s", operatorAddr)
		return nil, 0, err
	}

	result := make([]dto.ValidatorProposedBlock, len(blocks))
	for idx, block := range blocks {
		result[idx] = dto.ValidatorProposedBlock{
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

func (r *ValidatorRepository) GetValidatorHistoricalPowers(operatorAddr string) ([]dto.ValidatorHistoricalPower, int64, error) {
	var historicalPowers []dto.ValidatorHistoricalPower

	since := time.Now().AddDate(0, 0, -90)
	if err := r.db.Model(db.ValidatorHistoricalPower{}).
		Select("hour_rounded_timestamp, timestamp, voting_power").
		Where("validator_address = ? AND timestamp >= ?", operatorAddr, since).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "timestamp",
			},
			Desc: false,
		}).Find(&historicalPowers).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query historical powers for %s", operatorAddr)
		return nil, 0, err
	}

	return historicalPowers, int64(len(historicalPowers)), nil
}
