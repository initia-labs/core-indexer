package repositories

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/pkg/db"
)

// NFTRepository defines the interface for NFT data access operations
type NFTRepository interface {
	// GetCollections retrieves NFT collections with pagination and search
	GetCollections(pagination dto.PaginationQuery, search string) ([]db.Collection, int64, error)
}

// TxRepository defines the interface for transaction data access operations
type TxRepository interface {
	// GetTxByHash retrieves a transaction by hash
	GetTxByHash(hash string) (*dto.RestTxResponse, error)
}
