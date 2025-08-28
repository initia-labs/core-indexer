package services_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories/mocks"
	"github.com/initia-labs/core-indexer/api/services"
	"github.com/initia-labs/core-indexer/pkg/db"
)

const (
	CollectionAddress = "0x1234567890abcdef1234567890abcdef12345678"
	NftAddress        = "0xabcdef1234567890abcdef1234567890abcdef12"
)

func TestNftService_GetCollections(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	// Test data
	pagination := dto.PaginationQuery{
		Limit:      10,
		Offset:     0,
		CountTotal: true,
	}
	search := "test"

	expectedCollections := []db.Collection{
		{
			ID:          CollectionAddress,
			Name:        "Test Collection 1",
			URI:         "https://example.com/collection1",
			Description: "Test collection description 1",
			Creator:     AccountAddress,
		},
		{
			ID:          "0x2345678901bcdef2345678901bcdef23456789",
			Name:        "Test Collection 2",
			URI:         "https://example.com/collection2",
			Description: "Test collection description 2",
			Creator:     "init1testaccount2",
		},
	}

	// Set up mock expectations
	mockRepo.On("GetCollections", pagination, search).Return(expectedCollections, int64(2), nil)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetCollections(pagination, search)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Collections, 2)
	assert.Equal(t, int64(2), result.Pagination.Total)

	// Test first collection
	assert.Equal(t, CollectionAddress, result.Collections[0].ObjectAddr)
	assert.Equal(t, "Test Collection 1", result.Collections[0].Collection.Name)
	assert.Equal(t, "https://example.com/collection1", result.Collections[0].Collection.URI)
	assert.Equal(t, "Test collection description 1", result.Collections[0].Collection.Description)
	assert.Equal(t, AccountAddress, result.Collections[0].Collection.Creator)
	assert.Nil(t, result.Collections[0].Collection.Nft)

	// Test second collection
	assert.Equal(t, "0x2345678901bcdef2345678901bcdef23456789", result.Collections[1].ObjectAddr)
	assert.Equal(t, "Test Collection 2", result.Collections[1].Collection.Name)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetCollections_Error(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	pagination := dto.PaginationQuery{
		Limit:      10,
		Offset:     0,
		CountTotal: true,
	}
	search := "test"

	// Set up mock expectations for error case
	mockRepo.On("GetCollections", pagination, search).Return(nil, int64(0), assert.AnError)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetCollections(pagination, search)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, assert.AnError, err)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetCollectionsByAccountAddress(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	expectedCollections := []dto.CollectionByAccountAddressModel{
		{
			ID:          CollectionAddress,
			Name:        "Test Collection 1",
			URI:         "https://example.com/collection1",
			Description: "Test collection description 1",
			Creator:     AccountAddress,
			Count:       5,
		},
		{
			ID:          "0x2345678901bcdef2345678901bcdef23456789",
			Name:        "Test Collection 2",
			URI:         "https://example.com/collection2",
			Description: "Test collection description 2",
			Creator:     AccountAddress,
			Count:       3,
		},
	}

	// Set up mock expectations
	mockRepo.On("GetCollectionsByAccountAddress", AccountAddress).Return(expectedCollections, nil)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetCollectionsByAccountAddress(AccountAddress)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Collections, 2)
	assert.Equal(t, int64(2), result.Pagination.Total)

	// Test first collection
	assert.Equal(t, CollectionAddress, result.Collections[0].ObjectAddr)
	assert.Equal(t, "Test Collection 1", result.Collections[0].Collection.Name)
	assert.Equal(t, AccountAddress, result.Collections[0].Collection.Creator)
	assert.NotNil(t, result.Collections[0].Collection.Nft)
	assert.Equal(t, int64(5), result.Collections[0].Collection.Nft.Length)

	// Test second collection
	assert.Equal(t, "0x2345678901bcdef2345678901bcdef23456789", result.Collections[1].ObjectAddr)
	assert.Equal(t, int64(3), result.Collections[1].Collection.Nft.Length)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetCollectionsByAccountAddress_Error(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	// Set up mock expectations for error case
	mockRepo.On("GetCollectionsByAccountAddress", AccountAddress).Return(nil, assert.AnError)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetCollectionsByAccountAddress(AccountAddress)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, assert.AnError, err)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetCollectionsByCollectionAddress(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	expectedCollection := &db.Collection{
		ID:          CollectionAddress,
		Name:        "Test Collection",
		URI:         "https://example.com/collection",
		Description: "Test collection description",
		Creator:     AccountAddress,
	}

	// Set up mock expectations
	mockRepo.On("GetCollectionsByCollectionAddress", CollectionAddress).Return(expectedCollection, nil)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetCollectionsByCollectionAddress(CollectionAddress)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, CollectionAddress, result.ObjectAddr)
	assert.Equal(t, "Test Collection", result.Collection.Name)
	assert.Equal(t, "https://example.com/collection", result.Collection.URI)
	assert.Equal(t, "Test collection description", result.Collection.Description)
	assert.Equal(t, AccountAddress, result.Collection.Creator)
	assert.Nil(t, result.Collection.Nft)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetCollectionsByCollectionAddress_Error(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	// Set up mock expectations for error case
	mockRepo.On("GetCollectionsByCollectionAddress", CollectionAddress).Return(nil, assert.AnError)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetCollectionsByCollectionAddress(CollectionAddress)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, assert.AnError, err)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetCollectionActivities(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	pagination := dto.PaginationQuery{
		Limit:      10,
		Offset:     0,
		CountTotal: true,
	}
	search := "test"

	expectedActivities := []dto.CollectionActivityModel{
		{
			Hash:               "0x1234567890abcdef",
			Timestamp:          "2024-01-01T10:00:00Z",
			IsNftBurn:          false,
			IsNftMint:          true,
			IsNftTransfer:      false,
			NftID:              NftAddress,
			TokenID:            "token1",
			IsCollectionCreate: false,
		},
		{
			Hash:               "0xabcdef1234567890",
			Timestamp:          "2024-01-01T11:00:00Z",
			IsNftBurn:          false,
			IsNftMint:          false,
			IsNftTransfer:      true,
			NftID:              NftAddress,
			TokenID:            "token1",
			IsCollectionCreate: false,
		},
	}

	// Set up mock expectations
	mockRepo.On("GetCollectionActivities", pagination, CollectionAddress, search).Return(expectedActivities, int64(2), nil)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetCollectionActivities(pagination, CollectionAddress, search)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.CollectionActivities, 2)
	assert.Equal(t, int64(2), result.Pagination.Total)

	// Test first activity
	assert.Equal(t, "307831323334353637383930616263646566", result.CollectionActivities[0].Hash)
	assert.Equal(t, "2024-01-01T10:00:00Z", result.CollectionActivities[0].Timestamp)
	assert.False(t, result.CollectionActivities[0].IsNftBurn)
	assert.True(t, result.CollectionActivities[0].IsNftMint)
	assert.False(t, result.CollectionActivities[0].IsNftTransfer)
	assert.Equal(t, NftAddress, result.CollectionActivities[0].NftID)
	assert.Equal(t, "token1", result.CollectionActivities[0].TokenID)
	assert.False(t, result.CollectionActivities[0].IsCollectionCreate)

	// Test second activity
	assert.Equal(t, "307861626364656631323334353637383930", result.CollectionActivities[1].Hash)
	assert.True(t, result.CollectionActivities[1].IsNftTransfer)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetCollectionCreator(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	expectedCreator := &dto.CollectionCreatorModel{
		Height:    1000,
		Timestamp: "2024-01-01T10:00:00Z",
		Creator:   AccountAddress,
		Hash:      "0x1234567890abcdef",
	}

	// Set up mock expectations
	mockRepo.On("GetCollectionCreator", CollectionAddress).Return(expectedCreator, nil)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetCollectionCreator(CollectionAddress)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1000), result.Creator.Height)
	assert.Equal(t, "2024-01-01T10:00:00Z", result.Creator.Timestamp)
	assert.Equal(t, AccountAddress, result.Creator.Creator)
	assert.Equal(t, "307831323334353637383930616263646566", result.Creator.Hash)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetCollectionMutateEvents(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	pagination := dto.PaginationQuery{
		Limit:      10,
		Offset:     0,
		CountTotal: true,
	}

	expectedEvents := []dto.MutateEventModel{
		{
			MutatedFieldName: "name",
			NewValue:         "New Collection Name",
			OldValue:         "Old Collection Name",
			Remark:           []byte(`{"reason": "update"}`),
			Timestamp:        "2024-01-01T10:00:00Z",
		},
		{
			MutatedFieldName: "description",
			NewValue:         "New description",
			OldValue:         "Old description",
			Remark:           []byte(`{"reason": "improvement"}`),
			Timestamp:        "2024-01-01T11:00:00Z",
		},
	}

	// Set up mock expectations
	mockRepo.On("GetCollectionMutateEvents", pagination, CollectionAddress).Return(expectedEvents, int64(2), nil)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetCollectionMutateEvents(pagination, CollectionAddress)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.CollectionMutateEvents, 2)
	assert.Equal(t, int64(2), result.Pagination.Total)

	// Test first event
	assert.Equal(t, "name", result.CollectionMutateEvents[0].MutatedFieldName)
	assert.Equal(t, "New Collection Name", result.CollectionMutateEvents[0].NewValue)
	assert.Equal(t, "Old Collection Name", result.CollectionMutateEvents[0].OldValue)
	assert.Equal(t, "2024-01-01T10:00:00Z", result.CollectionMutateEvents[0].Timestamp)

	// Test second event
	assert.Equal(t, "description", result.CollectionMutateEvents[1].MutatedFieldName)
	assert.Equal(t, "New description", result.CollectionMutateEvents[1].NewValue)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetNftByNftAddress(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	expectedNft := &dto.NftByAddressModel{
		TokenID:        "token1",
		URI:            "https://example.com/nft1",
		Description:    "Test NFT description",
		IsBurned:       false,
		Owner:          AccountAddress,
		ID:             NftAddress,
		Collection:     CollectionAddress,
		CollectionName: "Test Collection",
	}

	// Set up mock expectations
	mockRepo.On("GetNftByNftAddress", CollectionAddress, NftAddress).Return(expectedNft, nil)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetNftByNftAddress(CollectionAddress, NftAddress)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, NftAddress, result.ObjectAddr)
	assert.Equal(t, CollectionAddress, result.CollectionAddr)
	assert.Equal(t, "Test Collection", result.CollectionName)
	assert.Equal(t, AccountAddress, result.OwnerAddr)
	assert.Equal(t, CollectionAddress, result.Nft.Collection.Inner)
	assert.Equal(t, "token1", result.Nft.TokenID)
	assert.Equal(t, "https://example.com/nft1", result.Nft.URI)
	assert.Equal(t, "Test NFT description", result.Nft.Description)
	assert.False(t, result.Nft.IsBurned)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetNftByNftAddress_Error(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	// Set up mock expectations for error case
	mockRepo.On("GetNftByNftAddress", CollectionAddress, NftAddress).Return(nil, assert.AnError)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetNftByNftAddress(CollectionAddress, NftAddress)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, assert.AnError, err)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetNftsByAccountAddress(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	pagination := dto.PaginationQuery{
		Limit:      10,
		Offset:     0,
		CountTotal: true,
	}
	collectionAddress := CollectionAddress
	search := "test"

	expectedNfts := []dto.NftByAddressModel{
		{
			TokenID:        "token1",
			URI:            "https://example.com/nft1",
			Description:    "Test NFT 1 description",
			IsBurned:       false,
			Owner:          AccountAddress,
			ID:             NftAddress,
			Collection:     CollectionAddress,
			CollectionName: "Test Collection",
		},
		{
			TokenID:        "token2",
			URI:            "https://example.com/nft2",
			Description:    "Test NFT 2 description",
			IsBurned:       false,
			Owner:          AccountAddress,
			ID:             "0xabcdef1234567890abcdef1234567890abcdef13",
			Collection:     CollectionAddress,
			CollectionName: "Test Collection",
		},
	}

	// Set up mock expectations
	mockRepo.On("GetNftsByAccountAddress", pagination, AccountAddress, collectionAddress, search).Return(expectedNfts, int64(2), nil)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetNftsByAccountAddress(pagination, AccountAddress, collectionAddress, search)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Tokens, 2)
	assert.Equal(t, int64(2), result.Pagination.Total)

	// Test first NFT
	assert.Equal(t, NftAddress, result.Tokens[0].ObjectAddr)
	assert.Equal(t, CollectionAddress, result.Tokens[0].CollectionAddr)
	assert.Equal(t, "Test Collection", result.Tokens[0].CollectionName)
	assert.Equal(t, AccountAddress, result.Tokens[0].OwnerAddr)
	assert.Equal(t, "token1", result.Tokens[0].Nft.TokenID)
	assert.Equal(t, "https://example.com/nft1", result.Tokens[0].Nft.URI)
	assert.Equal(t, "Test NFT 1 description", result.Tokens[0].Nft.Description)
	assert.False(t, result.Tokens[0].Nft.IsBurned)

	// Test second NFT
	assert.Equal(t, "0xabcdef1234567890abcdef1234567890abcdef13", result.Tokens[1].ObjectAddr)
	assert.Equal(t, "token2", result.Tokens[1].Nft.TokenID)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetNftsByCollectionAddress(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	pagination := dto.PaginationQuery{
		Limit:      10,
		Offset:     0,
		CountTotal: true,
	}
	search := "test"

	expectedNfts := []dto.NftByAddressModel{
		{
			TokenID:        "token1",
			URI:            "https://example.com/nft1",
			Description:    "Test NFT 1 description",
			IsBurned:       false,
			Owner:          AccountAddress,
			ID:             NftAddress,
			Collection:     CollectionAddress,
			CollectionName: "Test Collection",
		},
		{
			TokenID:        "token2",
			URI:            "https://example.com/nft2",
			Description:    "Test NFT 2 description",
			IsBurned:       false,
			Owner:          "init1testaccount2",
			ID:             "0xabcdef1234567890abcdef1234567890abcdef13",
			Collection:     CollectionAddress,
			CollectionName: "Test Collection",
		},
	}

	// Set up mock expectations
	mockRepo.On("GetNftsByCollectionAddress", pagination, CollectionAddress, search).Return(expectedNfts, int64(2), nil)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetNftsByCollectionAddress(pagination, CollectionAddress, search)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Tokens, 2)
	assert.Equal(t, int64(2), result.Pagination.Total)

	// Test first NFT
	assert.Equal(t, NftAddress, result.Tokens[0].ObjectAddr)
	assert.Equal(t, CollectionAddress, result.Tokens[0].CollectionAddr)
	assert.Equal(t, "Test Collection", result.Tokens[0].CollectionName)
	assert.Equal(t, AccountAddress, result.Tokens[0].OwnerAddr)
	assert.Equal(t, "token1", result.Tokens[0].Nft.TokenID)

	// Test second NFT
	assert.Equal(t, "0xabcdef1234567890abcdef1234567890abcdef13", result.Tokens[1].ObjectAddr)
	assert.Equal(t, "init1testaccount2", result.Tokens[1].OwnerAddr)
	assert.Equal(t, "token2", result.Tokens[1].Nft.TokenID)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetNftMintInfo(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	expectedMintInfo := &dto.NftMintInfoModel{
		Address:   AccountAddress,
		Hash:      "0x1234567890abcdef",
		Height:    1000,
		Timestamp: "2024-01-01T10:00:00Z",
	}

	// Set up mock expectations
	mockRepo.On("GetNftMintInfo", NftAddress).Return(expectedMintInfo, nil)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetNftMintInfo(NftAddress)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1000), result.Height)
	assert.Equal(t, AccountAddress, result.Minter)
	assert.Equal(t, "307831323334353637383930616263646566", result.TxHash)
	assert.Equal(t, "2024-01-01T10:00:00Z", result.Timestamp)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetNftMintInfo_Error(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	// Set up mock expectations for error case
	mockRepo.On("GetNftMintInfo", NftAddress).Return(nil, assert.AnError)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetNftMintInfo(NftAddress)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, assert.AnError, err)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetNftMutateEvents(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	pagination := dto.PaginationQuery{
		Limit:      10,
		Offset:     0,
		CountTotal: true,
	}

	expectedEvents := []dto.MutateEventModel{
		{
			MutatedFieldName: "uri",
			NewValue:         "https://example.com/new-nft1",
			OldValue:         "https://example.com/old-nft1",
			Remark:           []byte(`{"reason": "update"}`),
			Timestamp:        "2024-01-01T10:00:00Z",
		},
		{
			MutatedFieldName: "description",
			NewValue:         "New NFT description",
			OldValue:         "Old NFT description",
			Remark:           []byte(`{"reason": "improvement"}`),
			Timestamp:        "2024-01-01T11:00:00Z",
		},
	}

	// Set up mock expectations
	mockRepo.On("GetNftMutateEvents", pagination, NftAddress).Return(expectedEvents, int64(2), nil)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetNftMutateEvents(pagination, NftAddress)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.NftMutateEvents, 2)
	assert.Equal(t, int64(2), result.Pagination.Total)

	// Test first event
	assert.Equal(t, "uri", result.NftMutateEvents[0].MutatedFieldName)
	assert.Equal(t, "https://example.com/new-nft1", result.NftMutateEvents[0].NewValue)
	assert.Equal(t, "https://example.com/old-nft1", result.NftMutateEvents[0].OldValue)
	assert.Equal(t, "2024-01-01T10:00:00Z", result.NftMutateEvents[0].Timestamp)

	// Test second event
	assert.Equal(t, "description", result.NftMutateEvents[1].MutatedFieldName)
	assert.Equal(t, "New NFT description", result.NftMutateEvents[1].NewValue)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_GetNftTxs(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	pagination := dto.PaginationQuery{
		Limit:      10,
		Offset:     0,
		CountTotal: true,
	}

	expectedTxs := []dto.NftTxModel{
		{
			IsNftBurn:     false,
			IsNftMint:     true,
			IsNftTransfer: false,
			Hash:          "0x1234567890abcdef",
			Height:        1000,
			Timestamp:     "2024-01-01T10:00:00Z",
		},
		{
			IsNftBurn:     false,
			IsNftMint:     false,
			IsNftTransfer: true,
			Hash:          "0xabcdef1234567890",
			Height:        1001,
			Timestamp:     "2024-01-01T11:00:00Z",
		},
		{
			IsNftBurn:     true,
			IsNftMint:     false,
			IsNftTransfer: false,
			Hash:          "0x9876543210fedcba",
			Height:        1002,
			Timestamp:     "2024-01-01T12:00:00Z",
		},
	}

	// Set up mock expectations
	mockRepo.On("GetNftTxs", pagination, NftAddress).Return(expectedTxs, int64(3), nil)

	// Create service with mock repository
	service := services.NewNftService(mockRepo)

	// Call the method
	result, err := service.GetNftTxs(pagination, NftAddress)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.NftTxs, 3)
	assert.Equal(t, int64(3), result.Pagination.Total)

	// Test first transaction (mint)
	assert.False(t, result.NftTxs[0].IsNftBurn)
	assert.True(t, result.NftTxs[0].IsNftMint)
	assert.False(t, result.NftTxs[0].IsNftTransfer)
	assert.Equal(t, "307831323334353637383930616263646566", result.NftTxs[0].TxHash)
	assert.Equal(t, "2024-01-01T10:00:00Z", result.NftTxs[0].Timestamp)

	// Test second transaction (transfer)
	assert.False(t, result.NftTxs[1].IsNftBurn)
	assert.False(t, result.NftTxs[1].IsNftMint)
	assert.True(t, result.NftTxs[1].IsNftTransfer)
	assert.Equal(t, "307861626364656631323334353637383930", result.NftTxs[1].TxHash)

	// Test third transaction (burn)
	assert.True(t, result.NftTxs[2].IsNftBurn)
	assert.False(t, result.NftTxs[2].IsNftMint)
	assert.False(t, result.NftTxs[2].IsNftTransfer)
	assert.Equal(t, "307839383736353433323130666564636261", result.NftTxs[2].TxHash)

	// Verify mock was called as expected
	mockRepo.AssertExpectations(t)
}

