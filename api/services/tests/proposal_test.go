package services_test

// import (
// 	"context"
// 	"fmt"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/assert"

// 	"github.com/initia-labs/core-indexer/api/dto"
// 	"github.com/initia-labs/core-indexer/api/repositories/mocks"
// 	"github.com/initia-labs/core-indexer/api/services"
// )

// const (
// 	TestProposalID = 1
// 	TestProposer   = "init1m8p6rakcfl4z5ruwa0578cqgn8c86mkc6ety2z"
// )

// // func TestProposalService_GetProposals(t *testing.T) {
// // 	// Create mock repository
// // 	mockRepo := mocks.NewMockProposalRepository()

// // 	// Test data
// // 	pagination := dto.PaginationQuery{
// // 		Limit:      10,
// // 		Offset:     0,
// // 		CountTotal: true,
// // 	}

// // 	expectedProposals := []dto.ProposalSummary{
// // 		{
// // 			Id:             1,
// // 			Title:          "Test Proposal 1",
// // 			Status:         "VotingPeriod",
// // 			Proposer:       TestProposer,
// // 			DepositEndTime: time.Now().Add(24 * time.Hour),
// // 			VotingEndTime:  &[]time.Time{time.Now().Add(48 * time.Hour)}[0],
// // 			ResolvedHeight: 1000,
// // 			IsEmergency:    false,
// // 			IsExpedited:    false,
// // 		},
// // 		{
// // 			Id:             2,
// // 			Title:          "Test Proposal 2",
// // 			Status:         "Passed",
// // 			Proposer:       TestProposer,
// // 			DepositEndTime: time.Now().Add(-24 * time.Hour),
// // 			VotingEndTime:  &[]time.Time{time.Now().Add(-12 * time.Hour)}[0],
// // 			ResolvedHeight: 2000,
// // 			IsEmergency:    true,
// // 			IsExpedited:    true,
// // 		},
// // 	}

// // 	// Set up mock expectations
// // 	mockRepo.On("SearchProposals", context.Background(), pagination, "", "", []string{}, []string{}).Return(expectedProposals, int64(2), nil)

// // 	// Create service with mock repository
// // 	service := services.NewProposalService(mockRepo)

// // 	// Call the method
// // 	result, err := service.GetProposals(context.Background(), pagination, "", "", "", "")

// // 	// Assertions
// // 	assert.NoError(t, err)
// // 	assert.NotNil(t, result)
// // 	assert.Len(t, result.Proposals, 2)
// // 	assert.Equal(t, fmt.Sprintf("%d", int64(2)), result.Pagination.Total)
// // 	assert.Equal(t, "Test Proposal 1", result.Proposals[0].Title)
// // 	assert.Equal(t, "Test Proposal 2", result.Proposals[1].Title)

// // 	// Verify mock was called as expected
// // 	mockRepo.AssertExpectations(t)
// // }

// func TestProposalService_GetProposals_DuplicateStatus(t *testing.T) {
// 	// Create mock repository
// 	mockRepo := mocks.NewMockProposalRepository()

// 	// Test data
// 	pagination := dto.PaginationQuery{
// 		Limit:      10,
// 		Offset:     0,
// 		CountTotal: true,
// 	}

// 	// Create service with mock repository
// 	service := services.NewProposalService(mockRepo)

// 	// Call the method with duplicate status
// 	result, err := service.GetProposals(context.Background(), pagination, "", "VotingPeriod,VotingPeriod", "", "")

// 	// Assertions
// 	assert.Error(t, err)
// 	assert.Nil(t, result)

// 	// Verify no mock calls were made
// 	mockRepo.AssertNotCalled(t, "SearchProposals")
// }

// func TestProposalService_GetProposalsTypes(t *testing.T) {
// 	// Create mock repository
// 	mockRepo := mocks.NewMockProposalRepository()

// 	// Test data
// 	expectedTypes := &dto.ProposalsTypesResponse{
// 		"Text",
// 		"ParameterChange",
// 		"SoftwareUpgrade",
// 	}

// 	// Set up mock expectations
// 	mockRepo.On("GetAllProposalTypes", context.Background()).Return(expectedTypes, nil)

// 	// Create service with mock repository
// 	service := services.NewProposalService(mockRepo)

// 	// Call the method
// 	result, err := service.GetProposalsTypes(context.Background())

// 	// Assertions
// 	assert.NoError(t, err)
// 	assert.NotNil(t, result)
// 	assert.Len(t, *result, 3)
// 	assert.Contains(t, *result, "Text")
// 	assert.Contains(t, *result, "ParameterChange")
// 	assert.Contains(t, *result, "SoftwareUpgrade")

