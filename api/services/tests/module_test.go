package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories/mocks"
	"github.com/initia-labs/core-indexer/api/services"
)

func TestModuleService_GetModules(t *testing.T) {
	tests := []struct {
		name           string
		pagination     dto.PaginationQuery
		mockModules    []dto.ModuleResponse
		mockTotal      int64
		mockError      error
		expectedResult *dto.ModulesResponse
		expectedError  error
	}{
		{
			name: "successful get modules",
			pagination: dto.PaginationQuery{
				Limit:      10,
				Offset:     0,
				CountTotal: true,
			},
			mockModules: []dto.ModuleResponse{
				{
					ModuleName:    "test_module",
					Digest:        "abc123",
					IsVerify:      true,
					Address:       "0x123",
					Height:        100,
					LatestUpdated: "2023-01-01T00:00:00Z",
					IsRepublished: false,
				},
			},
			mockTotal: 1,
			mockError: nil,
			expectedResult: &dto.ModulesResponse{
				Modules: []dto.ModuleResponse{
					{
						ModuleName:    "test_module",
						Digest:        "abc123",
						IsVerify:      true,
						Address:       "0x123",
						Height:        100,
						LatestUpdated: "2023-01-01T00:00:00Z",
						IsRepublished: false,
					},
				},
				Pagination: dto.NewPaginationResponse(0, 10, 1),
			},
			expectedError: nil,
		},
		{
			name: "repository error",
			pagination: dto.PaginationQuery{
				Limit:  10,
				Offset: 0,
			},
			mockModules:    nil,
			mockTotal:      0,
			mockError:      errors.New("database error"),
			expectedResult: nil,
			expectedError:  errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockModuleRepository()
			service := services.NewModuleService(mockRepo)

			mockRepo.On("GetModules", context.Background(), tt.pagination).Return(tt.mockModules, tt.mockTotal, tt.mockError)

			result, err := service.GetModules(context.Background(), tt.pagination)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestModuleService_GetModuleById(t *testing.T) {
	tests := []struct {
		name           string
		vmAddress      string
		moduleName     string
		mockModule     *dto.ModuleResponse
		mockError      error
		expectedResult *dto.ModuleResponse
		expectedError  error
	}{
		{
			name:       "successful get module by id",
			vmAddress:  "0x123",
			moduleName: "test_module",
			mockModule: &dto.ModuleResponse{
				ModuleName:    "test_module",
				Digest:        "abc123",
				IsVerify:      true,
				Address:       "0x123",
				Height:        100,
				LatestUpdated: "2023-01-01T00:00:00Z",
				IsRepublished: false,
			},
			mockError: nil,
			expectedResult: &dto.ModuleResponse{
				ModuleName:    "test_module",
				Digest:        "abc123",
				IsVerify:      true,
				Address:       "0x123",
				Height:        100,
				LatestUpdated: "2023-01-01T00:00:00Z",
				IsRepublished: false,
			},
			expectedError: nil,
		},
		{
			name:           "module not found",
			vmAddress:      "0x123",
			moduleName:     "nonexistent_module",
			mockModule:     nil,
			mockError:      errors.New("module 0x123::nonexistent_module not found"),
			expectedResult: nil,
			expectedError:  errors.New("module 0x123::nonexistent_module not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockModuleRepository()
			service := services.NewModuleService(mockRepo)

			mockRepo.On("GetModuleById", context.Background(), tt.vmAddress, tt.moduleName).Return(tt.mockModule, tt.mockError)

			result, err := service.GetModuleById(context.Background(), tt.vmAddress, tt.moduleName)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestModuleService_GetModuleHistories(t *testing.T) {
	tests := []struct {
		name           string
		pagination     dto.PaginationQuery
		vmAddress      string
		moduleName     string
		mockHistories  []dto.ModuleHistoryResponse
		mockTotal      int64
		mockError      error
		expectedResult *dto.ModuleHistoriesResponse
		expectedError  error
	}{
		{
			name: "successful get module histories with multiple entries",
			pagination: dto.PaginationQuery{
				Limit:      10,
				Offset:     0,
				CountTotal: true,
			},
			vmAddress:  "0x123",
			moduleName: "test_module",
			mockHistories: []dto.ModuleHistoryResponse{
				{
					Height:        100,
					Remark:        []byte(`{"key": "value"}`),
					UpgradePolicy: "immutable",
					Timestamp:     "2023-01-01T00:00:00Z",
				},
				{
					Height:        99,
					Remark:        []byte(`{"key": "old_value"}`),
					UpgradePolicy: "compatible",
					Timestamp:     "2023-01-01T00:00:00Z",
				},
			},
			mockTotal: 2,
			mockError: nil,
			expectedResult: &dto.ModuleHistoriesResponse{
				ModuleHistories: []dto.ModuleHistoryResponse{
					{
						Height:        100,
						Remark:        []byte(`{"key": "value"}`),
						UpgradePolicy: "IMMUTABLE",
						Timestamp:     "2023-01-01T00:00:00Z",
						PreviousPolicy: func() *string {
							s := "COMPATIBLE"
							return &s
						}(),
					},
					{
						Height:         99,
						Remark:         []byte(`{"key": "old_value"}`),
						UpgradePolicy:  "COMPATIBLE",
						Timestamp:      "2023-01-01T00:00:00Z",
						PreviousPolicy: nil,
					},
				},
				Pagination: dto.NewPaginationResponse(0, 10, 2),
			},
			expectedError: nil,
		},
		{
			name: "repository error",
			pagination: dto.PaginationQuery{
				Limit:  10,
				Offset: 0,
			},
			vmAddress:      "0x123",
			moduleName:     "test_module",
			mockHistories:  nil,
			mockTotal:      0,
			mockError:      errors.New("database error"),
			expectedResult: nil,
			expectedError:  errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockModuleRepository()
			service := services.NewModuleService(mockRepo)

			mockRepo.On("GetModuleHistories", context.Background(), tt.pagination, tt.vmAddress, tt.moduleName).Return(tt.mockHistories, tt.mockTotal, tt.mockError)

			result, err := service.GetModuleHistories(context.Background(), tt.pagination, tt.vmAddress, tt.moduleName)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestModuleService_GetModulePublishInfo(t *testing.T) {
	tests := []struct {
		name            string
		vmAddress       string
		moduleName      string
		mockPublishInfo []dto.ModulePublishInfoModel
		mockError       error
		expectedResult  *dto.ModulePublishInfoResponse
		expectedError   error
	}{
		{
			name:       "successful get module publish info with transaction hash",
			vmAddress:  "0x123",
			moduleName: "test_module",
			mockPublishInfo: []dto.ModulePublishInfoModel{
				{
					Height:    100,
					Proposal:  nil,
					Timestamp: "2023-01-01T00:00:00Z",
					TransactionHash: func() *string {
						s := "0xabc123"
						return &s
					}(),
				},
			},
			mockError: nil,
			expectedResult: &dto.ModulePublishInfoResponse{
				RecentPublishTransaction: func() *string {
					s := "3078616263313233" // hex representation of "0xabc123"
					return &s
				}(),
				IsRepublished:               false,
				RecentPublishBlockHeight:    100,
				RecentPublishBlockTimestamp: "2023-01-01T00:00:00Z",
				RecentPublishProposal:       nil,
			},
			expectedError: nil,
		},
		{
			name:       "successful get module publish info with republished module",
			vmAddress:  "0x123",
			moduleName: "test_module",
			mockPublishInfo: []dto.ModulePublishInfoModel{
				{
					Height:          100,
					Proposal:        nil,
					Timestamp:       "2023-01-01T00:00:00Z",
					TransactionHash: nil,
				},
				{
					Height:          99,
					Proposal:        nil,
					Timestamp:       "2023-01-01T00:00:00Z",
					TransactionHash: nil,
				},
			},
			mockError: nil,
			expectedResult: &dto.ModulePublishInfoResponse{
				RecentPublishTransaction:    nil,
				IsRepublished:               true,
				RecentPublishBlockHeight:    100,
				RecentPublishBlockTimestamp: "2023-01-01T00:00:00Z",
				RecentPublishProposal:       nil,
			},
			expectedError: nil,
		},
		{
			name:            "no publish info found",
			vmAddress:       "0x123",
			moduleName:      "test_module",
			mockPublishInfo: []dto.ModulePublishInfoModel{},
			mockError:       nil,
			expectedResult:  nil,
			expectedError:   gorm.ErrRecordNotFound,
		},
		{
			name:            "repository error",
			vmAddress:       "0x123",
			moduleName:      "test_module",
			mockPublishInfo: nil,
			mockError:       errors.New("database error"),
			expectedResult:  nil,
			expectedError:   errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockModuleRepository()
			service := services.NewModuleService(mockRepo)

			mockRepo.On("GetModulePublishInfo", context.Background(), tt.vmAddress, tt.moduleName).Return(tt.mockPublishInfo, tt.mockError)

			result, err := service.GetModulePublishInfo(context.Background(), tt.vmAddress, tt.moduleName)

			if tt.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
				} else {
					assert.Equal(t, tt.expectedError.Error(), err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestModuleService_GetModuleProposals(t *testing.T) {
	tests := []struct {
		name           string
		pagination     dto.PaginationQuery
		vmAddress      string
		moduleName     string
		mockProposals  []dto.ModuleProposalModel
		mockTotal      int64
		mockError      error
		expectedResult *dto.ModuleProposalsResponse
		expectedError  error
	}{
		{
			name: "successful get module proposals",
			pagination: dto.PaginationQuery{
				Limit:      10,
				Offset:     0,
				CountTotal: true,
			},
			vmAddress:  "0x123",
			moduleName: "test_module",
			mockProposals: []dto.ModuleProposalModel{
				{
					ID:             1,
					Title:          "Test Proposal",
					Status:         "PROPOSAL_STATUS_PASSED",
					VotingEndTime:  "2023-01-01T00:00:00Z",
					DepositEndTime: "2023-01-01T00:00:00Z",
					Types:          []byte(`["gov"]`),
					IsExpedited:    false,
					IsEmergency:    false,
					ResolvedHeight: 100,
					Proposer:       "0x456",
				},
			},
			mockTotal: 1,
			mockError: nil,
			expectedResult: &dto.ModuleProposalsResponse{
				Proposals: []dto.ModuleProposalModel{
					{
						ID:             1,
						Title:          "Test Proposal",
						Status:         "PROPOSAL_STATUS_PASSED",
						VotingEndTime:  "2023-01-01T00:00:00Z",
						DepositEndTime: "2023-01-01T00:00:00Z",
						Types:          []byte(`["gov"]`),
						IsExpedited:    false,
						IsEmergency:    false,
						ResolvedHeight: 100,
						Proposer:       "0x456",
					},
				},
				Pagination: dto.NewPaginationResponse(0, 10, 1),
			},
			expectedError: nil,
		},
		{
			name: "repository error",
			pagination: dto.PaginationQuery{
				Limit:  10,
				Offset: 0,
			},
			vmAddress:      "0x123",
			moduleName:     "test_module",
			mockProposals:  nil,
			mockTotal:      0,
			mockError:      errors.New("database error"),
			expectedResult: nil,
			expectedError:  errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockModuleRepository()
			service := services.NewModuleService(mockRepo)

			mockRepo.On("GetModuleProposals", context.Background(), tt.pagination, tt.vmAddress, tt.moduleName).Return(tt.mockProposals, tt.mockTotal, tt.mockError)

			result, err := service.GetModuleProposals(context.Background(), tt.pagination, tt.vmAddress, tt.moduleName)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestModuleService_GetModuleTransactions(t *testing.T) {
	tests := []struct {
		name           string
		pagination     dto.PaginationQuery
		vmAddress      string
		moduleName     string
		mockTxs        []dto.ModuleTxResponse
		mockTotal      int64
		mockError      error
		expectedResult *dto.ModuleTxsResponse
		expectedError  error
	}{
		{
			name: "successful get module transactions",
			pagination: dto.PaginationQuery{
				Limit:      10,
				Offset:     0,
				CountTotal: true,
			},
			vmAddress:  "0x123",
			moduleName: "test_module",
			mockTxs: []dto.ModuleTxResponse{
				{
					Height:             100,
					Timestamp:          "2023-01-01T00:00:00Z",
					Sender:             "0x456",
					TxHash:             "0xabc123",
					Success:            true,
					Messages:           []byte(`[{"type": "move_execute"}]`),
					IsSend:             false,
					IsIBC:              false,
					IsMoveExecute:      true,
					IsMoveExecuteEvent: false,
					IsMovePublish:      false,
					IsMoveScript:       false,
					IsMoveUpgrade:      false,
					IsOpinit:           false,
				},
			},
			mockTotal: 1,
			mockError: nil,
			expectedResult: &dto.ModuleTxsResponse{
				ModuleTxs: []dto.ModuleTxResponse{
					{
						Height:             100,
						Timestamp:          "2023-01-01T00:00:00Z",
						Sender:             "0x456",
						TxHash:             "3078616263313233", // Fixed: hex encoding of "0xabc123"
						Success:            true,
						Messages:           []byte(`[{"type": "move_execute"}]`),
						IsSend:             false,
						IsIBC:              false,
						IsMoveExecute:      true,
						IsMoveExecuteEvent: false,
						IsMovePublish:      false,
						IsMoveScript:       false,
						IsMoveUpgrade:      false,
						IsOpinit:           false,
					},
				},
				Pagination: dto.NewPaginationResponse(0, 10, 1),
			},
			expectedError: nil,
		},
		{
			name: "repository error",
			pagination: dto.PaginationQuery{
				Limit:  10,
				Offset: 0,
			},
			vmAddress:      "0x123",
			moduleName:     "test_module",
			mockTxs:        nil,
			mockTotal:      0,
			mockError:      errors.New("database error"),
			expectedResult: nil,
			expectedError:  errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockModuleRepository()
			service := services.NewModuleService(mockRepo)

			mockRepo.On("GetModuleTransactions", context.Background(), tt.pagination, tt.vmAddress, tt.moduleName).Return(tt.mockTxs, tt.mockTotal, tt.mockError)

			result, err := service.GetModuleTransactions(context.Background(), tt.pagination, tt.vmAddress, tt.moduleName)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestModuleService_GetModuleStats(t *testing.T) {
	tests := []struct {
		name           string
		vmAddress      string
		moduleName     string
		mockStats      *dto.ModuleStatsResponse
		mockError      error
		expectedResult *dto.ModuleStatsResponse
		expectedError  error
	}{
		{
			name:       "successful get module stats",
			vmAddress:  "0x123",
			moduleName: "test_module",
			mockStats: &dto.ModuleStatsResponse{
				TotalHistories: 5,
				TotalProposals: func() *int64 {
					var count int64 = 2
					return &count
				}(),
				TotalTxs: 10,
			},
			mockError: nil,
			expectedResult: &dto.ModuleStatsResponse{
				TotalHistories: 5,
				TotalProposals: func() *int64 {
					var count int64 = 2
					return &count
				}(),
				TotalTxs: 10,
			},
			expectedError: nil,
		},
		{
			name:           "repository error",
			vmAddress:      "0x123",
			moduleName:     "test_module",
			mockStats:      nil,
			mockError:      errors.New("database error"),
			expectedResult: nil,
			expectedError:  errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockModuleRepository()
			service := services.NewModuleService(mockRepo)

			mockRepo.On("GetModuleStats", context.Background(), tt.vmAddress, tt.moduleName).Return(tt.mockStats, tt.mockError)

			result, err := service.GetModuleStats(context.Background(), tt.vmAddress, tt.moduleName)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestModuleService_NewModuleService(t *testing.T) {
	mockRepo := mocks.NewMockModuleRepository()
	service := services.NewModuleService(mockRepo)

	assert.NotNil(t, service)
	assert.Implements(t, (*services.ModuleService)(nil), service)
}