func TestNftService_PaginationScenarios(t *testing.T) {
	testCases := []struct {
		name              string
		pagination        dto.PaginationQuery
		expectedTotal     int64
		expectedLength    int
		expectedFirstName string
		expectedLastName  string
	}{
		{
			name: "Default pagination",
			pagination: dto.PaginationQuery{
				Limit:      10,
				Offset:     0,
				CountTotal: true,
			},
			expectedTotal:     15,
			expectedLength:    10,
			expectedFirstName: "Collection 1",
			expectedLastName:  "Collection 10",
		},
		{
			name: "Small limit",
			pagination: dto.PaginationQuery{
				Limit:      3,
				Offset:     0,
				CountTotal: true,
			},
			expectedTotal:     15,
			expectedLength:    3,
			expectedFirstName: "Collection 1",
			expectedLastName:  "Collection 3",
		},
		{
			name: "With offset",
			pagination: dto.PaginationQuery{
				Limit:      5,
				Offset:     5,
				CountTotal: true,
			},
			expectedTotal:     15,
			expectedLength:    5,
			expectedFirstName: "Collection 6",
			expectedLastName:  "Collection 10",
		},
		{
			name: "Without count total",
			pagination: dto.PaginationQuery{
				Limit:  10,
				Offset: 0,
			},
			expectedTotal:     15,
			expectedLength:    10,
			expectedFirstName: "Collection 1",
			expectedLastName:  "Collection 10",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock repository for each test case
			mockRepo := mocks.NewMockNftRepository()

			// Create test data
			allCollections := make([]db.Collection, 15)
			for i := 0; i < 15; i++ {
				allCollections[i] = db.Collection{
					ID:          fmt.Sprintf("0x%040d", i+1),
					Name:        fmt.Sprintf("Collection %d", i+1),
					URI:         fmt.Sprintf("https://example.com/collection%d", i+1),
					Description: fmt.Sprintf("Description for collection %d", i+1),
					Creator:     AccountAddress,
				}
			}

			// Calculate expected subset
			start := tc.pagination.Offset
			end := start + tc.pagination.Limit
			if end > len(allCollections) {
				end = len(allCollections)
			}
			expectedCollections := allCollections[start:end]

			// Set up mock expectations
			mockRepo.On("GetCollections", tc.pagination, "").Return(expectedCollections, tc.expectedTotal, nil)

			// Create service with mock repository
			service := services.NewNftService(mockRepo)

			// Call the method
			result, err := service.GetCollections(tc.pagination, "")

			// Assertions
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tc.expectedTotal, result.Pagination.Total)
			assert.Len(t, result.Collections, tc.expectedLength)

			if tc.expectedLength > 0 {
				assert.Equal(t, tc.expectedFirstName, result.Collections[0].Collection.Name)
				assert.Equal(t, tc.expectedLastName, result.Collections[tc.expectedLength-1].Collection.Name)
			}

			// Verify mock was called as expected
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestNftService_EmptyResults(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	t.Run("Empty collections", func(t *testing.T) {
		pagination := dto.PaginationQuery{
			Limit:      10,
			Offset:     0,
			CountTotal: true,
		}

		// Set up mock expectations for empty result
		mockRepo.On("GetCollections", pagination, "").Return([]db.Collection{}, int64(0), nil)

		// Create service with mock repository
		service := services.NewNftService(mockRepo)

		// Call the method
		result, err := service.GetCollections(pagination, "")

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(0), result.Pagination.Total)
		assert.Len(t, result.Collections, 0)
		assert.NotNil(t, result.Collections) // Should be empty slice, not nil

		// Verify mock was called as expected
		mockRepo.AssertExpectations(t)
	})

	t.Run("Empty NFTs by account", func(t *testing.T) {
		pagination := dto.PaginationQuery{
			Limit:      10,
			Offset:     0,
			CountTotal: true,
		}

		// Set up mock expectations for empty result
		mockRepo.On("GetNftsByAccountAddress", pagination, AccountAddress, "", "").Return([]dto.NftByAddressModel{}, int64(0), nil)

		// Create service with mock repository
		service := services.NewNftService(mockRepo)

		// Call the method
		result, err := service.GetNftsByAccountAddress(pagination, AccountAddress, "", "")

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(0), result.Pagination.Total)
		assert.Len(t, result.Tokens, 0)
		assert.NotNil(t, result.Tokens) // Should be empty slice, not nil

		// Verify mock was called as expected
		mockRepo.AssertExpectations(t)
	})
}

func TestNftService_ErrorHandling(t *testing.T) {
	// Create mock repository
	mockRepo := mocks.NewMockNftRepository()

	t.Run("GetCollections error", func(t *testing.T) {
		pagination := dto.PaginationQuery{
			Limit:      10,
			Offset:     0,
			CountTotal: true,
		}

		// Set up mock expectations for error
		mockRepo.On("GetCollections", pagination, "").Return(nil, int64(0), assert.AnError)

		// Create service with mock repository
		service := services.NewNftService(mockRepo)

		// Call the method
		result, err := service.GetCollections(pagination, "")

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, assert.AnError, err)

		// Verify mock was called as expected
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetNftByNftAddress error", func(t *testing.T) {
		// Set up mock expectations for error
		mockRepo.On("GetNftByNftAddress", CollectionAddress, NftAddress).Return(nil, assert.AnError)

		// Create service with mock repository
		service := services.NewNftService(mockRepo)

		// Call the method
		result, err := service.GetNftByNftAddress(CollectionAddress, NftAddress)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, assert.AnError, err)

		// Verify mock was called as expected
		mockRepo.AssertExpectations(t)
	})
}
