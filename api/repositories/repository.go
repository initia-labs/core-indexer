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
	ModuleRepository    *ModuleRepository
	NftRepository       *NftRepository
	ProposalRepository  *ProposalRepository
	TxRepository        *TxRepository
	ValidatorRepository *ValidatorRepository
	AccountRepository   *AccountRepository
}

func SetupRepositories(dbClient *gorm.DB, bucket *blob.Bucket) *Repositories {
	return &Repositories{
		BlockRepository:     NewBlockRepository(dbClient),
		ModuleRepository:    NewModuleRepository(dbClient),
		NftRepository:       NewNftRepository(dbClient),
		ProposalRepository:  NewProposalRepository(dbClient),
		TxRepository:        NewTxRepository(dbClient, bucket),
		ValidatorRepository: NewValidatorRepository(dbClient),
		AccountRepository:   NewAccountRepository(dbClient),
	}
}

// ModuleRepository defines the interface for module data access operations
type ModuleRepositoryI interface {
	GetModules(pagination dto.PaginationQuery) ([]dto.ModuleResponse, int64, error)
	GetModuleById(vmAddress string, name string) (*dto.ModuleResponse, error)
	GetModuleHistories(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleHistoryResponse, int64, error)
	GetModulePublishInfo(vmAddress string, name string) ([]dto.ModulePublishInfoModel, error)
	GetModuleProposals(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleProposalModel, int64, error)
	GetModuleTransactions(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleTxResponse, int64, error)
	GetModuleStats(vmAddress string, name string) (*dto.ModuleStatsResponse, error)
}

// NftRepositoryI defines the interface for Nft data access operations
type NftRepositoryI interface {
	// GetCollections retrieves Nft collections with pagination and search
	GetCollections(pagination dto.PaginationQuery, search string) ([]db.Collection, int64, error)
	GetCollectionsByAccountAddress(accountAddress string) ([]dto.CollectionByAccountAddressModel, error)
	GetCollectionsByCollectionAddress(collectionAddress string) (*db.Collection, error)
	GetCollectionActivities(pagination dto.PaginationQuery, collectionAddress string, search string) ([]dto.CollectionActivityModel, int64, error)
	GetCollectionCreator(collectionAddress string) (*dto.CollectionCreatorModel, error)
	GetCollectionMutateEvents(pagination dto.PaginationQuery, collectionAddress string) ([]dto.MutateEventModel, int64, error)
	GetNftByNftAddress(collectionAddress string, nftAddress string) (*dto.NftByAddressModel, error)
	GetNftsByAccountAddress(pagination dto.PaginationQuery, accountAddress string, collectionAddress string, search string) ([]dto.NftByAddressModel, int64, error)
	GetNftsByCollectionAddress(pagination dto.PaginationQuery, collectionAddress string, search string) ([]dto.NftByAddressModel, int64, error)
	GetNftMintInfo(nftAddress string) (*dto.NftMintInfoModel, error)
	GetNftMutateEvents(pagination dto.PaginationQuery, nftAddress string) ([]dto.MutateEventModel, int64, error)
	GetNftTxs(pagination dto.PaginationQuery, nftAddress string) ([]dto.NftTxModel, int64, error)
}

// ProposalRepositoryI defines the interface for proposal data access operations
type ProposalRepositoryI interface {
	GetProposals() ([]db.Proposal, error)
	GetProposalVotesByValidator(operatorAddr string) ([]db.ProposalVote, error)
	SearchProposals(limit, offset int64, proposer, search string, statuses, types []string) ([]dto.ProposalSummary, int64, error)
	GetAllProposalTypes() (*dto.ProposalsTypesResponse, error)
	GetProposalInfo(id int) (*dto.ProposalInfo, error)
	GetProposalVotes(id int, limit, offset int64, search, answer string) ([]dto.ProposalVote, int64, error)
	GetProposalValidatorVotes(id int) ([]dto.ProposalVote, error)
	GetProposalAnswerCounts(id int) (*dto.ProposalAnswerCountsResponse, error)
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
	GetBlockTxs(pagination dto.PaginationQuery, height int64) ([]dto.BlockTxModel, int64, error)
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
