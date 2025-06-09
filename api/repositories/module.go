package repositories

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

// moduleRepository implements ModuleRepository
type moduleRepository struct {
	db *gorm.DB
}

func NewModuleRepository(db *gorm.DB) ModuleRepository {
	return &moduleRepository{
		db: db,
	}
}

// GetModules retrieves modules with pagination
func (r *moduleRepository) GetModules(pagination dto.PaginationQuery) ([]dto.ModuleResponse, int64, error) {
	var modules []dto.ModuleResponse
	var total int64

	query := r.db.Model(&db.Module{})

	// TODO: Consider optimizing this query
	err := query.
		Select(
			"name AS module_name",
			"digest",
			"is_verify",
			"publisher_id AS address",
			"block_info.height",
			"block_info.timestamp AS latest_updated",
			"(SELECT COUNT(*) > 1 FROM module_histories WHERE module_histories.module_id = modules.id) AS is_republished",
		).
		Joins(
			"LEFT JOIN LATERAL (SELECT blocks.height, blocks.timestamp FROM module_histories JOIN blocks ON blocks.height = module_histories.block_height WHERE module_histories.module_id = modules.id ORDER BY module_histories.block_height DESC LIMIT 1) AS block_info ON true",
		).
		Order("(SELECT MAX(block_height) FROM module_histories WHERE module_histories.module_id = modules.id) DESC").
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset)).
		Find(&modules).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query modules")
		return nil, 0, err
	}

	// Get total count if requested
	if pagination.CountTotal {
		if err := query.Count(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count modules")
			return nil, 0, err
		}
	}

	return modules, total, nil
}

func (r *moduleRepository) GetModuleById(vmAddress string, name string) (*dto.ModuleResponse, error) {
	var module dto.ModuleResponse

	query := r.db.Model(&db.Module{})

	// TODO: Consider optimizing this query
	err := query.
		Select(
			"name AS module_name",
			"digest",
			"is_verify",
			"publisher_id AS address",
			"block_info.height",
			"block_info.timestamp AS latest_updated",
			"(SELECT COUNT(*) > 1 FROM module_histories WHERE module_histories.module_id = modules.id) AS is_republished",
		).
		Joins(
			"LEFT JOIN LATERAL (SELECT blocks.height, blocks.timestamp FROM module_histories JOIN blocks ON blocks.height = module_histories.block_height WHERE module_histories.module_id = modules.id ORDER BY module_histories.block_height DESC LIMIT 1) AS block_info ON true",
		).
		Where("modules.id = ?", vmAddress+"::"+name).
		First(&module).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NewNotFound(fmt.Sprintf("module %s::%s not found", vmAddress, name))
		}
		logger.Get().Error().Err(err).Msg("Failed to query module")
		return nil, err
	}

	return &module, nil
}

func (r *moduleRepository) GetModuleHistories(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleHistoryResponse, int64, error) {
	var histories []dto.ModuleHistoryResponse
	var total int64

	moduleId := fmt.Sprintf("%s::%s", vmAddress, name)

	err := r.db.Model(&db.ModuleHistory{}).
		Select(
			"module_histories.remark",
			"module_histories.upgrade_policy",
			"block_height as height",
			"blocks.timestamp",
		).
		Joins("LEFT JOIN blocks ON blocks.height = module_histories.block_height").
		Where("module_histories.module_id = ?", moduleId).
		Order("module_histories.block_height DESC").
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset)).
		Find(&histories).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query module histories")
		return nil, 0, err
	}

	if pagination.CountTotal {
		if err := r.db.Model(&db.ModuleHistory{}).
			Where("module_histories.module_id = ?", moduleId).
			Count(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count module histories")
			return nil, 0, err
		}
	}

	return histories, total, nil
}

func (r *moduleRepository) GetModulePublishInfo(vmAddress string, name string) ([]dto.ModulePublishInfoModel, error) {
	var modulePublishInfos []dto.ModulePublishInfoModel

	moduleId := fmt.Sprintf("%s::%s", vmAddress, name)

	err := r.db.Model(&db.ModuleHistory{}).
		Select(
			"\\x || encode(transactions.hash::bytea, 'hex') as transaction_hash",
			"blocks.timestamp",
			"proposals.title as proposal_title",
			"proposals.id as proposal_id",
			"module_histories.block_height as height",
		).
		Joins("LEFT JOIN transactions ON transactions.id = module_histories.tx_id").
		Joins("LEFT JOIN blocks ON blocks.height = transactions.block_height").
		Joins("LEFT JOIN proposals ON proposals.id = module_histories.proposal_id").
		Where("module_histories.module_id = ?", moduleId).
		Limit(2).
		Order("module_histories.block_height DESC").
		Find(&modulePublishInfos).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query module publish info")
		return nil, err
	}

	return modulePublishInfos, nil
}
