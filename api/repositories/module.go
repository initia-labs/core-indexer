package repositories

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/db/query"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

var _ ModuleRepositoryI = (*ModuleRepository)(nil)

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
	listQ := query.ModuleListQuery(r.db, pagination.Limit, pagination.Offset)
	countQ := query.ModuleCountQuery(r.db)
	total, err := db.ListWithCount(listQ, countQ, &modules, pagination.CountTotal, r.countQueryTimeout)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query modules")
		return nil, 0, err
	}
	return modules, total, nil
}

// GetModuleById returns a single module by vmAddress and name
func (r *ModuleRepository) GetModuleById(vmAddress string, name string) (*dto.ModuleResponse, error) {
	var module dto.ModuleResponse
	err := query.ModuleByIDQuery(r.db, vmAddress, name).First(&module).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("module %s::%s not found", vmAddress, name)
		}
		logger.Get().Error().Err(err).Msg("Failed to query module")
		return nil, err
	}
	return &module, nil
}

// GetModuleHistories returns module histories with pagination
func (r *ModuleRepository) GetModuleHistories(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleHistoryResponse, int64, error) {
	var histories []dto.ModuleHistoryResponse
	moduleID := fmt.Sprintf("%s::%s", vmAddress, name)
	listQ := query.ModuleHistoriesListQuery(r.db, moduleID, pagination.Limit, pagination.Offset)
	countQ := query.ModuleHistoriesCountQuery(r.db, moduleID)
	total, err := db.ListWithCount(listQ, countQ, &histories, pagination.CountTotal, r.countQueryTimeout)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query module histories")
		return nil, 0, err
	}
	return histories, total, nil
}

// GetModulePublishInfo returns publish info for a module (up to 2 rows)
func (r *ModuleRepository) GetModulePublishInfo(vmAddress string, name string) ([]dto.ModulePublishInfoModel, error) {
	var modulePublishInfos []dto.ModulePublishInfoModel
	moduleID := fmt.Sprintf("%s::%s", vmAddress, name)
	if err := query.ModulePublishInfoQuery(r.db, moduleID).Find(&modulePublishInfos).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query module publish info")
		return nil, err
	}
	return modulePublishInfos, nil
}

// GetModuleProposals returns module proposals with pagination
func (r *ModuleRepository) GetModuleProposals(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleProposalModel, int64, error) {
	var proposals []dto.ModuleProposalModel
	moduleID := fmt.Sprintf("%s::%s", vmAddress, name)
	listQ := query.ModuleProposalsListQuery(r.db, moduleID, pagination.Limit, pagination.Offset)
	countQ := query.ModuleProposalsCountQuery(r.db, moduleID)
	total, err := db.ListWithCount(listQ, countQ, &proposals, pagination.CountTotal, r.countQueryTimeout)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query module proposal")
		return nil, 0, err
	}
	return proposals, total, nil
}

// GetModuleTransactions returns module transactions with pagination
func (r *ModuleRepository) GetModuleTransactions(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleTxResponse, int64, error) {
	var txs []dto.ModuleTxResponse
	moduleID := fmt.Sprintf("%s::%s", vmAddress, name)
	listQ := query.ModuleTransactionsListQuery(r.db, moduleID, pagination.Limit, pagination.Offset)
	countQ := query.ModuleTransactionsCountQuery(r.db, moduleID)
	total, err := db.ListWithCount(listQ, countQ, &txs, pagination.CountTotal, r.countQueryTimeout)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query module txs")
		return nil, 0, err
	}
	return txs, total, nil
}

// GetModuleStats returns total_txs, total_histories, total_proposals for a module
func (r *ModuleRepository) GetModuleStats(vmAddress string, name string) (*dto.ModuleStatsResponse, error) {
	var stats dto.ModuleStatsResponse
	moduleID := fmt.Sprintf("%s::%s", vmAddress, name)
	if err := query.ModuleStatsQuery(r.db, moduleID).Scan(&stats).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to get module stats")
		return nil, err
	}
	return &stats, nil
}
