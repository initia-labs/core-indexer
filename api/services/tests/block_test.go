package services_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories/mocks"
	"github.com/initia-labs/core-indexer/api/services"
)

func TestBlockService_GetBlockHeightLatest(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockBlockRepository()

	// Test data
	expectedHeight := int64(1000)

	// Set up mock expectations
	mockRepo.On("GetBlockHeightLatest", context.Background()).Return(&expectedHeight, nil)

	// Create service with mock repository
	service := services.NewBlockService(mockRepo)

	// Call the method
	result, err := service.GetBlockHeightLatest()

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedHeight, result.Height)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestBlockService_GetBlockHeightLatest_Error(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockBlockRepository()

	// Set up mock expectations for error case
	mockRepo.On("GetBlockHeightLatest").Return(nil, assert.AnError)

	// Create service with mock repository
	service := services.NewBlockService(mockRepo)

	// Call the method
	result, err := service.GetBlockHeightLatest()

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, assert.AnError, err)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestBlockService_GetBlockHeightInformativeLatest(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockBlockRepository()

	// Test data
	expectedHeight := int64(950)

	// Set up mock expectations
	mockRepo.On("GetBlockHeightInformativeLatest").Return(&expectedHeight, nil)

	// Create service with mock repository
	service := services.NewBlockService(mockRepo)

	// Call the method
	result, err := service.GetBlockHeightInformativeLatest()

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedHeight, result.Height)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestBlockService_GetBlockTimeAverage(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockBlockRepository()

	// Test data
	latestHeight := int64(1000)
	timestamps := []time.Time{
		time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 9, 59, 30, 0, time.UTC),
		time.Date(2024, 1, 1, 9, 59, 0, 0, time.UTC),
	}

	// Set up mock expectations
	mockRepo.On("GetBlockHeightLatest").Return(&latestHeight, nil)
	mockRepo.On("GetBlockTimestamp", latestHeight).Return(timestamps, nil)

	// Create service with mock repository
	service := services.NewBlockService(mockRepo)

	// Call the method
	result, err := service.GetBlockTimeAverage()

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 30.0, result.AverageBlockTime)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestBlockService_GetBlocks(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockBlockRepository()

	// Test data
	pagination := dto.PaginationQuery{
		Limit:      10,
		Offset:     0,
		CountTotal: true,
	}

	expectedBlocks := []dto.BlockModel{
		{
			Hash:            "block_hash_123",
			Height:          1000,
			Timestamp:       "2024-01-01T10:00:00Z",
			OperatorAddress: "initvaloper1test",
			Moniker:         "Test Validator",
			Identity:        "test-identity",
			TxCount:         5,
		},
	}

	// Set up mock expectations
	mockRepo.On("GetBlocks", pagination).Return(expectedBlocks, int64(1), nil)

	// Create service with mock repository
	service := services.NewBlockService(mockRepo)

	// Call the method
	result, err := service.GetBlocks(pagination)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Blocks, 1)
	assert.Equal(t, fmt.Sprintf("%d", int64(1)), result.Pagination.Total)
	assert.Equal(t, "626c6f636b5f686173685f313233", result.Blocks[0].Hash) // hex encoded

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestBlockService_GetBlockInfo(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockBlockRepository()

	// Test data
	height := int64(1000)
	expectedBlockInfo := &dto.BlockInfoModel{
		Hash:            "block_hash_123",
		Height:          height,
		Timestamp:       "2024-01-01T10:00:00Z",
		OperatorAddress: "initvaloper1test",
		Moniker:         "Test Validator",
		Identity:        "test-identity",
		GasUsed:         1000000,
		GasLimit:        2000000,
	}

	// Set up mock expectations
	mockRepo.On("GetBlockInfo", height).Return(expectedBlockInfo, nil)

	// Create service with mock repository
	service := services.NewBlockService(mockRepo)

	// Call the method
	result, err := service.GetBlockInfo(height)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "626c6f636b5f686173685f313233", result.Hash) // hex encoded
	assert.Equal(t, height, result.Height)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestBlockService_GetBlockTxs(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockBlockRepository()

	// Test data
	pagination := dto.PaginationQuery{
		Limit:      10,
		Offset:     0,
		CountTotal: true,
	}
	height := int64(1000)

	expectedTxs := []dto.BlockTxModel{
		{
			Height:    height,
			Timestamp: "2024-01-01T10:00:00Z",
			Address:   "init1test1",
			Hash:      "tx_hash_1",
			Success:   true,
			Messages:  []byte(`[{"type": "send"}]`),
			IsSend:    true,
			IsIbc:     false,
			IsOpinit:  false,
		},
		{
			Height:    height,
			Timestamp: "2024-01-01T10:00:05Z",
			Address:   "init1test2",
			Hash:      "tx_hash_2",
			Success:   true,
			Messages:  []byte(`[{"type": "ibc_transfer"}]`),
			IsSend:    false,
			IsIbc:     true,
			IsOpinit:  false,
		},
		{
			Height:    height,
			Timestamp: "2024-01-01T10:00:10Z",
			Address:   "init1test3",
			Hash:      "tx_hash_3",
			Success:   false,
			Messages:  []byte(`[{"type": "opinit"}]`),
			IsSend:    false,
			IsIbc:     false,
			IsOpinit:  true,
		},
		{
			Height:    height,
			Timestamp: "2024-01-01T10:00:15Z",
			Address:   "init1test4",
			Hash:      "tx_hash_4",
			Success:   true,
			Messages:  []byte(`[{"type": "move"}]`),
			IsSend:    false,
			IsIbc:     false,
			IsOpinit:  false,
		},
	}

	// Set up mock expectations
	mockRepo.On("GetBlockTxs", pagination, height).Return(expectedTxs, int64(4), nil)

	// Create service with mock repository
	service := services.NewBlockService(mockRepo)

	// Call the method
	result, err := service.GetBlockTxs(pagination, height)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.BlockTxs, 4)
	assert.Equal(t, fmt.Sprintf("%d", int64(4)), result.Pagination.Total)

	// Check first transaction (send type)
	assert.Equal(t, "74785f686173685f31", result.BlockTxs[0].Hash) // hex encoded
	assert.Equal(t, "init1test1", result.BlockTxs[0].Address)
	assert.True(t, result.BlockTxs[0].Success)
	assert.True(t, result.BlockTxs[0].IsSend)
	assert.False(t, result.BlockTxs[0].IsIbc)
	assert.False(t, result.BlockTxs[0].IsOpinit)

	// Check second transaction (IBC type)
	assert.Equal(t, "74785f686173685f32", result.BlockTxs[1].Hash) // hex encoded
	assert.Equal(t, "init1test2", result.BlockTxs[1].Address)
	assert.True(t, result.BlockTxs[1].Success)
	assert.False(t, result.BlockTxs[1].IsSend)
	assert.True(t, result.BlockTxs[1].IsIbc)
	assert.False(t, result.BlockTxs[1].IsOpinit)

	// Check third transaction (Opinit type, failed)
	assert.Equal(t, "74785f686173685f33", result.BlockTxs[2].Hash) // hex encoded
	assert.Equal(t, "init1test3", result.BlockTxs[2].Address)
	assert.False(t, result.BlockTxs[2].Success)
	assert.False(t, result.BlockTxs[2].IsSend)
	assert.False(t, result.BlockTxs[2].IsIbc)
	assert.True(t, result.BlockTxs[2].IsOpinit)

	// Check fourth transaction (Move type)
	assert.Equal(t, "74785f686173685f34", result.BlockTxs[3].Hash) // hex encoded
	assert.Equal(t, "init1test4", result.BlockTxs[3].Address)
	assert.True(t, result.BlockTxs[3].Success)
	assert.False(t, result.BlockTxs[3].IsSend)
	assert.False(t, result.BlockTxs[3].IsIbc)
	assert.False(t, result.BlockTxs[3].IsOpinit)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestBlockService_GetBlocks_WithPagination(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockBlockRepository()

	// Test data for different pagination scenarios
	testCases := []struct {
		name           string
		pagination     dto.PaginationQuery
		expectedBlocks []dto.BlockModel
		expectedTotal  int64
		expectedCount  int
	}{
		{
			name: "First page with limit 5",
			pagination: dto.PaginationQuery{
				Limit:      5,
				Offset:     0,
				CountTotal: true,
			},
			expectedBlocks: []dto.BlockModel{
				{
					Hash:            "block_hash_1",
					Height:          1000,
					Timestamp:       "2024-01-01T10:00:00Z",
					OperatorAddress: "initvaloper1test1",
					Moniker:         "Test Validator 1",
					Identity:        "test-identity-1",
					TxCount:         5,
				},
				{
					Hash:            "block_hash_2",
					Height:          999,
					Timestamp:       "2024-01-01T09:59:30Z",
					OperatorAddress: "initvaloper1test2",
					Moniker:         "Test Validator 2",
					Identity:        "test-identity-2",
					TxCount:         3,
				},
			},
			expectedTotal: 50,
			expectedCount: 2,
		},
		{
			name: "Second page with offset",
			pagination: dto.PaginationQuery{
				Limit:      3,
				Offset:     5,
				CountTotal: true,
			},
			expectedBlocks: []dto.BlockModel{
				{
					Hash:            "block_hash_6",
					Height:          995,
					Timestamp:       "2024-01-01T09:57:00Z",
					OperatorAddress: "initvaloper1test6",
					Moniker:         "Test Validator 6",
					Identity:        "test-identity-6",
					TxCount:         2,
				},
			},
			expectedTotal: 50,
			expectedCount: 1,
		},
		{
			name: "Empty result with high offset",
			pagination: dto.PaginationQuery{
				Limit:      10,
				Offset:     100,
				CountTotal: true,
			},
			expectedBlocks: []dto.BlockModel{},
			expectedTotal:  50,
			expectedCount:  0,
		},
		{
			name: "Large limit",
			pagination: dto.PaginationQuery{
				Limit:      1000,
				Offset:     0,
				CountTotal: true,
			},
			expectedBlocks: []dto.BlockModel{
				{
					Hash:            "block_hash_1",
					Height:          1000,
					Timestamp:       "2024-01-01T10:00:00Z",
					OperatorAddress: "initvaloper1test1",
					Moniker:         "Test Validator 1",
					Identity:        "test-identity-1",
					TxCount:         5,
				},
			},
			expectedTotal: 50,
			expectedCount: 1,
		},
		{
			name: "Without count total",
			pagination: dto.PaginationQuery{
				Limit:      5,
				Offset:     0,
				CountTotal: false,
			},
			expectedBlocks: []dto.BlockModel{
				{
					Hash:            "block_hash_1",
					Height:          1000,
					Timestamp:       "2024-01-01T10:00:00Z",
					OperatorAddress: "initvaloper1test1",
					Moniker:         "Test Validator 1",
					Identity:        "test-identity-1",
					TxCount:         5,
				},
			},
			expectedTotal: 0, // When CountTotal is false, total should be 0
			expectedCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up mock expectations for this test case
			mockRepo.On("GetBlocks", tc.pagination).Return(tc.expectedBlocks, tc.expectedTotal, nil)

			// Create service with mock repository
			service := services.NewBlockService(mockRepo)

			// Call the method
			result, err := service.GetBlocks(tc.pagination)

			// Assertions
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Len(t, result.Blocks, tc.expectedCount)
			assert.Equal(t, fmt.Sprintf("%d", tc.expectedTotal), result.Pagination.Total)

			// Verify the blocks are properly formatted
			if len(result.Blocks) > 0 {
				assert.Equal(t, fmt.Sprintf("%x", tc.expectedBlocks[0].Hash), result.Blocks[0].Hash)
				assert.Equal(t, tc.expectedBlocks[0].Height, result.Blocks[0].Height)
				assert.Equal(t, tc.expectedBlocks[0].Timestamp, result.Blocks[0].Timestamp)
				assert.Equal(t, tc.expectedBlocks[0].TxCount, result.Blocks[0].TxCount)
				assert.Equal(t, tc.expectedBlocks[0].OperatorAddress, result.Blocks[0].Proposer.OperatorAddress)
				assert.Equal(t, tc.expectedBlocks[0].Moniker, result.Blocks[0].Proposer.Moniker)
				assert.Equal(t, tc.expectedBlocks[0].Identity, result.Blocks[0].Proposer.Identity)
			}

			// Verify mock was called as expected
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestBlockService_GetBlockTxs_WithPagination(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockBlockRepository()

	// Test data for different pagination scenarios
	testCases := []struct {
		name          string
		pagination    dto.PaginationQuery
		height        int64
		expectedTxs   []dto.BlockTxModel
		expectedTotal int64
		expectedCount int
	}{
		{
			name: "First page of transactions",
			pagination: dto.PaginationQuery{
				Limit:      3,
				Offset:     0,
				CountTotal: true,
			},
			height: 1000,
			expectedTxs: []dto.BlockTxModel{
				{
					Height:    1000,
					Timestamp: "2024-01-01T10:00:00Z",
					Address:   "init1test1",
					Hash:      "tx_hash_1",
					Success:   true,
					Messages:  []byte(`[{"type": "send"}]`),
					IsSend:    true,
					IsIbc:     false,
					IsOpinit:  false,
				},
				{
					Height:    1000,
					Timestamp: "2024-01-01T10:00:05Z",
					Address:   "init1test2",
					Hash:      "tx_hash_2",
					Success:   true,
					Messages:  []byte(`[{"type": "ibc_transfer"}]`),
					IsSend:    false,
					IsIbc:     true,
					IsOpinit:  false,
				},
				{
					Height:    1000,
					Timestamp: "2024-01-01T10:00:10Z",
					Address:   "init1test3",
					Hash:      "tx_hash_3",
					Success:   false,
					Messages:  []byte(`[{"type": "opinit"}]`),
					IsSend:    false,
					IsIbc:     false,
					IsOpinit:  true,
				},
			},
			expectedTotal: 15,
			expectedCount: 3,
		},
		{
			name: "Second page of transactions",
			pagination: dto.PaginationQuery{
				Limit:      2,
				Offset:     3,
				CountTotal: true,
			},
			height: 1000,
			expectedTxs: []dto.BlockTxModel{
				{
					Height:    1000,
					Timestamp: "2024-01-01T10:00:15Z",
					Address:   "init1test4",
					Hash:      "tx_hash_4",
					Success:   true,
					Messages:  []byte(`[{"type": "move"}]`),
					IsSend:    false,
					IsIbc:     false,
					IsOpinit:  false,
				},
				{
					Height:    1000,
					Timestamp: "2024-01-01T10:00:20Z",
					Address:   "init1test5",
					Hash:      "tx_hash_5",
					Success:   true,
					Messages:  []byte(`[{"type": "send"}]`),
					IsSend:    true,
					IsIbc:     false,
					IsOpinit:  false,
				},
			},
			expectedTotal: 15,
			expectedCount: 2,
		},
		{
			name: "Empty result with high offset",
			pagination: dto.PaginationQuery{
				Limit:      10,
				Offset:     20,
				CountTotal: true,
			},
			height:        1000,
			expectedTxs:   []dto.BlockTxModel{},
			expectedTotal: 15,
			expectedCount: 0,
		},
		{
			name: "Without count total",
			pagination: dto.PaginationQuery{
				Limit:      5,
				Offset:     0,
				CountTotal: false,
			},
			height: 1000,
			expectedTxs: []dto.BlockTxModel{
				{
					Height:    1000,
					Timestamp: "2024-01-01T10:00:00Z",
					Address:   "init1test1",
					Hash:      "tx_hash_1",
					Success:   true,
					Messages:  []byte(`[{"type": "send"}]`),
					IsSend:    true,
					IsIbc:     false,
					IsOpinit:  false,
				},
			},
			expectedTotal: 0, // When CountTotal is false, total should be 0
			expectedCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up mock expectations for this test case
			mockRepo.On("GetBlockTxs", tc.pagination, tc.height).Return(tc.expectedTxs, tc.expectedTotal, nil)

			// Create service with mock repository
			service := services.NewBlockService(mockRepo)

			// Call the method
			result, err := service.GetBlockTxs(tc.pagination, tc.height)

			// Assertions
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Len(t, result.BlockTxs, tc.expectedCount)
			assert.Equal(t, fmt.Sprintf("%d", tc.expectedTotal), result.Pagination.Total)

			// Verify the transactions are properly formatted
			if len(result.BlockTxs) > 0 {
				assert.Equal(t, fmt.Sprintf("%x", tc.expectedTxs[0].Hash), result.BlockTxs[0].Hash)
				assert.Equal(t, tc.expectedTxs[0].Height, result.BlockTxs[0].Height)
				assert.Equal(t, tc.expectedTxs[0].Timestamp, result.BlockTxs[0].Timestamp)
				assert.Equal(t, tc.expectedTxs[0].Address, result.BlockTxs[0].Address)
				assert.Equal(t, tc.expectedTxs[0].Success, result.BlockTxs[0].Success)
				assert.Equal(t, tc.expectedTxs[0].Messages, result.BlockTxs[0].Messages)
				assert.Equal(t, tc.expectedTxs[0].IsSend, result.BlockTxs[0].IsSend)
				assert.Equal(t, tc.expectedTxs[0].IsIbc, result.BlockTxs[0].IsIbc)
				assert.Equal(t, tc.expectedTxs[0].IsOpinit, result.BlockTxs[0].IsOpinit)
			}

			// Verify mock was called as expected
			mockRepo.AssertExpectations(t)
		})
	}
}
