package mocks

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/stretchr/testify/mock"
)

// MockModuleRepository is a mock implementation of ModuleRepositoryI
type MockModuleRepository struct {
	mock.Mock
}

// Ensure MockModuleRepository implements ModuleRepositoryI interface
var _ repositories.ModuleRepositoryI = (*MockModuleRepository)(nil)

// NewMockModuleRepository creates a new mock module repository
func NewMockModuleRepository() *MockModuleRepository {
	return &MockModuleRepository{}
}

// GetModules mocks the GetModules method
func (m *MockModuleRepository) GetModules(pagination dto.PaginationQuery) ([]dto.ModuleResponse, int64, error) {
	args := m.Called(pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.ModuleResponse), args.Get(1).(int64), args.Error(2)
}

// GetModuleById mocks the GetModuleById method
func (m *MockModuleRepository) GetModuleById(vmAddress string, name string) (*dto.ModuleResponse, error) {
	args := m.Called(vmAddress, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ModuleResponse), args.Error(1)
}

// GetModuleHistories mocks the GetModuleHistories method
func (m *MockModuleRepository) GetModuleHistories(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleHistoryResponse, int64, error) {
	args := m.Called(pagination, vmAddress, name)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.ModuleHistoryResponse), args.Get(1).(int64), args.Error(2)
}

// GetModulePublishInfo mocks the GetModulePublishInfo method
func (m *MockModuleRepository) GetModulePublishInfo(vmAddress string, name string) ([]dto.ModulePublishInfoModel, error) {
	args := m.Called(vmAddress, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.ModulePublishInfoModel), args.Error(1)
}

// GetModuleProposals mocks the GetModuleProposals method
func (m *MockModuleRepository) GetModuleProposals(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleProposalModel, int64, error) {
	args := m.Called(pagination, vmAddress, name)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.ModuleProposalModel), args.Get(1).(int64), args.Error(2)
}

// GetModuleTransactions mocks the GetModuleTransactions method
func (m *MockModuleRepository) GetModuleTransactions(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleTxResponse, int64, error) {
	args := m.Called(pagination, vmAddress, name)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.ModuleTxResponse), args.Get(1).(int64), args.Error(2)
}

// GetModuleStats mocks the GetModuleStats method
func (m *MockModuleRepository) GetModuleStats(vmAddress string, name string) (*dto.ModuleStatsResponse, error) {
	args := m.Called(vmAddress, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ModuleStatsResponse), args.Error(1)
}
