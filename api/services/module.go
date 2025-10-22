package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"gorm.io/gorm"
)

type ModuleService interface {
	GetModules(ctx context.Context, pagination dto.PaginationQuery) (*dto.ModulesResponse, error)
	GetModuleById(ctx context.Context, vmAddress string, name string) (*dto.ModuleResponse, error)
	GetModuleHistories(ctx context.Context, pagination dto.PaginationQuery, vmAddress string, name string) (*dto.ModuleHistoriesResponse, error)
	GetModulePublishInfo(ctx context.Context, vmAddress string, name string) (*dto.ModulePublishInfoResponse, error)
	GetModuleProposals(ctx context.Context, pagination dto.PaginationQuery, vmAddress string, name string) (*dto.ModuleProposalsResponse, error)
	GetModuleTransactions(ctx context.Context, pagination dto.PaginationQuery, vmAddress string, name string) (*dto.ModuleTxsResponse, error)
	GetModuleStats(ctx context.Context, vmAddress string, name string) (*dto.ModuleStatsResponse, error)
}

type moduleService struct {
	repo repositories.ModuleRepositoryI
}

func NewModuleService(repo repositories.ModuleRepositoryI) ModuleService {
	return &moduleService{
		repo: repo,
	}
}

// GetModules retrieves modules with pagination
func (s *moduleService) GetModules(ctx context.Context, pagination dto.PaginationQuery) (*dto.ModulesResponse, error) {
	modules, total, err := s.repo.GetModules(ctx, pagination)
	if err != nil {
		return nil, err
	}

	response := &dto.ModulesResponse{
		Modules:    modules,
		Pagination: dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}

	return response, nil
}

// GetModuleById retrieves a module by id
func (s *moduleService) GetModuleById(ctx context.Context, vmAddress string, name string) (*dto.ModuleResponse, error) {
	module, err := s.repo.GetModuleById(ctx, vmAddress, name)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// GetModuleHistories retrieves module histories with pagination
func (s *moduleService) GetModuleHistories(ctx context.Context, pagination dto.PaginationQuery, vmAddress string, name string) (*dto.ModuleHistoriesResponse, error) {
	moduleHistories, total, err := s.repo.GetModuleHistories(ctx, pagination, vmAddress, name)
	if err != nil {
		return nil, err
	}

	for i := range moduleHistories {
		moduleHistories[i].UpgradePolicy = strings.ToUpper(moduleHistories[i].UpgradePolicy)
		if i < len(moduleHistories)-1 {
			previousPolicy := strings.ToUpper(moduleHistories[i+1].UpgradePolicy)
			moduleHistories[i].PreviousPolicy = &previousPolicy
		}
	}

	response := &dto.ModuleHistoriesResponse{
		ModuleHistories: moduleHistories,
		Pagination:      dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}

	return response, nil
}

// GetModulePublishInfo retrieves a module publish info
func (s *moduleService) GetModulePublishInfo(ctx context.Context, vmAddress string, name string) (*dto.ModulePublishInfoResponse, error) {
	modulePublishInfoResponse := &dto.ModulePublishInfoResponse{}
	modulePublishInfo, err := s.repo.GetModulePublishInfo(ctx, vmAddress, name)
	if err != nil {
		return nil, err
	}

	if len(modulePublishInfo) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	recentPublish := modulePublishInfo[0]

	if recentPublish.TransactionHash != nil {
		txHash := fmt.Sprintf("%x", *recentPublish.TransactionHash)
		modulePublishInfoResponse.RecentPublishTransaction = &txHash
	}

	modulePublishInfoResponse.IsRepublished = len(modulePublishInfo) > 1
	modulePublishInfoResponse.RecentPublishBlockHeight = recentPublish.Height
	modulePublishInfoResponse.RecentPublishBlockTimestamp = recentPublish.Timestamp
	modulePublishInfoResponse.RecentPublishProposal = recentPublish.Proposal

	return modulePublishInfoResponse, nil
}

// GetModuleProposals retrieves a module proposal
func (s *moduleService) GetModuleProposals(ctx context.Context, pagination dto.PaginationQuery, vmAddress string, name string) (*dto.ModuleProposalsResponse, error) {
	proposals, total, err := s.repo.GetModuleProposals(ctx, pagination, vmAddress, name)
	if err != nil {
		return nil, err
	}

	return &dto.ModuleProposalsResponse{
		Proposals:  proposals,
		Pagination: dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}, nil
}

// GetModuleTransactions retrieves a module transaction
func (s *moduleService) GetModuleTransactions(ctx context.Context, pagination dto.PaginationQuery, vmAddress string, name string) (*dto.ModuleTxsResponse, error) {
	txs, total, err := s.repo.GetModuleTransactions(ctx, pagination, vmAddress, name)
	if err != nil {
		return nil, err
	}

	moduleTxs := make([]dto.ModuleTxResponse, len(txs))
	for i, tx := range txs {
		moduleTxs[i] = dto.ModuleTxResponse{
			Height:             tx.Height,
			Timestamp:          tx.Timestamp,
			Sender:             tx.Sender,
			TxHash:             fmt.Sprintf("%x", tx.TxHash),
			Success:            tx.Success,
			Messages:           tx.Messages,
			IsSend:             tx.IsSend,
			IsIBC:              tx.IsIBC,
			IsMoveExecute:      tx.IsMoveExecute,
			IsMoveExecuteEvent: tx.IsMoveExecuteEvent,
			IsMovePublish:      tx.IsMovePublish,
			IsMoveScript:       tx.IsMoveScript,
			IsMoveUpgrade:      tx.IsMoveUpgrade,
			IsOpinit:           tx.IsOpinit,
		}
	}

	return &dto.ModuleTxsResponse{
		ModuleTxs:  moduleTxs,
		Pagination: dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}, nil
}

// GetModuleStats retrieves a module stats by module id
func (s *moduleService) GetModuleStats(ctx context.Context, vmAddress string, name string) (*dto.ModuleStatsResponse, error) {
	stats, err := s.repo.GetModuleStats(ctx, vmAddress, name)
	if err != nil {
		return nil, err
	}

	return stats, nil
}