// 	// Verify mock was called as expected
// 	mockRepo.AssertExpectations(t)
// }

// func TestProposalService_GetProposalInfo(t *testing.T) {
// 	// Create mock repository
// 	mockRepo := mocks.NewMockProposalRepository()

// 	// Test data
// 	expectedProposal := &dto.ProposalInfo{
// 		Id:                       TestProposalID,
// 		Title:                    "Test Proposal",
// 		Description:              "This is a test proposal",
// 		Status:                   "VotingPeriod",
// 		Proposer:                 TestProposer,
// 		SubmitTime:               time.Now(),
// 		DepositEndTime:           time.Now().Add(24 * time.Hour),
// 		VotingEndTime:            &[]time.Time{time.Now().Add(48 * time.Hour)}[0],
// 		CreatedHeight:            1000,
// 		CreatedTimestamp:         time.Now(),
// 		CreatedTxHash:            "abc123",
// 		IsEmergency:              false,
// 		IsExpedited:              false,
// 		Yes:                      "1000000",
// 		No:                       "500000",
// 		NoWithVeto:               "100000",
// 		Abstain:                  "200000",
// 		TotalDeposit:             dto.Coins{},
// 		ProposalDeposits:         []dto.ProposalDeposit{},
// 		ResolvedTotalVotingPower: nil,
// 	}

// 	// Set up mock expectations
// 	mockRepo.On("GetProposalInfo", context.Background(), TestProposalID).Return(expectedProposal, nil)

// 	// Create service with mock repository
// 	service := services.NewProposalService(mockRepo)

// 	// Call the method
// 	result, err := service.GetProposalInfo(context.Background(), TestProposalID)

// 	// Assertions
// 	assert.NoError(t, err)
// 	assert.NotNil(t, result)
// 	assert.Equal(t, expectedProposal, &result.Info)
// 	assert.Equal(t, "Test Proposal", result.Info.Title)
// 	assert.Equal(t, TestProposer, result.Info.Proposer)

// 	// Verify mock was called as expected
// 	mockRepo.AssertExpectations(t)
// }

// func TestProposalService_GetProposalVotes(t *testing.T) {
// 	// Create mock repository
// 	mockRepo := mocks.NewMockProposalRepository()

// 	// Test data
// 	pagination := dto.PaginationQuery{
// 		Limit:      10,
// 		Offset:     0,
// 		CountTotal: true,
// 	}

// 	expectedVotes := []dto.ProposalVote{
// 		{
// 			ProposalId:     TestProposalID,
// 			Voter:          "init1voter1",
// 			Yes:            1,
// 			No:             0,
// 			NoWithVeto:     0,
// 			Abstain:        0,
// 			IsVoteWeighted: false,
// 			TxHash:         &[]string{"tx_hash_1"}[0],
// 			Timestamp:      &[]time.Time{time.Now()}[0],
// 		},
// 		{
// 			ProposalId:     TestProposalID,
// 			Voter:          "init1voter2",
// 			Yes:            0,
// 			No:             1,
// 			NoWithVeto:     0,
// 			Abstain:        0,
// 			IsVoteWeighted: false,
// 			TxHash:         &[]string{"tx_hash_2"}[0],
// 			Timestamp:      &[]time.Time{time.Now()}[0],
// 		},
// 	}

// 	// Set up mock expectations
// 	mockRepo.On("GetProposalVotes", context.Background(), TestProposalID, int64(10), int64(0), "", "").Return(expectedVotes, int64(2), nil)

// 	// Create service with mock repository
// 	service := services.NewProposalService(mockRepo)

// 	// Call the method
// 	result, err := service.GetProposalVotes(context.Background(), pagination, TestProposalID, "", "")

// 	// Assertions
// 	assert.NoError(t, err)
// 	assert.NotNil(t, result)
// 	assert.Len(t, result.Votes, 2)
// 	assert.Equal(t, fmt.Sprintf("%d", int64(2)), result.Pagination.Total)
// 	assert.Equal(t, "init1voter1", result.Votes[0].Voter)
// 	assert.Equal(t, "init1voter2", result.Votes[1].Voter)

// 	// Verify mock was called as expected
// 	mockRepo.AssertExpectations(t)
// }

// func TestProposalService_GetProposalValidatorVotes(t *testing.T) {
// 	// Create mock repository
// 	mockRepo := mocks.NewMockProposalRepository()

// 	// Test data
// 	pagination := dto.PaginationQuery{
// 		Limit:      10,
// 		Offset:     0,
// 		CountTotal: true,
// 	}

