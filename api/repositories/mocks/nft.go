package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/pkg/db"
)

// MockNftRepository is a mock implementation of NftRepositoryI
type MockNftRepository struct {
	mock.Mock
}

// Ensure MockNftRepository implements NftRepositoryI interface
var _ repositories.NftRepositoryI = (*MockNftRepository)(nil)

// NewMockNftRepository creates a new mock NFT repository
func NewMockNftRepository() *MockNftRepository {
	return &MockNftRepository{}
}

// GetCollections mocks the GetCollections method
func (m *MockNftRepository) GetCollections(ctx context.Context, pagination dto.PaginationQuery, search string) ([]db.Collection, int64, error) {
	args := m.Called(ctx, pagination, search)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]db.Collection), args.Get(1).(int64), args.Error(2)
}

// GetCollectionsByAccountAddress mocks the GetCollectionsByAccountAddress method
func (m *MockNftRepository) GetCollectionsByAccountAddress(ctx context.Context, accountAddress string) ([]dto.CollectionByAccountAddressModel, error) {
	args := m.Called(ctx, accountAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.CollectionByAccountAddressModel), args.Error(1)
}

// GetCollectionsByCollectionAddress mocks the GetCollectionsByCollectionAddress method
func (m *MockNftRepository) GetCollectionsByCollectionAddress(ctx context.Context, collectionAddress string) (*db.Collection, error) {
	args := m.Called(ctx, collectionAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Collection), args.Error(1)
}

// GetCollectionActivities mocks the GetCollectionActivities method
func (m *MockNftRepository) GetCollectionActivities(ctx context.Context, pagination dto.PaginationQuery, collectionAddress string, search string) ([]dto.CollectionActivityModel, int64, error) {
	args := m.Called(ctx, pagination, collectionAddress, search)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.CollectionActivityModel), args.Get(1).(int64), args.Error(2)
}

// GetCollectionCreator mocks the GetCollectionCreator method
func (m *MockNftRepository) GetCollectionCreator(ctx context.Context, collectionAddress string) (*dto.CollectionCreatorModel, error) {
	args := m.Called(ctx, collectionAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.CollectionCreatorModel), args.Error(1)
}

// GetCollectionMutateEvents mocks the GetCollectionMutateEvents method
func (m *MockNftRepository) GetCollectionMutateEvents(ctx context.Context, pagination dto.PaginationQuery, collectionAddress string) ([]dto.MutateEventModel, int64, error) {
	args := m.Called(ctx, pagination, collectionAddress)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.MutateEventModel), args.Get(1).(int64), args.Error(2)
}

// GetNftByNftAddress mocks the GetNftByNftAddress method
func (m *MockNftRepository) GetNftByNftAddress(ctx context.Context, collectionAddress string, nftAddress string) (*dto.NftByAddressModel, error) {
	args := m.Called(ctx, collectionAddress, nftAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.NftByAddressModel), args.Error(1)
}

// GetNftsByAccountAddress mocks the GetNftsByAccountAddress method
func (m *MockNftRepository) GetNftsByAccountAddress(ctx context.Context, pagination dto.PaginationQuery, accountAddress string, collectionAddress string, search string) ([]dto.NftByAddressModel, int64, error) {
	args := m.Called(ctx, pagination, accountAddress, collectionAddress, search)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.NftByAddressModel), args.Get(1).(int64), args.Error(2)
}

// GetNftsByCollectionAddress mocks the GetNftsByCollectionAddress method
func (m *MockNftRepository) GetNftsByCollectionAddress(ctx context.Context, pagination dto.PaginationQuery, collectionAddress string, search string) ([]dto.NftByAddressModel, int64, error) {
	args := m.Called(ctx, pagination, collectionAddress, search)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.NftByAddressModel), args.Get(1).(int64), args.Error(2)
}

// GetNftMintInfo mocks the GetNftMintInfo method
func (m *MockNftRepository) GetNftMintInfo(ctx context.Context, nftAddress string) (*dto.NftMintInfoModel, error) {
	args := m.Called(ctx, nftAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.NftMintInfoModel), args.Error(1)
}

// GetNftMutateEvents mocks the GetNftMutateEvents method
func (m *MockNftRepository) GetNftMutateEvents(ctx context.Context, pagination dto.PaginationQuery, nftAddress string) ([]dto.MutateEventModel, int64, error) {
	args := m.Called(ctx, pagination, nftAddress)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.MutateEventModel), args.Get(1).(int64), args.Error(2)
}

// GetNftTxs mocks the GetNftTxs method
func (m *MockNftRepository) GetNftTxs(ctx context.Context, pagination dto.PaginationQuery, nftAddress string) ([]dto.NftTxModel, int64, error) {
	args := m.Called(ctx, pagination, nftAddress)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.NftTxModel), args.Get(1).(int64), args.Error(2)
}
