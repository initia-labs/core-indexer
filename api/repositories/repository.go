package repositories

import (
	"time"

	"gocloud.dev/blob"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/pkg/db"
)

type Repositories struct {
	BlockRepository     *BlockRepository
	NftRepository       *NftRepository
	ProposalRepository  *ProposalRepository
	TxRepository        *TxRepository
	ValidatorRepository *ValidatorRepository
}

func SetupRepositories(dbClient *gorm.DB, bucket *blob.Bucket) *Repositories {
	return &Repositories{
		BlockRepository:     NewBlockRepository(dbClient),
		NftRepository:       NewNFTRepository(dbClient),
		ProposalRepository:  NewProposalRepository(dbClient),
		TxRepository:        NewTxRepository(dbClient, bucket),
		ValidatorRepository: NewValidatorRepository(dbClient),
	}
}

// NFTRepositoryI defines the interface for NFT data access operations
type NFTRepositoryI interface {
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

// ProposalRepositoryI defines the interface for proposal data access operations
type ProposalRepositoryI interface {
	GetProposals() ([]db.Proposal, error)
	GetProposalVotesByValidator(operatorAddr string) ([]db.ProposalVote, error)
}

// TxRepositoryI defines the interface for transaction data access operations
type TxRepositoryI interface {
	GetTxByHash(hash string) (*dto.TxByHashResponse, error)
	GetTxCount() (*int64, error)
	GetTxs(pagination dto.PaginationQuery) ([]dto.TxModel, int64, error)
}

type BlockRepositoryI interface {
	GetBlockHeightLatest() (*int64, error)
	GetBlockTimestamp(latestBlockHeight int64) ([]time.Time, error)
	GetBlocks(pagination dto.PaginationQuery) ([]dto.BlockModel, int64, error)
	GetBlockInfo(height int64) (*dto.BlockInfoModel, error)
	GetBlockTxs(pagination dto.PaginationQuery, height int64) ([]dto.BlockTxResponse, int64, error)
	GetLatestBlock() (*db.Block, error)
}

type AccountRepositoryI interface {
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

type ValidatorRepositoryI interface {
	GetValidators(pagination dto.PaginationQuery, isActive bool, sortBy, search string) ([]dto.ValidatorWithVoteCountModel, int64, error)
	GetValidatorsByPower(pagination *dto.PaginationQuery, onlyActive bool) ([]db.Validator, error)
	GetValidatorRow(operatorAddr string) (*db.Validator, error)
	GetValidatorBlockVoteByBlockLimit(minHeight, maxHeight int64) ([]dto.ValidatorBlockVoteModel, error)
	GetValidatorCommitSignatures(operatorAddr string, minHeight, maxHeight int64) ([]dto.ValidatorBlockVoteModel, error)
	GetValidatorSlashEvents(operatorAddr string, minTimestamp time.Time) ([]dto.ValidatorUptimeEventModel, error)
	GetValidatorUptimeInfo(operatorAddr string) (*dto.ValidatorWithVoteCountModel, error)
	GetValidatorBondedTokenChanges(pagination dto.PaginationQuery, operatorAddr string) ([]db.ValidatorBondedTokenChange, int64, error)
	GetValidatorProposedBlocks(pagination dto.PaginationQuery, operatorAddr string) ([]dto.ValidatorProposedBlockModel, int64, error)
	GetValidatorHistoricalPowers(operatorAddr string) ([]dto.ValidatorHistoricalPowerModel, int64, error)
}
