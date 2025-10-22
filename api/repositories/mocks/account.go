package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/pkg/db"
)

// MockAccountRepository is a mock implementation of AccountRepositoryI
type MockAccountRepository struct {
	mock.Mock
}

// Ensure MockAccountRepository implements AccountRepositoryI interface
var _ repositories.AccountRepositoryI = (*MockAccountRepository)(nil)

// NewMockAccountRepository creates a new mock account repository
func NewMockAccountRepository() *MockAccountRepository {
	return &MockAccountRepository{}
}

// GetAccountByAccountAddress mocks the GetAccountByAccountAddress method
func (m *MockAccountRepository) GetAccountByAccountAddress(ctx context.Context, accountAddress string) (*db.Account, error) {
	args := m.Called(ctx, accountAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Account), args.Error(1)
}

// GetAccountProposals mocks the GetAccountProposals method
func (m *MockAccountRepository) GetAccountProposals(ctx context.Context, pagination dto.PaginationQuery, accountAddress string) ([]db.Proposal, int64, error) {
	args := m.Called(ctx, pagination, accountAddress)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]db.Proposal), args.Get(1).(int64), args.Error(2)
}

// GetAccountTxs mocks the GetAccountTxs method
func (m *MockAccountRepository) GetAccountTxs(
	ctx context.Context,
	pagination dto.PaginationQuery,
	accountAddress string,
	search string,
	isSend bool,
	isIbc bool,
	isOpinit bool,
	isMovePublish bool,
	isMoveUpgrade bool,
	isMoveExecute bool,
	isMoveScript bool,
	isSigner *bool,
) ([]dto.AccountTxModel, int64, error) {
	args := m.Called(ctx, pagination, accountAddress, search, isSend, isIbc, isOpinit, isMovePublish, isMoveUpgrade, isMoveExecute, isMoveScript, isSigner)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.AccountTxModel), args.Get(1).(int64), args.Error(2)
}
