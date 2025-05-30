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
	// GetTxByHash retrieves a transaction by hash
	GetTxByHash(hash string) (*dto.RestTxResponse, error)
	GetTxCount() (*dto.RestTxCountResponse, error)
}

type BlockRepository interface {
	GetBlockHeightLatest() (*dto.RestBlockHeightLatestResponse, error)
	GetBlockTimestamp(latestBlockHeight *int64) ([]time.Time, error)
}
