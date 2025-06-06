package repositories

import (
	"time"

	"github.com/initia-labs/core-indexer/api/dto"
)

// NFTRepository defines the interface for NFT data access operations
type NFTRepository interface {
	// GetCollections retrieves NFT collections with pagination and search
	GetCollections(pagination dto.PaginationQuery, search string) ([]dto.NFTCollection, int64, error)
}

// TxRepository defines the interface for transaction data access operations
type TxRepository interface {
	GetTxByHash(hash string) (*dto.RestTxByHashResponse, error)
	GetTxCount() (*int64, error)
	GetTxs(pagination dto.PaginationQuery) ([]dto.TxModel, int64, error)
}

type BlockRepository interface {
	GetBlockHeightLatest() (*int32, error)
	GetBlockTimestamp(latestBlockHeight *int32) ([]time.Time, error)
}
