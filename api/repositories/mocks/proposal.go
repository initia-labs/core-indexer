package mocks

import (
	"context"

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
func (m *MockProposalRepository) GetProposals(ctx context.Context, pagination *dto.PaginationQuery) ([]db.Proposal, error) {
	args := m.Called(ctx, pagination)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.Proposal), args.Error(1)
}

// GetProposalVotesByValidator mocks the GetProposalVotesByValidator method
func (m *MockProposalRepository) GetProposalVotesByValidator(ctx context.Context, operatorAddr string) ([]db.ProposalVote, error) {
	args := m.Called(ctx, operatorAddr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.ProposalVote), args.Error(1)
}

// SearchProposals mocks the SearchProposals method
func (m *MockProposalRepository) SearchProposals(ctx context.Context, pagination dto.PaginationQuery, proposer, search string, statuses, types []string) ([]dto.ProposalSummary, int64, error) {
	args := m.Called(ctx, pagination, proposer, search, statuses, types)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.ProposalSummary), args.Get(1).(int64), args.Error(2)
}

// GetAllProposalTypes mocks the GetAllProposalTypes method
func (m *MockProposalRepository) GetAllProposalTypes(ctx context.Context) (*dto.ProposalsTypesResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ProposalsTypesResponse), args.Error(1)
}

// GetProposalInfo mocks the GetProposalInfo method
func (m *MockProposalRepository) GetProposalInfo(ctx context.Context, id int) (*dto.ProposalInfo, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ProposalInfo), args.Error(1)
}

// GetProposalVotes mocks the GetProposalVotes method
func (m *MockProposalRepository) GetProposalVotes(ctx context.Context, id int, limit, offset int64, search, answer string) ([]dto.ProposalVote, int64, error) {
	args := m.Called(ctx, id, limit, offset, search, answer)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.ProposalVote), args.Get(1).(int64), args.Error(2)
}

// GetProposalValidatorVotes mocks the GetProposalValidatorVotes method
func (m *MockProposalRepository) GetProposalValidatorVotes(ctx context.Context, id int) ([]dto.ProposalVote, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.ProposalVote), args.Error(1)
}

// GetProposalAnswerCounts mocks the GetProposalAnswerCounts method
func (m *MockProposalRepository) GetProposalAnswerCounts(ctx context.Context, id int) (*dto.ProposalAnswerCountsResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ProposalAnswerCountsResponse), args.Error(1)
}
