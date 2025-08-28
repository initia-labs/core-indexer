package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories/mocks"
	"github.com/initia-labs/core-indexer/api/services"
	"github.com/initia-labs/core-indexer/pkg/db"
)

const (
	AccountAddress = "init1m8p6rakcfl4z5ruwa0578cqgn8c86mkc6ety2z"
)

func TestAccountService_GetAccountByAccountAddress(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockAccountRepository()

	// Test data
	expectedAccount := &db.Account{
		Address:     AccountAddress,
		Type:        "validator",
		Name:        "Test Validator",
		VMAddressID: "vm_address_123",
	}

	// Set up mock expectations
	mockRepo.On("GetAccountByAccountAddress", AccountAddress).Return(expectedAccount, nil)

	// Create service with mock repository
	service := services.NewAccountService(mockRepo)

	// Call the method
	result, err := service.GetAccountByAccountAddress(AccountAddress)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, expectedAccount, result)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestAccountService_GetAccountByAccountAddress_Error(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockAccountRepository()

	// Set up mock expectations for error case
	mockRepo.On("GetAccountByAccountAddress", AccountAddress).Return(nil, assert.AnError)

	// Create service with mock repository
	service := services.NewAccountService(mockRepo)

	// Call the method
	result, err := service.GetAccountByAccountAddress(AccountAddress)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, assert.AnError, err)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestAccountService_GetAccountProposals(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockAccountRepository()

	// Test data
	pagination := dto.PaginationQuery{
		Limit:      10,
		Offset:     0,
		CountTotal: true,
	}

	expectedProposals := []db.Proposal{
		{
			ID:         1,
			Title:      "Test Proposal",
			Status:     "PROPOSAL_STATUS_VOTING_PERIOD",
			Type:       "Text",
			ProposerID: AccountAddress,
		},
	}

	// Set up mock expectations
	mockRepo.On("GetAccountProposals", pagination, AccountAddress).Return(expectedProposals, int64(1), nil)

	// Create service with mock repository
	service := services.NewAccountService(mockRepo)

	// Call the method
	result, err := service.GetAccountProposals(pagination, AccountAddress)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Proposals, 1)
	assert.Equal(t, int64(1), result.Pagination.Total)
	assert.Equal(t, "Test Proposal", result.Proposals[0].Title)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestAccountService_GetAccountTxs(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockAccountRepository()

	// Test data
	pagination := dto.PaginationQuery{
		Limit:      10,
		Offset:     0,
		CountTotal: true,
	}

	expectedTxs := []dto.AccountTxModel{
		{
			Height:        1000,
			Timestamp:     "2024-01-01T10:00:00Z",
			Sender:        AccountAddress,
			Hash:          "tx_hash_1",
			Success:       true,
			IsSigner:      true,
			IsSend:        true,
			IsIbc:         false,
			IsMovePublish: false,
			IsMoveUpgrade: false,
			IsMoveExecute: false,
			IsMoveScript:  false,
			IsOpinit:      false,
		},
		{
			Height:        1001,
			Timestamp:     "2024-01-01T11:00:00Z",
			Sender:        AccountAddress,
			Hash:          "tx_hash_2",
			Success:       false,
			IsSigner:      false,
			IsSend:        false,
			IsIbc:         true,
			IsMovePublish: false,
			IsMoveUpgrade: false,
			IsMoveExecute: false,
			IsMoveScript:  false,
			IsOpinit:      false,
		},
		{
			Height:        1002,
			Timestamp:     "2024-01-01T12:00:00Z",
			Sender:        AccountAddress,
			Hash:          "tx_hash_3",
			Success:       true,
			IsSigner:      true,
			IsSend:        false,
			IsIbc:         false,
			IsMovePublish: true,
			IsMoveUpgrade: false,
			IsMoveExecute: false,
			IsMoveScript:  false,
			IsOpinit:      false,
		},
		{
			Height:        1003,
			Timestamp:     "2024-01-01T13:00:00Z",
			Sender:        AccountAddress,
			Hash:          "tx_hash_4",
			Success:       true,
			IsSigner:      false,
			IsSend:        false,
			IsIbc:         false,
			IsMovePublish: false,
			IsMoveUpgrade: false,
			IsMoveExecute: true,
			IsMoveScript:  false,
			IsOpinit:      false,
		},
		{
			Height:        1004,
			Timestamp:     "2024-01-01T14:00:00Z",
			Sender:        AccountAddress,
			Hash:          "tx_hash_5",
			Success:       true,
			IsSigner:      true,
			IsSend:        false,
			IsIbc:         false,
			IsMovePublish: false,
			IsMoveUpgrade: false,
			IsMoveExecute: false,
			IsMoveScript:  true,
			IsOpinit:      false,
		},
	}

	// Set up mock expectations
	mockRepo.On("GetAccountTxs", pagination, AccountAddress, "", false, false, false, false, false, false, false, (*bool)(nil)).Return(expectedTxs, int64(5), nil)

	// Create service with mock repository
	service := services.NewAccountService(mockRepo)

	// Call the method
	result, err := service.GetAccountTxs(
		pagination,
		AccountAddress,
		"",
		false,
		false,
		false,
		false,
		false,
		false,
		false,
		nil,
	)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.AccountTxs, 5)
	assert.Equal(t, int64(5), result.Pagination.Total)

	// Test individual transaction properties
	assert.Equal(t, "74785f686173685f31", result.AccountTxs[0].Hash) // hex encoded "tx_hash_1"
	assert.Equal(t, int64(1000), result.AccountTxs[0].Height)
	assert.True(t, result.AccountTxs[0].IsSigner)
	assert.True(t, result.AccountTxs[0].IsSend)

	assert.Equal(t, "74785f686173685f32", result.AccountTxs[1].Hash) // hex encoded "tx_hash_2"
	assert.Equal(t, int64(1001), result.AccountTxs[1].Height)
	assert.False(t, result.AccountTxs[1].IsSigner)
	assert.True(t, result.AccountTxs[1].IsIbc)

	assert.Equal(t, "74785f686173685f33", result.AccountTxs[2].Hash) // hex encoded "tx_hash_3"
	assert.True(t, result.AccountTxs[2].IsMovePublish)

	assert.Equal(t, "74785f686173685f34", result.AccountTxs[3].Hash) // hex encoded "tx_hash_4"
	assert.True(t, result.AccountTxs[3].IsMoveExecute)

	assert.Equal(t, "74785f686173685f35", result.AccountTxs[4].Hash) // hex encoded "tx_hash_5"
	assert.True(t, result.AccountTxs[4].IsMoveScript)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestAccountService_GetAccountProposals_Pagination(t *testing.T) {
	testCases := []struct {
		name               string
		pagination         dto.PaginationQuery
		expectedTotal      int64
		expectedLength     int
		expectedFirstTitle string
		expectedLastTitle  string
	}{
		{
			name: "Default pagination",
			pagination: dto.PaginationQuery{
				Limit:      10,
				Offset:     0,
				CountTotal: true,
			},
			expectedTotal:      11,
			expectedLength:     10,
			expectedFirstTitle: "Test Proposal 1",
			expectedLastTitle:  "Test Proposal 10",
		},
		{
			name: "Custom limit and offset",
			pagination: dto.PaginationQuery{
				Limit:      5,
				Offset:     5,
				CountTotal: true,
			},
			expectedTotal:      11,
			expectedLength:     5,
			expectedFirstTitle: "Test Proposal 6",
			expectedLastTitle:  "Test Proposal 10",
		},
		{
			name: "Large limit",
			pagination: dto.PaginationQuery{
				Limit:      100,
				Offset:     0,
				CountTotal: true,
			},
			expectedTotal:      11,
			expectedLength:     11,
			expectedFirstTitle: "Test Proposal 1",
			expectedLastTitle:  "Test Proposal 11",
		},
		{
			name: "Zero offset",
			pagination: dto.PaginationQuery{
				Limit:      10,
				Offset:     0,
				CountTotal: true,
			},
			expectedTotal:      11,
			expectedLength:     10,
			expectedFirstTitle: "Test Proposal 1",
			expectedLastTitle:  "Test Proposal 10",
		},
		{
			name: "Cases without count total",
			pagination: dto.PaginationQuery{
				Limit:  10,
				Offset: 0,
			},
			expectedTotal:      11,
			expectedLength:     10,
			expectedFirstTitle: "Test Proposal 1",
			expectedLastTitle:  "Test Proposal 10",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock repository for each test case
			mockRepo := mocks.NewMockAccountRepository()

			// Create the full dataset
			allProposals := []db.Proposal{
				{
					ID:         1,
					Title:      "Test Proposal 1",
					Status:     "PROPOSAL_STATUS_VOTING_PERIOD",
					Type:       "Text",
					ProposerID: AccountAddress,
				},
				{
					ID:         2,
					Title:      "Test Proposal 2",
					Status:     "PROPOSAL_STATUS_VOTING_PERIOD",
					Type:       "Text",
					ProposerID: AccountAddress,
				},
				{
					ID:         3,
					Title:      "Test Proposal 3",
					Status:     "PROPOSAL_STATUS_VOTING_PERIOD",
					Type:       "Text",
					ProposerID: AccountAddress,
				},
				{
					ID:         4,
					Title:      "Test Proposal 4",
					Status:     "PROPOSAL_STATUS_VOTING_PERIOD",
					Type:       "Text",
					ProposerID: AccountAddress,
				},
				{
					ID:         5,
					Title:      "Test Proposal 5",
					Status:     "PROPOSAL_STATUS_VOTING_PERIOD",
					Type:       "Text",
					ProposerID: AccountAddress,
				},
				{
					ID:         6,
					Title:      "Test Proposal 6",
					Status:     "PROPOSAL_STATUS_VOTING_PERIOD",
					Type:       "Text",
					ProposerID: AccountAddress,
				},
				{
					ID:         7,
					Title:      "Test Proposal 7",
					Status:     "PROPOSAL_STATUS_VOTING_PERIOD",
					Type:       "Text",
					ProposerID: AccountAddress,
				},
				{
					ID:         8,
					Title:      "Test Proposal 8",
					Status:     "PROPOSAL_STATUS_VOTING_PERIOD",
					Type:       "Text",
					ProposerID: AccountAddress,
				},
				{
					ID:         9,
					Title:      "Test Proposal 9",
					Status:     "PROPOSAL_STATUS_VOTING_PERIOD",
					Type:       "Text",
					ProposerID: AccountAddress,
				},
				{
					ID:         10,
					Title:      "Test Proposal 10",
					Status:     "PROPOSAL_STATUS_VOTING_PERIOD",
					Type:       "Text",
					ProposerID: AccountAddress,
				},
				{
					ID:         11,
					Title:      "Test Proposal 11",
					Status:     "PROPOSAL_STATUS_VOTING_PERIOD",
					Type:       "Text",
					ProposerID: AccountAddress,
				},
			}

			// Calculate the expected subset based on pagination
			start := tc.pagination.Offset
			end := start + tc.pagination.Limit
			if end > len(allProposals) {
				end = len(allProposals)
			}
			expectedProposals := allProposals[start:end]

			// Set up mock expectations for this specific test case
			mockRepo.On("GetAccountProposals", tc.pagination, AccountAddress).Return(expectedProposals, tc.expectedTotal, nil)

			// Create service with mock repository
			service := services.NewAccountService(mockRepo)

			// Call the method
			result, err := service.GetAccountProposals(tc.pagination, AccountAddress)

			// Assertions
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tc.expectedTotal, result.Pagination.Total)
			assert.Len(t, result.Proposals, tc.expectedLength)

			if tc.expectedLength > 0 {
				assert.Equal(t, tc.expectedFirstTitle, result.Proposals[0].Title)
				assert.Equal(t, tc.expectedLastTitle, result.Proposals[tc.expectedLength-1].Title)
			}

			// Verify mock was called as expected
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_GetAccountTxs_Pagination(t *testing.T) {
	testCases := []struct {
		name              string
		pagination        dto.PaginationQuery
		expectedTotal     int64
		expectedLength    int
		expectedFirstHash string
		expectedLastHash  string
	}{
		{
			name: "Default pagination",
			pagination: dto.PaginationQuery{
				Limit:      10,
				Offset:     0,
				CountTotal: true,
			},
			expectedTotal:     11,
			expectedLength:    10,
			expectedFirstHash: "74785f686173685f31",   // hex for "tx_hash_1"
			expectedLastHash:  "74785f686173685f3130", // hex for "tx_hash_10"
		},
		{
			name: "Small limit (3 items)",
			pagination: dto.PaginationQuery{
				Limit:      3,
				Offset:     0,
				CountTotal: true,
			},
			expectedTotal:     11,
			expectedLength:    3,
			expectedFirstHash: "74785f686173685f31", // hex for "tx_hash_1"
			expectedLastHash:  "74785f686173685f33", // hex for "tx_hash_3"
		},
		{
			name: "Offset pagination",
			pagination: dto.PaginationQuery{
				Limit:      6,
				Offset:     5,
				CountTotal: true,
			},
			expectedTotal:     11,
			expectedLength:    6,
			expectedFirstHash: "74785f686173685f36",   // hex for "tx_hash_6"
			expectedLastHash:  "74785f686173685f3131", // hex for "tx_hash_11"
		},
		{
			name: "Large limit",
			pagination: dto.PaginationQuery{
				Limit:      100,
				Offset:     0,
				CountTotal: true,
			},
			expectedTotal:     11,
			expectedLength:    11,
			expectedFirstHash: "74785f686173685f31",   // hex for "tx_hash_1"
			expectedLastHash:  "74785f686173685f3131", // hex for "tx_hash_11"
		},
		{
			name: "Cases without count total",
			pagination: dto.PaginationQuery{
				Limit:  10,
				Offset: 0,
			},
			expectedTotal:     11,
			expectedLength:    10,
			expectedFirstHash: "74785f686173685f31",   // hex for "tx_hash_1"
			expectedLastHash:  "74785f686173685f3130", // hex for "tx_hash_10"
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock repository for each test case
			mockRepo := mocks.NewMockAccountRepository()

			// Create the full dataset
			allTxs := []dto.AccountTxModel{
				{
					Height:        1000,
					Timestamp:     "2024-01-01T10:00:00Z",
					Sender:        AccountAddress,
					Hash:          "tx_hash_1",
					Success:       true,
					IsSigner:      true,
					IsSend:        true,
					IsIbc:         false,
					IsMovePublish: false,
					IsMoveUpgrade: false,
					IsMoveExecute: false,
					IsMoveScript:  false,
					IsOpinit:      false,
				},
				{
					Height:        1001,
					Timestamp:     "2024-01-01T11:00:00Z",
					Sender:        AccountAddress,
					Hash:          "tx_hash_2",
					Success:       false,
					IsSigner:      false,
					IsSend:        false,
					IsIbc:         true,
					IsMovePublish: false,
					IsMoveUpgrade: false,
					IsMoveExecute: false,
					IsMoveScript:  false,
					IsOpinit:      false,
				},
				{
					Height:        1002,
					Timestamp:     "2024-01-01T12:00:00Z",
					Sender:        AccountAddress,
					Hash:          "tx_hash_3",
					Success:       true,
					IsSigner:      true,
					IsSend:        false,
					IsIbc:         false,
					IsMovePublish: true,
					IsMoveUpgrade: false,
					IsMoveExecute: false,
					IsMoveScript:  false,
					IsOpinit:      false,
				},
				{
					Height:        1003,
					Timestamp:     "2024-01-01T13:00:00Z",
					Sender:        AccountAddress,
					Hash:          "tx_hash_4",
					Success:       true,
					IsSigner:      false,
					IsSend:        false,
					IsIbc:         false,
					IsMovePublish: false,
					IsMoveUpgrade: false,
					IsMoveExecute: true,
					IsMoveScript:  false,
					IsOpinit:      false,
				},
				{
					Height:        1004,
					Timestamp:     "2024-01-01T14:00:00Z",
					Sender:        AccountAddress,
					Hash:          "tx_hash_5",
					Success:       true,
					IsSigner:      true,
					IsSend:        false,
					IsIbc:         false,
					IsMovePublish: false,
					IsMoveUpgrade: false,
					IsMoveExecute: false,
					IsMoveScript:  true,
					IsOpinit:      false,
				},
				{
					Height:        1005,
					Timestamp:     "2024-01-01T15:00:00Z",
					Sender:        AccountAddress,
					Hash:          "tx_hash_6",
					Success:       true,
					IsSigner:      false,
					IsSend:        true,
					IsIbc:         true,
					IsMovePublish: false,
					IsMoveUpgrade: false,
					IsMoveExecute: false,
					IsMoveScript:  false,
					IsOpinit:      false,
				},
				{
					Height:        1006,
					Timestamp:     "2024-01-01T16:00:00Z",
					Sender:        AccountAddress,
					Hash:          "tx_hash_7",
					Success:       false,
					IsSigner:      true,
					IsSend:        false,
					IsIbc:         false,
					IsMovePublish: false,
					IsMoveUpgrade: true,
					IsMoveExecute: false,
					IsMoveScript:  false,
					IsOpinit:      false,
				},
				{
					Height:        1007,
					Timestamp:     "2024-01-01T17:00:00Z",
					Sender:        AccountAddress,
					Hash:          "tx_hash_8",
					Success:       true,
					IsSigner:      false,
					IsSend:        true,
					IsIbc:         true,
					IsMovePublish: false,
					IsMoveUpgrade: false,
					IsMoveExecute: false,
					IsMoveScript:  false,
					IsOpinit:      false,
				},
				{
					Height:        1008,
					Timestamp:     "2024-01-01T18:00:00Z",
					Sender:        AccountAddress,
					Hash:          "tx_hash_9",
					Success:       true,
					IsSigner:      true,
					IsSend:        false,
					IsIbc:         false,
					IsMovePublish: false,
					IsMoveUpgrade: false,
					IsMoveExecute: false,
					IsMoveScript:  false,
					IsOpinit:      false,
				},
				{
					Height:        1009,
					Timestamp:     "2024-01-01T19:00:00Z",
					Sender:        AccountAddress,
					Hash:          "tx_hash_10",
					Success:       true,
					IsSigner:      false,
					IsSend:        false,
					IsIbc:         false,
					IsMovePublish: false,
					IsMoveUpgrade: false,
					IsMoveExecute: false,
					IsMoveScript:  false,
					IsOpinit:      false,
				},
				{
					Height:        1010,
					Timestamp:     "2024-01-01T20:00:00Z",
					Sender:        AccountAddress,
					Hash:          "tx_hash_11",
					Success:       true,
					IsSigner:      true,
					IsSend:        false,
					IsIbc:         false,
					IsMovePublish: false,
					IsMoveUpgrade: false,
					IsMoveExecute: false,
					IsMoveScript:  false,
					IsOpinit:      false,
				},
			}

			// Calculate the expected subset based on pagination
			start := tc.pagination.Offset
			end := start + tc.pagination.Limit
			if end > len(allTxs) {
				end = len(allTxs)
			}
			expectedTxs := allTxs[start:end]

			// Set up mock expectations for this specific test case
			mockRepo.On("GetAccountTxs", tc.pagination, AccountAddress, "", false, false, false, false, false, false, false, (*bool)(nil)).Return(expectedTxs, tc.expectedTotal, nil)

			// Create service with mock repository
			service := services.NewAccountService(mockRepo)

			// Call the method
			result, err := service.GetAccountTxs(
				tc.pagination,
				AccountAddress,
				"",
				false,
				false,
				false,
				false,
				false,
				false,
				false,
				nil,
			)

			// Assertions
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tc.expectedTotal, result.Pagination.Total)
			assert.Len(t, result.AccountTxs, tc.expectedLength)

			if tc.expectedLength > 0 {
				assert.Equal(t, tc.expectedFirstHash, result.AccountTxs[0].Hash)
				assert.Equal(t, tc.expectedLastHash, result.AccountTxs[tc.expectedLength-1].Hash)
			}

			// Verify mock was called as expected
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_GetAccountProposals_PaginationEdgeCases(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockAccountRepository()

	t.Run("Empty result set", func(t *testing.T) {
		pagination := dto.PaginationQuery{
			Limit:      10,
			Offset:     0,
			CountTotal: true,
		}

		// Set up mock expectations for empty result
		mockRepo.On("GetAccountProposals", pagination, AccountAddress).Return([]db.Proposal{}, int64(0), nil)

		// Create service with mock repository
		service := services.NewAccountService(mockRepo)

		// Call the method
		result, err := service.GetAccountProposals(pagination, AccountAddress)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(0), result.Pagination.Total)
		assert.Len(t, result.Proposals, 0)
		assert.NotNil(t, result.Proposals) // Should be empty slice, not nil

		// Verify mock was called as expected
		mockRepo.AssertExpectations(t)
	})

	t.Run("Offset beyond total count", func(t *testing.T) {
		pagination := dto.PaginationQuery{
			Limit:      10,
			Offset:     100, // Beyond total count
			CountTotal: true,
		}

		// Set up mock expectations for offset beyond total
		mockRepo.On("GetAccountProposals", pagination, AccountAddress).Return([]db.Proposal{}, int64(25), nil)

		// Create service with mock repository
		service := services.NewAccountService(mockRepo)

		// Call the method
		result, err := service.GetAccountProposals(pagination, AccountAddress)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(25), result.Pagination.Total)
		assert.Len(t, result.Proposals, 0)

		// Verify mock was called as expected
		mockRepo.AssertExpectations(t)
	})
}

func TestAccountService_GetAccountTxs_PaginationEdgeCases(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockAccountRepository()

	t.Run("Empty result set", func(t *testing.T) {
		pagination := dto.PaginationQuery{
			Limit:      10,
			Offset:     0,
			CountTotal: true,
		}

		// Set up mock expectations for empty result
		mockRepo.On("GetAccountTxs", pagination, AccountAddress, "", false, false, false, false, false, false, false, (*bool)(nil)).Return([]dto.AccountTxModel{}, int64(0), nil)

		// Create service with mock repository
		service := services.NewAccountService(mockRepo)

		// Call the method
		result, err := service.GetAccountTxs(
			pagination,
			AccountAddress,
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			false,
			nil,
		)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(0), result.Pagination.Total)
		assert.Len(t, result.AccountTxs, 0)
		assert.NotNil(t, result.AccountTxs) // Should be empty slice, not nil

		// Verify mock was called as expected
		mockRepo.AssertExpectations(t)
	})

	t.Run("Limit 1 pagination", func(t *testing.T) {
		pagination := dto.PaginationQuery{
			Limit:      1,
			Offset:     0,
			CountTotal: true,
		}

		expectedTx := dto.AccountTxModel{
			Height:        1000,
			Timestamp:     "2024-01-01T10:00:00Z",
			Sender:        AccountAddress,
			Hash:          "tx_hash_1",
			Success:       true,
			IsSigner:      true,
			IsSend:        true,
			IsIbc:         false,
			IsMovePublish: false,
			IsMoveUpgrade: false,
			IsMoveExecute: false,
			IsMoveScript:  false,
			IsOpinit:      false,
		}

		// Set up mock expectations for single result
		mockRepo.On("GetAccountTxs", pagination, AccountAddress, "", false, false, false, false, false, false, false, (*bool)(nil)).Return([]dto.AccountTxModel{expectedTx}, int64(100), nil)

		// Create service with mock repository
		service := services.NewAccountService(mockRepo)

		// Call the method
		result, err := service.GetAccountTxs(
			pagination,
			AccountAddress,
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			false,
			nil,
		)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(100), result.Pagination.Total)
		assert.Len(t, result.AccountTxs, 1)
		assert.Equal(t, "74785f686173685f31", result.AccountTxs[0].Hash) // hex encoded "tx_hash_1"

		// Verify mock was called as expected
		mockRepo.AssertExpectations(t)
	})
}