// 	expectedVotes := []dto.ProposalVote{
// 		{
// 			ProposalId: TestProposalID,
// 			Validator: &dto.ProposalValidatorVote{
// 				Moniker:          "Validator 1",
// 				Identity:         "identity1",
// 				ValidatorAddress: "init1validator1",
// 			},
// 			Yes:            1,
// 			No:             0,
// 			NoWithVeto:     0,
// 			Abstain:        0,
// 			IsVoteWeighted: false,
// 			Voter:          "init1validator1",
// 			TxHash:         &[]string{"tx_hash_1"}[0],
// 			Timestamp:      &[]time.Time{time.Now()}[0],
// 		},
// 		{
// 			ProposalId: TestProposalID,
// 			Validator: &dto.ProposalValidatorVote{
// 				Moniker:          "Validator 2",
// 				Identity:         "identity2",
// 				ValidatorAddress: "init1validator2",
// 			},
// 			Yes:            0,
// 			No:             1,
// 			NoWithVeto:     0,
// 			Abstain:        0,
// 			IsVoteWeighted: false,
// 			Voter:          "init1validator2",
// 			TxHash:         &[]string{"tx_hash_2"}[0],
// 			Timestamp:      &[]time.Time{time.Now()}[0],
// 		},
// 	}

// 	// Set up mock expectations
// 	mockRepo.On("GetProposalValidatorVotes", context.Background(), TestProposalID).Return(expectedVotes, nil)

// 	// Create service with mock repository
// 	service := services.NewProposalService(mockRepo)

// 	// Call the method
// 	result, err := service.GetProposalValidatorVotes(context.Background(), pagination, TestProposalID, "", "")

// 	// Assertions
// 	assert.NoError(t, err)
// 	assert.NotNil(t, result)
// 	assert.Len(t, result.Votes, 2)
// 	assert.Equal(t, fmt.Sprintf("%d", int64(2)), result.Pagination.Total)
// 	assert.Equal(t, "Validator 1", result.Votes[0].Validator.Moniker)
// 	assert.Equal(t, "Validator 2", result.Votes[1].Validator.Moniker)

// 	// Verify mock was called as expected
// 	mockRepo.AssertExpectations(t)
// }

// func TestProposalService_GetProposalValidatorVotes_WithSearch(t *testing.T) {
// 	// Create mock repository
// 	mockRepo := mocks.NewMockProposalRepository()

// 	// Test data
// 	pagination := dto.PaginationQuery{
// 		Limit:      10,
// 		Offset:     0,
// 		CountTotal: true,
// 	}

// 	allVotes := []dto.ProposalVote{
// 		{
// 			ProposalId: TestProposalID,
// 			Validator: &dto.ProposalValidatorVote{
// 				Moniker:          "Validator Alpha",
// 				Identity:         "identity1",
// 				ValidatorAddress: "init1validator1",
// 			},
// 			Yes:            1,
// 			No:             0,
// 			NoWithVeto:     0,
// 			Abstain:        0,
// 			IsVoteWeighted: false,
// 			Voter:          "init1validator1",
// 			TxHash:         &[]string{"tx_hash_1"}[0],
// 			Timestamp:      &[]time.Time{time.Now()}[0],
// 		},
// 		{
// 			ProposalId: TestProposalID,
// 			Validator: &dto.ProposalValidatorVote{
// 				Moniker:          "Validator Beta",
// 				Identity:         "identity2",
// 				ValidatorAddress: "init1validator2",
// 			},
// 			Yes:            0,
// 			No:             1,
// 			NoWithVeto:     0,
// 			Abstain:        0,
// 			IsVoteWeighted: false,
// 			Voter:          "init1validator2",
// 			TxHash:         &[]string{"tx_hash_2"}[0],
// 			Timestamp:      &[]time.Time{time.Now()}[0],
// 		},
// 	}

// 	// Set up mock expectations
// 	mockRepo.On("GetProposalValidatorVotes", context.Background(), TestProposalID).Return(allVotes, nil)

// 	// Create service with mock repository
// 	service := services.NewProposalService(mockRepo)

// 	// Call the method with search filter
// 	result, err := service.GetProposalValidatorVotes(context.Background(), pagination, TestProposalID, "alpha", "")

// 	// Assertions
// 	assert.NoError(t, err)
// 	assert.NotNil(t, result)
// 	assert.Len(t, result.Votes, 1) // Only "Validator Alpha" should match
// 	assert.Equal(t, "Validator Alpha", result.Votes[0].Validator.Moniker)

// 	// Verify mock was called as expected
// 	mockRepo.AssertExpectations(t)
// }

// func TestProposalService_GetProposalAnswerCounts(t *testing.T) {
// 	// Create mock repository
// 	mockRepo := mocks.NewMockProposalRepository()

