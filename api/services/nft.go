package services

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
)

// NFTService defines the interface for NFT-related operations
type NFTService interface {
	GetCollections(pagination dto.PaginationQuery, search string) (*dto.NFTCollectionsResponse, error)
}

// nftService implements the NFTService interface
type nftService struct {
	repo repositories.NFTRepository
}

// NewNFTService creates a new instance of NFTService
func NewNFTService(repo repositories.NFTRepository) NFTService {
	return &nftService{
		repo: repo,
	}
}

// GetCollections retrieves NFT collections with pagination and search
func (s *nftService) GetCollections(pagination dto.PaginationQuery, search string) (*dto.NFTCollectionsResponse, error) {
	collections, total, err := s.repo.GetCollections(pagination, search)
	if err != nil {
		return nil, err
	}

	response := &dto.NFTCollectionsResponse{
		Collections: collections,
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   total,
		},
	}

	// If we have items and count_total is true, we can calculate the next key
	if len(collections) > 0 && pagination.CountTotal {
		// TODO: Implement next key calculation based on the last item
		// This would typically be a base64 encoded cursor to the next page
	}

	return response, nil
}
