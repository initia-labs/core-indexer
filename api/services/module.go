package services

import (
	"strings"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
)

// ModuleService defines the interface for module-related operations
type ModuleService interface {
	GetModules(pagination dto.PaginationQuery) (*dto.ModulesResponse, error)
	GetModuleById(vmAddress string, name string) (*dto.ModuleResponse, error)
	GetModuleHistories(pagination dto.PaginationQuery, vmAddress string, name string) (*dto.ModuleHistoriesResponse, error)
	GetModulePublishInfo(vmAddress string, name string) (*dto.ModulePublishInfoResponse, error)
}

type moduleService struct {
	repo repositories.ModuleRepository
}

func NewModuleService(repo repositories.ModuleRepository) ModuleService {
	return &moduleService{
		repo: repo,
	}
}

// GetModules retrieves modules with pagination
func (s *moduleService) GetModules(pagination dto.PaginationQuery) (*dto.ModulesResponse, error) {
	modules, total, err := s.repo.GetModules(pagination)
	if err != nil {
		return nil, err
	}

	response := &dto.ModulesResponse{
		Modules: modules,
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   total,
		},
	}

	return response, nil
}

// GetModuleById retrieves a module by id
func (s *moduleService) GetModuleById(vmAddress string, name string) (*dto.ModuleResponse, error) {
	module, err := s.repo.GetModuleById(vmAddress, name)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// GetModuleHistories retrieves module histories with pagination
func (s *moduleService) GetModuleHistories(pagination dto.PaginationQuery, vmAddress string, name string) (*dto.ModuleHistoriesResponse, error) {
	moduleHistories, total, err := s.repo.GetModuleHistories(pagination, vmAddress, name)
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
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   total,
		},
	}

	return response, nil
}

// GetModulePublishInfo retrieves a module publish info
func (s *moduleService) GetModulePublishInfo(vmAddress string, name string) (*dto.ModulePublishInfoResponse, error) {
	modulePublishInfoResponse := &dto.ModulePublishInfoResponse{}
	modulePublishInfo, err := s.repo.GetModulePublishInfo(vmAddress, name)
	if err != nil {
		return nil, err
	}

	recentPublish := modulePublishInfo[0]

	modulePublishInfoResponse.RecentPublishTransaction = recentPublish.TransactionHash
	modulePublishInfoResponse.IsRepublished = len(modulePublishInfo) > 1
	modulePublishInfoResponse.RecentPublishBlockHeight = recentPublish.Height
	modulePublishInfoResponse.RecentPublishBlockTimestamp = recentPublish.Timestamp
	modulePublishInfoResponse.RecentPublishProposal = &recentPublish.ProposalTitle

	return modulePublishInfoResponse, nil
}