// 	// Test data
// 	expectedCounts := &dto.ProposalAnswerCountsResponse{
// 		All: dto.ProposalAnswerCounts{
// 			Yes:        10,
// 			No:         5,
// 			NoWithVeto: 2,
// 			Abstain:    3,
// 			Weighted:   1,
// 			Total:      21,
// 		},
// 		Validator: dto.ProposalValidatorAnswerCounts{
// 			ProposalAnswerCounts: dto.ProposalAnswerCounts{
// 				Yes:        8,
// 				No:         3,
// 				NoWithVeto: 1,
// 				Abstain:    2,
// 				Weighted:   0,
// 				Total:      14,
// 			},
// 			DidNotVote:      6,
// 			TotalValidators: 20,
// 		},
// 	}

// 	// Set up mock expectations
// 	mockRepo.On("GetProposalAnswerCounts", TestProposalID).Return(expectedCounts, nil)

// 	// Create service with mock repository
// 	service := services.NewProposalService(mockRepo)

// 	// Call the method
// 	result, err := service.GetProposalAnswerCounts(context.Background(), TestProposalID)

// 	// Assertions
// 	assert.NoError(t, err)
// 	assert.NotNil(t, result)
// 	assert.Equal(t, int(10), result.All.Yes)
// 	assert.Equal(t, int(5), result.All.No)
// 	assert.Equal(t, int(21), result.All.Total)
// 	assert.Equal(t, int(8), result.Validator.Yes)
// 	assert.Equal(t, int(6), result.Validator.DidNotVote)
// 	assert.Equal(t, int(20), result.Validator.TotalValidators)

// 	// Verify mock was called as expected
// 	mockRepo.AssertExpectations(t)
// }

// func TestProposalService_GetProposalInfo_RepositoryError(t *testing.T) {
// 	// Create mock repository
// 	mockRepo := mocks.NewMockProposalRepository()

// 	// Set up mock expectations for error
// 	mockRepo.On("GetProposalInfo", TestProposalID).Return(nil, assert.AnError)

// 	// Create service with mock repository
// 	service := services.NewProposalService(mockRepo)

// 	// Call the method
// 	result, err := service.GetProposalInfo(context.Background(), TestProposalID)

// 	// Assertions
// 	assert.Error(t, err)
// 	assert.Nil(t, result)
// 	assert.Equal(t, assert.AnError, err)

// 	// Verify mock was called as expected
// 	mockRepo.AssertExpectations(t)
// }

// func TestProposalService_GetProposalValidatorVotes_Pagination(t *testing.T) {
// 	// Create mock repository
// 	mockRepo := mocks.NewMockProposalRepository()

// 	// Test data with pagination
// 	pagination := dto.PaginationQuery{
// 		Limit:      2,
// 		Offset:     1,
// 		CountTotal: true,
// 	}

// 	allVotes := []dto.ProposalVote{
// 		{
// 			ProposalId: TestProposalID,
// 			Validator: &dto.ProposalValidatorVote{
// 				Moniker:          "Validator 1",
// 				Identity:         "identity1",
// 				ValidatorAddress: "init1validator1",
// 			},
// 			Yes: 1,
// 		},
// 		{
// 			ProposalId: TestProposalID,
// 			Validator: &dto.ProposalValidatorVote{
// 				Moniker:          "Validator 2",
// 				Identity:         "identity2",
// 				ValidatorAddress: "init1validator2",
// 			},
// 			No: 1,
// 		},
// 		{
// 			ProposalId: TestProposalID,
// 			Validator: &dto.ProposalValidatorVote{
// 				Moniker:          "Validator 3",
// 				Identity:         "identity3",
// 				ValidatorAddress: "init1validator3",
// 			},
// 			Abstain: 1,
// 		},
// 	}

// 	// Set up mock expectations
// 	mockRepo.On("GetProposalValidatorVotes", context.Background(), TestProposalID).Return(allVotes, nil)

// 	// Create service with mock repository
// 	service := services.NewProposalService(mockRepo)

// 	// Call the method
// 	result, err := service.GetProposalValidatorVotes(context.Background(), pagination, TestProposalID, "", "")

// 	// Assertions
// 	assert.NoError(t, err)
// 	assert.NotNil(t, result)
// 	assert.Len(t, result.Votes, 2)                                        // Should return 2 items due to limit
// 	assert.Equal(t, fmt.Sprintf("%d", int64(3)), result.Pagination.Total) // Total should be 3
// 	assert.Equal(t, "Validator 2", result.Votes[0].Validator.Moniker)     // Should start from offset 1
// 	assert.Equal(t, "Validator 3", result.Votes[1].Validator.Moniker)

// 	// Verify mock was called as expected
// 	mockRepo.AssertExpectations(t)
// }
