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

// BlockRepositoryI defines the interface for block data access operations
type BlockRepositoryI interface {
	GetBlockHeightLatest() (*int64, error)
	GetBlockTimestamp(latestBlockHeight int64) ([]time.Time, error)
	GetLatestBlock() (*db.Block, error)
}

// NFTRepositoryI defines the interface for NFT data access operations
type NFTRepositoryI interface {
	// GetCollections retrieves NFT collections with pagination and search
	GetCollections(pagination dto.PaginationQuery, search string) ([]db.Collection, int64, error)
	GetNFTByNFTAddress(collectionAddress string, nftAddress string) (*dto.NFTByAddressModel, error)
	GetNFTsByAccountAddress(pagination dto.PaginationQuery, accountAddress string, collectionAddress string, search string) ([]dto.NFTByAddressModel, int64, error)
	GetNFTsByCollectionAddress(pagination dto.PaginationQuery, collectionAddress string, search string) ([]dto.NFTByAddressModel, int64, error)
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

// ValidatorRepositoryI defines the interface for validator data access operations
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

// ModuleRepository defines the interface for module data access operations
type ModuleRepository interface {
	// GetModules retrieves modules with pagination
	GetModules(pagination dto.PaginationQuery) ([]dto.ModuleResponse, int64, error)

	// GetModuleById retrieves a module by id
	GetModuleById(vmAddress string, name string) (*dto.ModuleResponse, error)

	// GetModuleHistories retrieves module histories with pagination
	GetModuleHistories(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleHistoryResponse, int64, error)

	// GetModulePublishInfo retrieves a module publish info
	GetModulePublishInfo(vmAddress string, name string) ([]dto.ModulePublishInfoModel, error)

	// GetModuleProposals retrieves a module proposal
	GetModuleProposals(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleProposalModel, int64, error)

	// GetModuleTxs retrieves a module txs
	GetModuleTransactions(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleTxResponse, int64, error)

	// GetModuleStats retrieves a module stats
	GetModuleStats(vmAddress string, name string) (*dto.ModuleStatsResponse, error)
}
