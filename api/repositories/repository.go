package repositories

import (
	"context"
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

func SetupRepositories(dbClient *gorm.DB, buckets []*blob.Bucket, countQueryTimeout time.Duration) *Repositories {
	return &Repositories{
		BlockRepository:     NewBlockRepository(dbClient, countQueryTimeout),
		ModuleRepository:    NewModuleRepository(dbClient, countQueryTimeout),
		NftRepository:       NewNftRepository(dbClient, countQueryTimeout),
		ProposalRepository:  NewProposalRepository(dbClient, countQueryTimeout),
		TxRepository:        NewTxRepository(dbClient, buckets, countQueryTimeout),
		ValidatorRepository: NewValidatorRepository(dbClient, countQueryTimeout),
		AccountRepository:   NewAccountRepository(dbClient, countQueryTimeout),
	}
}

// ModuleRepository defines the interface for module data access operations
type ModuleRepositoryI interface {
	GetModules(ctx context.Context, pagination dto.PaginationQuery) ([]dto.ModuleResponse, int64, error)
	GetModuleById(ctx context.Context, vmAddress string, name string) (*dto.ModuleResponse, error)
	GetModuleHistories(ctx context.Context, pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleHistoryResponse, int64, error)
	GetModulePublishInfo(ctx context.Context, vmAddress string, name string) ([]dto.ModulePublishInfoModel, error)
	GetModuleProposals(ctx context.Context, pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleProposalModel, int64, error)
	GetModuleTransactions(ctx context.Context, pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleTxResponse, int64, error)
	GetModuleStats(ctx context.Context, vmAddress string, name string) (*dto.ModuleStatsResponse, error)
}

// NftRepositoryI defines the interface for Nft data access operations
type NftRepositoryI interface {
	// GetCollections retrieves Nft collections with pagination and search
	GetCollections(ctx context.Context, pagination dto.PaginationQuery, search string) ([]db.Collection, int64, error)
	GetCollectionsByAccountAddress(ctx context.Context, accountAddress string) ([]dto.CollectionByAccountAddressModel, error)
	GetCollectionsByCollectionAddress(ctx context.Context, collectionAddress string) (*db.Collection, error)
	GetCollectionActivities(ctx context.Context, pagination dto.PaginationQuery, collectionAddress string, search string) ([]dto.CollectionActivityModel, int64, error)
	GetCollectionCreator(ctx context.Context, collectionAddress string) (*dto.CollectionCreatorModel, error)
	GetCollectionMutateEvents(ctx context.Context, pagination dto.PaginationQuery, collectionAddress string) ([]dto.MutateEventModel, int64, error)
	GetNftByNftAddress(ctx context.Context, collectionAddress string, nftAddress string) (*dto.NftByAddressModel, error)
	GetNftsByAccountAddress(ctx context.Context, pagination dto.PaginationQuery, accountAddress string, collectionAddress string, search string) ([]dto.NftByAddressModel, int64, error)
	GetNftsByCollectionAddress(ctx context.Context, pagination dto.PaginationQuery, collectionAddress string, search string) ([]dto.NftByAddressModel, int64, error)
	GetNftMintInfo(ctx context.Context, nftAddress string) (*dto.NftMintInfoModel, error)
	GetNftMutateEvents(ctx context.Context, pagination dto.PaginationQuery, nftAddress string) ([]dto.MutateEventModel, int64, error)
	GetNftTxs(ctx context.Context, pagination dto.PaginationQuery, nftAddress string) ([]dto.NftTxModel, int64, error)
}

// ProposalRepositoryI defines the interface for proposal data access operations
type ProposalRepositoryI interface {
	GetProposals(ctx context.Context, pagination *dto.PaginationQuery) ([]db.Proposal, error)
	GetProposalVotesByValidator(ctx context.Context, operatorAddr string) ([]db.ProposalVote, error)
	SearchProposals(ctx context.Context, pagination dto.PaginationQuery, proposer, search string, statuses, types []string) ([]dto.ProposalSummary, int64, error)
	GetAllProposalTypes(ctx context.Context) (*dto.ProposalsTypesResponse, error)
	GetProposalInfo(ctx context.Context, id int) (*dto.ProposalInfo, error)
	GetProposalVotes(ctx context.Context, id int, limit, offset int64, search, answer string) ([]dto.ProposalVote, int64, error)
	GetProposalValidatorVotes(ctx context.Context, id int) ([]dto.ProposalVote, error)
	GetProposalAnswerCounts(ctx context.Context, id int) (*dto.ProposalAnswerCountsResponse, error)
}

// TxRepositoryI defines the interface for transaction data access operations
type TxRepositoryI interface {
	GetTxByHash(ctx context.Context, hash string) (*dto.TxByHashResponse, error)
	GetTxCount(ctx context.Context) (*int64, error)
	GetTxs(ctx context.Context, pagination *dto.PaginationQuery) ([]dto.TxModel, int64, error)
}

type BlockRepositoryI interface {
	GetBlockHeightLatest(ctx context.Context) (*int64, error)
	GetBlockHeightInformativeLatest(ctx context.Context) (*int64, error)
	GetBlockTimestamp(ctx context.Context, latestBlockHeight int64) ([]time.Time, error)
	GetBlocks(ctx context.Context, pagination dto.PaginationQuery) ([]dto.BlockModel, int64, error)
	GetBlockInfo(ctx context.Context, height int64) (*dto.BlockInfoModel, error)
	GetBlockTxs(ctx context.Context, pagination dto.PaginationQuery, height int64) ([]dto.BlockTxModel, int64, error)
	GetLatestBlock(ctx context.Context) (*db.Block, error)
}

type AccountRepositoryI interface {
	GetAccountByAccountAddress(ctx context.Context, accountAddress string) (*db.Account, error)
	GetAccountProposals(ctx context.Context, pagination dto.PaginationQuery, accountAddress string) ([]db.Proposal, int64, error)
	GetAccountTxs(
		ctx context.Context,
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
	GetValidators(ctx context.Context, pagination dto.PaginationQuery, isActive bool, sortBy, search string) ([]dto.ValidatorWithVoteCountModel, int64, error)
	GetValidatorsByPower(ctx context.Context, pagination *dto.PaginationQuery, onlyActive bool) ([]db.Validator, error)
	GetValidatorRow(ctx context.Context, operatorAddr string) (*db.Validator, error)
	GetValidatorBlockVoteByBlockLimit(ctx context.Context, minHeight, maxHeight int64) ([]dto.ValidatorBlockVoteModel, error)
	GetValidatorCommitSignatures(ctx context.Context, operatorAddr string, minHeight, maxHeight int64) ([]dto.ValidatorBlockVoteModel, error)
	GetValidatorSlashEvents(ctx context.Context, operatorAddr string, minTimestamp time.Time) ([]dto.ValidatorUptimeEventModel, error)
	GetValidatorUptimeInfo(ctx context.Context, operatorAddr string) (*dto.ValidatorWithVoteCountModel, error)
	GetValidatorBondedTokenChanges(ctx context.Context, pagination dto.PaginationQuery, operatorAddr string) ([]db.ValidatorBondedTokenChange, int64, error)
	GetValidatorProposedBlocks(ctx context.Context, pagination dto.PaginationQuery, operatorAddr string) ([]dto.ValidatorProposedBlockModel, int64, error)
	GetValidatorHistoricalPowers(ctx context.Context, operatorAddr string) ([]dto.ValidatorHistoricalPowerModel, int64, error)
}
