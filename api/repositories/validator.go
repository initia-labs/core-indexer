package repositories

import (
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

func (r *ValidatorRepository) GetValidators(pagination dto.PaginationQuery, isActive bool, ignoreIsActive bool, sortBy, search string) ([]dto.ValidatorWithVoteCountModel, int64, error) {
	record := make([]dto.ValidatorWithVoteCountModel, 0)
	total := int64(0)

	query := r.db.Model(&db.Validator{}).
		Select("validators.*, validator_vote_counts.last_100 AS last_100").
		Joins("LEFT JOIN validator_vote_counts ON validators.operator_address = validator_vote_counts.validator_address")

	if !ignoreIsActive {
		query = query.Where("is_active = ?", isActive)
	}

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
		countQuery := r.db.Model(&db.Validator{})
		if !ignoreIsActive {
			countQuery = countQuery.Where("is_active = ?", isActive)
		}
		if search != "" {
			countQuery = countQuery.Where("moniker ILIKE ? OR operator_address = ?", "%"+search+"%", search)
		}

		var err error
		total, err = utils.CountWithTimeout(countQuery, r.countQueryTimeout)
		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count validators")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *ValidatorRepository) GetValidatorsByPower(pagination *dto.PaginationQuery, onlyActive bool) ([]db.Validator, error) {
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

func (r *ValidatorRepository) GetValidatorRow(operatorAddr string) (*db.Validator, error) {
	var record db.Validator

	if err := r.db.Model(&db.Validator{}).
		Where("operator_address = ?", operatorAddr).
		First(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to find validator of operator address: %s", operatorAddr)
		return nil, err
	}

	return &record, nil
}

func (r *ValidatorRepository) GetValidatorBlockVoteByBlockLimit(minHeight, maxHeight int64) ([]dto.ValidatorBlockVoteModel, error) {
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

func (r *ValidatorRepository) GetValidatorCommitSignatures(operatorAddr string, minHeight, maxHeight int64) ([]dto.ValidatorBlockVoteModel, error) {
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

func (r *ValidatorRepository) GetValidatorSlashEvents(operatorAddr string, minTimestamp time.Time) ([]dto.ValidatorUptimeEventModel, error) {
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

func (r *ValidatorRepository) GetValidatorUptimeInfo(operatorAddr string) (*dto.ValidatorWithVoteCountModel, error) {
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

func (r *ValidatorRepository) GetValidatorBondedTokenChanges(pagination dto.PaginationQuery, operatorAddr string) ([]db.ValidatorBondedTokenChange, int64, error) {
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
		total, err = utils.CountWithTimeout(r.db.Model(&db.ValidatorBondedTokenChange{}).Where("validator_address = ?", operatorAddr), r.countQueryTimeout)
		if err != nil {
			logger.Get().Error().Err(err).Msgf("Failed to count validator bonded token changes for %s", operatorAddr)
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *ValidatorRepository) GetValidatorProposedBlocks(pagination dto.PaginationQuery, operatorAddr string) ([]dto.ValidatorProposedBlockModel, int64, error) {
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
		total, err = utils.CountWithTimeout(r.db.Model(&db.Block{}).Where("proposer = ? AND timestamp >= ?", operatorAddr, since), r.countQueryTimeout)
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

func (r *ValidatorRepository) GetValidatorHistoricalPowers(operatorAddr string) ([]dto.ValidatorHistoricalPowerModel, int64, error) {
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

func (r *ValidatorRepository) GetValidatorBlockStats(operatorAddresses []string) (map[string]struct{ TotalBlocks, SignedBlocks int64 }, error) {
	result := make(map[string]struct{ TotalBlocks, SignedBlocks int64 })

	if len(operatorAddresses) == 0 {
		return result, nil
	}

	// Get latest block height
	var latestBlock db.Block
	if err := r.db.Model(&db.Block{}).
		Order("height DESC").
		Limit(1).
		First(&latestBlock).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query latest block")
		return nil, err
	}

	latestHeight := int64(latestBlock.Height)
	minHeight := latestHeight - 99 // Last 100 blocks
	if minHeight < 1 {
		minHeight = 1
	}

	// Get total proposed blocks for each validator in the last 100 blocks
	var proposedBlocks []struct {
		ValidatorAddress string `gorm:"column:proposer"`
		Count            int64  `gorm:"column:count"`
	}

	if err := r.db.Model(&db.Block{}).
		Select("proposer, COUNT(*) as count").
		Where("proposer IN ? AND height >= ? AND height <= ?", operatorAddresses, minHeight, latestHeight).
		Group("proposer").
		Find(&proposedBlocks).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query validator proposed blocks count")
		return nil, err
	}

	// Initialize result map with proposed blocks
	for _, pb := range proposedBlocks {
		result[pb.ValidatorAddress] = struct{ TotalBlocks, SignedBlocks int64 }{
			TotalBlocks:  pb.Count,
			SignedBlocks: 0,
		}
	}

	// Get signed blocks (VOTE and PROPOSE) for each validator in the last 100 blocks
	var signedBlocks []struct {
		ValidatorAddress string `gorm:"column:validator_address"`
		Count            int64  `gorm:"column:count"`
	}

	if err := r.db.Model(&db.ValidatorCommitSignature{}).
		Select("validator_address, COUNT(*) as count").
		Where("validator_address IN ? AND block_height >= ? AND block_height <= ? AND vote IN ?", operatorAddresses, minHeight, latestHeight, []string{"VOTE", "PROPOSE"}).
		Group("validator_address").
		Find(&signedBlocks).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query validator signed blocks count")
		return nil, err
	}

	// Update result map with signed blocks
	for _, sb := range signedBlocks {
		if stats, exists := result[sb.ValidatorAddress]; exists {
			stats.SignedBlocks = sb.Count
			result[sb.ValidatorAddress] = stats
		} else {
			result[sb.ValidatorAddress] = struct{ TotalBlocks, SignedBlocks int64 }{
				TotalBlocks:  0,
				SignedBlocks: sb.Count,
			}
		}
	}

	return result, nil
}
