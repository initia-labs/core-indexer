package repositories

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/utils"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

var _ ModuleRepositoryI = &ModuleRepository{}

type ModuleRepository struct {
	db                *gorm.DB
	countQueryTimeout time.Duration
}

func NewModuleRepository(db *gorm.DB, countQueryTimeout time.Duration) *ModuleRepository {
	return &ModuleRepository{
		db:                db,
		countQueryTimeout: countQueryTimeout,
	}
}

// GetModules retrieves modules with pagination
func (r *ModuleRepository) GetModules(pagination dto.PaginationQuery) ([]dto.ModuleResponse, int64, error) {
	var modules []dto.ModuleResponse
	var total int64

	// TODO: Consider optimizing this query
	if err := r.db.Model(&db.Module{}).
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
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&modules).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query modules")
		return nil, 0, err
	}

	// Get total count if requested
	if pagination.CountTotal {
		var err error
		total, err = utils.CountWithTimeout(r.db.Model(&db.Module{}), r.countQueryTimeout)
		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count modules")
			return nil, 0, err
		}
	}

	return modules, total, nil
}

func (r *ModuleRepository) GetModuleById(vmAddress string, name string) (*dto.ModuleResponse, error) {
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("module %s::%s not found", vmAddress, name)
		}
		logger.Get().Error().Err(err).Msg("Failed to query module")
		return nil, err
	}

	return &module, nil
}

func (r *ModuleRepository) GetModuleHistories(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleHistoryResponse, int64, error) {
	var histories []dto.ModuleHistoryResponse
	var total int64

	moduleId := fmt.Sprintf("%s::%s", vmAddress, name)

	err := r.db.Model(&db.ModuleHistory{}).
		Select(
			"module_histories.remark",
			"module_histories.upgrade_policy",
			"block_height AS height",
			"blocks.timestamp",
		).
		Joins("LEFT JOIN blocks ON blocks.height = module_histories.block_height").
		Where("module_histories.module_id = ?", moduleId).
		Order("module_histories.block_height DESC").
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&histories).Error
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query module histories")
		return nil, 0, err
	}

	if pagination.CountTotal {
		var err error
		total, err = utils.CountWithTimeout(r.db.Model(&db.ModuleHistory{}).Where("module_histories.module_id = ?", moduleId), r.countQueryTimeout)
		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count module histories")
			return nil, 0, err
		}
	}

	return histories, total, nil
}

func (r *ModuleRepository) GetModulePublishInfo(vmAddress string, name string) ([]dto.ModulePublishInfoModel, error) {
	var modulePublishInfos []dto.ModulePublishInfoModel

	moduleId := fmt.Sprintf("%s::%s", vmAddress, name)

	err := r.db.Model(&db.ModuleHistory{}).
		Select(
			"transactions.hash",
			"blocks.timestamp",
			"proposals.id AS proposal_proposal_id",
			"proposals.title AS proposal_proposal_title",
			"module_histories.block_height AS height",
		).
		Joins("LEFT JOIN transactions ON transactions.id = module_histories.tx_id").
		Joins("LEFT JOIN blocks ON blocks.height = module_histories.block_height").
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

func (r *ModuleRepository) GetModuleProposals(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleProposalModel, int64, error) {
	var proposals []dto.ModuleProposalModel
	var total int64

	moduleId := fmt.Sprintf("%s::%s", vmAddress, name)
	err := r.db.Model(&db.ModuleProposal{}).
		Select(
			"proposals.id",
			"proposals.title",
			"proposals.status",
			"proposals.voting_end_time",
			"proposals.deposit_end_time",
			"proposals.types",
			"proposals.is_expedited",
			"proposals.is_emergency",
			"proposals.resolved_height",
			"proposals.proposer_id AS proposer",
		).
		Joins("LEFT JOIN proposals ON proposals.id = module_proposals.proposal_id").
		Where("module_proposals.module_id = ?", moduleId).
		Order("module_proposals.proposal_id DESC").
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&proposals).Error
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query module proposal")
		return nil, 0, err
	}

	if pagination.CountTotal {
		var err error
		total, err = utils.CountWithTimeout(r.db.Model(&db.ModuleProposal{}).Where("module_proposals.module_id = ?", moduleId), r.countQueryTimeout)
		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count module proposal")
			return nil, 0, err
		}
	}

	return proposals, total, nil
}

func (r *ModuleRepository) GetModuleTransactions(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleTxResponse, int64, error) {
	var txs []dto.ModuleTxResponse
	var total int64

	moduleId := fmt.Sprintf("%s::%s", vmAddress, name)

	err := r.db.Model(&db.ModuleTransaction{}).
		Select(
			"blocks.height",
			"blocks.timestamp",
			"transactions.sender",
			"transactions.hash",
			"transactions.success",
			"transactions.messages",
			"transactions.is_send",
			"transactions.is_ibc",
			"transactions.is_move_execute",
			"transactions.is_move_execute_event",
			"transactions.is_move_publish",
			"transactions.is_move_script",
			"transactions.is_move_upgrade",
			"transactions.is_opinit",
		).
		Joins("LEFT JOIN blocks ON blocks.height = module_transactions.block_height").
		Joins("LEFT JOIN transactions ON transactions.id = module_transactions.tx_id").
		Where("module_transactions.module_id = ?", moduleId).
		Order("module_transactions.block_height DESC").
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&txs).Error
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query module txs")
		return nil, 0, err
	}

	if pagination.CountTotal {
		var err error
		total, err = utils.CountWithTimeout(r.db.Model(&db.ModuleTransaction{}).Where("module_transactions.module_id = ?", moduleId), r.countQueryTimeout)
		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count module txs")
			return nil, 0, err
		}
	}

	return txs, total, nil
}

func (r *ModuleRepository) GetModuleStats(vmAddress string, name string) (*dto.ModuleStatsResponse, error) {
	var stats dto.ModuleStatsResponse
	moduleId := fmt.Sprintf("%s::%s", vmAddress, name)

	err := r.db.Raw(`
		SELECT
			(SELECT COUNT(*) FROM module_transactions WHERE module_id = ?) AS total_txs,
			(SELECT COUNT(*) FROM module_histories WHERE module_id = ?) AS total_histories,
			(SELECT COUNT(*) FROM module_proposals WHERE module_id = ?) AS total_proposals
	`, moduleId, moduleId, moduleId).Scan(&stats).Error
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to get module stats")
		return nil, err
	}

	return &stats, nil
}
