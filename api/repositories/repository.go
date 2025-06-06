package repositories

import (
	"time"

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
	GetTxCount() (*int64, error)
}

type BlockRepository interface {
	GetBlockHeightLatest() (*int64, error)
	GetBlockTimestamp(latestBlockHeight int64) ([]time.Time, error)
}

// ModuleRepository defines the interface for module data access operations
type ModuleRepository interface {
	// GetModules retrieves modules with pagination
	GetModules(pagination dto.PaginationQuery) ([]dto.ModuleResponse, int64, error)

	// GetModuleById retrieves a module by id
	GetModuleById(vmAddress string, name string) (*dto.ModuleResponse, error)

	// GetModuleHistories retrieves module histories with pagination
	GetModuleHistories(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleHistory, int64, error)
}
