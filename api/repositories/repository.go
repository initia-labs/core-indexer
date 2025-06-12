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
	GetCollectionsByAccountAddress(accountAddress string) ([]dto.CollectionByAccountAddressModel, error)
	GetCollectionsByCollectionAddress(collectionAddress string) (*db.Collection, error)
	GetCollectionActivities(pagination dto.PaginationQuery, collectionAddress string, search string) ([]dto.CollectionActivityModel, int64, error)
	GetCollectionCreator(collectionAddress string) (*dto.CollectionCreatorModel, error)
	GetCollectionMutateEvents(pagination dto.PaginationQuery, collectionAddress string) ([]dto.CollectionMutateEventResponse, int64, error)
	GetNFTByNFTAddress(collectionAddress string, nftAddress string) (*dto.NFTByAddressModel, error)
	GetNFTsByAccountAddress(pagination dto.PaginationQuery, accountAddress string, collectionAddress string, search string) ([]dto.NFTByAddressModel, int64, error)
	GetNFTsByCollectionAddress(pagination dto.PaginationQuery, collectionAddress string, search string) ([]dto.NFTByAddressModel, int64, error)
	GetNFTMintInfo(nftAddress string) (*dto.NFTMintInfoModel, error)
	GetNFTMutateEvents(pagination dto.PaginationQuery, nftAddress string) ([]dto.NFTMutateEventResponse, int64, error)
	GetNFTTxs(pagination dto.PaginationQuery, nftAddress string) ([]dto.NFTTx, int64, error)
}

// TxRepository defines the interface for transaction data access operations
type TxRepository interface {
	GetTxByHash(hash string) (*dto.TxByHashResponse, error)
	GetTxCount() (*int64, error)
	GetTxs(pagination dto.PaginationQuery) ([]dto.TxModel, int64, error)
}

type BlockRepository interface {
	GetBlockHeightLatest() (*int64, error)
	GetBlockTimestamp(latestBlockHeight int64) ([]time.Time, error)
	GetBlocks(pagination dto.PaginationQuery) ([]dto.BlockModel, int64, error)
	GetBlockInfo(height int64) (*dto.BlockInfoModel, error)
	GetBlockTxs(pagination dto.PaginationQuery, height int64) ([]dto.BlockTxResponse, int64, error)
}

type AccountRepository interface {
	GetAccountByAccountAddress(accountAddress string) (*db.Account, error)
	GetAccountProposals(pagination dto.PaginationQuery, accountAddress string) ([]db.Proposal, int64, error)
	GetAccountTxs(
		pagination dto.PaginationQuery,
		accountAddress string,
		search string,
		isSend bool,
		isIbc bool,
		isOpinit bool,
		isMovePublish bool,
		isMoveUpgrade bool,
		isMoveExecute bool,
		isMoveScript bool,
		isSigner *bool,
	) ([]dto.AccountTxModel, int64, error)
}
