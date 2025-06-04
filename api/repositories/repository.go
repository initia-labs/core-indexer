package repositories

import (
	"time"

	"github.com/initia-labs/core-indexer/api/dto"
)

// NFTRepository defines the interface for NFT data access operations
type NFTRepository interface {
	// GetCollections retrieves NFT collections with pagination and search
	GetCollections(pagination dto.PaginationQuery, search string) ([]dto.NFTCollection, int64, error)
	GetNFTByNFTAddress(collectionAddress string, nftAddress string) (*dto.NFTByAddress, error)
	GetNFTsByAccountAddress(pagination dto.PaginationQuery, accountAddress string, collectionAddress *string, search *string) ([]dto.NFTByAddress, int64, error)
	GetNFTsByCollectionAddress(pagination dto.PaginationQuery, collectionAddress string, search *string) ([]dto.NFTByAddress, int64, error)
	GetNFTMintInfo(nftAddress string) (*dto.NFTMintInfo, error)
	GetNFTMutateEvents(pagination dto.PaginationQuery, nftAddress string) ([]dto.NFTMutateEventResponse, int64, error)
	GetNFTTxs(pagination dto.PaginationQuery, nftAddress string) ([]dto.NFTTx, int64, error)
}

// TxRepository defines the interface for transaction data access operations
type TxRepository interface {
	GetTxByHash(hash string) (*dto.RestTxByHashResponse, error)
	GetTxCount() (*dto.RestTxCountResponse, error)
	GetTxs(pagination dto.PaginationQuery) ([]dto.TxResponse, int64, error)
}

type BlockRepository interface {
	GetBlockHeightLatest() (*dto.RestBlockHeightLatestResponse, error)
	GetBlockTimestamp(latestBlockHeight *int64) ([]time.Time, error)
}
