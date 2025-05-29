package repositories

import (
	"github.com/initia-labs/core-indexer/api/dto"
)

// NFTRepository defines the interface for NFT data access operations
type NFTRepository interface {
	// GetCollections retrieves NFT collections with pagination and search
	GetCollections(pagination dto.PaginationQuery, search string) ([]dto.NFTCollection, int64, error)
}
