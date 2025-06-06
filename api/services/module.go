package services

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
)

// ModuleService defines the interface for module-related operations
type ModuleService interface {
	GetModules(pagination dto.PaginationQuery) (*dto.ModulesResponse, error)
	GetModuleById(vmAddress string, name string) (*dto.ModuleResponse, error)
	GetModuleHistories(pagination dto.PaginationQuery, vmAddress string, name string) (*dto.ModuleHistoriesResponse, error)
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

	response := &dto.ModuleHistoriesResponse{
		ModuleHistories: moduleHistories,
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   total,
		},
	}

	return response, nil
}
