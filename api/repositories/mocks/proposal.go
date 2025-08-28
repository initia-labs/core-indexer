package mocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/pkg/db"
)

// MockProposalRepository is a mock implementation of ProposalRepositoryI
type MockProposalRepository struct {
	mock.Mock
}

// Ensure MockProposalRepository implements ProposalRepositoryI interface
var _ repositories.ProposalRepositoryI = (*MockProposalRepository)(nil)

// NewMockProposalRepository creates a new mock proposal repository
func NewMockProposalRepository() *MockProposalRepository {
	return &MockProposalRepository{}
}

// GetProposals mocks the GetProposals method
func (m *MockProposalRepository) GetProposals(pagination *dto.PaginationQuery) ([]db.Proposal, error) {
	args := m.Called(pagination)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.Proposal), args.Error(1)
}

// GetProposalVotesByValidator mocks the GetProposalVotesByValidator method
func (m *MockProposalRepository) GetProposalVotesByValidator(operatorAddr string) ([]db.ProposalVote, error) {
	args := m.Called(operatorAddr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.ProposalVote), args.Error(1)
}

// SearchProposals mocks the SearchProposals method
func (m *MockProposalRepository) SearchProposals(pagination dto.PaginationQuery, proposer, search string, statuses, types []string) ([]dto.ProposalSummary, int64, error) {
	args := m.Called(pagination, proposer, search, statuses, types)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.ProposalSummary), args.Get(1).(int64), args.Error(2)
}

// GetAllProposalTypes mocks the GetAllProposalTypes method
func (m *MockProposalRepository) GetAllProposalTypes() (*dto.ProposalsTypesResponse, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ProposalsTypesResponse), args.Error(1)
}

// GetProposalInfo mocks the GetProposalInfo method
func (m *MockProposalRepository) GetProposalInfo(id int) (*dto.ProposalInfo, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ProposalInfo), args.Error(1)
}

// GetProposalVotes mocks the GetProposalVotes method
func (m *MockProposalRepository) GetProposalVotes(id int, limit, offset int64, search, answer string) ([]dto.ProposalVote, int64, error) {
	args := m.Called(id, limit, offset, search, answer)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.ProposalVote), args.Get(1).(int64), args.Error(2)
}

// GetProposalValidatorVotes mocks the GetProposalValidatorVotes method
func (m *MockProposalRepository) GetProposalValidatorVotes(id int) ([]dto.ProposalVote, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.ProposalVote), args.Error(1)
}

// GetProposalAnswerCounts mocks the GetProposalAnswerCounts method
func (m *MockProposalRepository) GetProposalAnswerCounts(id int) (*dto.ProposalAnswerCountsResponse, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ProposalAnswerCountsResponse), args.Error(1)
}
