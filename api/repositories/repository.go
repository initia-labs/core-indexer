package repositories

import (
	"gocloud.dev/blob"
	"gorm.io/gorm"
	"time"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/pkg/db"
)

type Repositories struct {
	NftRepository       *NftRepository
	TxRepository        *TxRepository
	BlockRepository     *BlockRepository
	ValidatorRepository *ValidatorRepository
}

func SetupRepositories(dbClient *gorm.DB, bucket *blob.Bucket) *Repositories {
	return &Repositories{
		NftRepository:       NewNFTRepository(dbClient),
		TxRepository:        NewTxRepository(dbClient, bucket),
		BlockRepository:     NewBlockRepository(dbClient),
		ValidatorRepository: NewValidatorRepository(dbClient),
	}
}

// NFTRepositoryI defines the interface for NFT data access operations
type NFTRepositoryI interface {
	// GetCollections retrieves NFT collections with pagination and search
	GetCollections(pagination dto.PaginationQuery, search string) ([]db.Collection, int64, error)
}

// TxRepositoryI defines the interface for transaction data access operations
type TxRepositoryI interface {
	GetTxByHash(hash string) (*dto.TxByHashResponse, error)
	GetTxCount() (*int64, error)
	GetTxs(pagination dto.PaginationQuery) ([]dto.TxModel, int64, error)
}

// BlockRepositoryI defines the interface for block data access operations
type BlockRepositoryI interface {
	GetBlockHeightLatest() (*int64, error)
	GetBlockTimestamp(latestBlockHeight int64) ([]time.Time, error)
	GetLatestBlock() (db.Block, error)
}

// ValidatorRepositoryI defines the interface for validator data access operations
type ValidatorRepositoryI interface {
	GetValidators(pagination dto.PaginationQuery, isActive bool, sortBy, search string) ([]dto.ValidatorWithVoteCountModel, int64, error)
	GetValidatorsByPower(pagination *dto.PaginationQuery, onlyActive bool) ([]db.Validator, error)
	GetValidatorRow(operatorAddr string) (*db.Validator, error)
	GetValidatorBlockVoteByBlockLimit(minHeight, maxHeight int64) ([]dto.ValidatorBlockVote, error)
	GetValidatorCommitSignatures(operatorAddr string, minHeight, maxHeight int64) ([]dto.ValidatorBlockVote, error)
	GetValidatorSlashEvents(operatorAddr string, minTimestamp time.Time) ([]dto.ValidatorUptimeEvent, error)
	GetValidatorUptimeInfo(operatorAddr string) (*dto.ValidatorWithVoteCountModel, error)
	GetValidatorBondedTokenChanges(pagination dto.PaginationQuery, operatorAddr string) ([]db.ValidatorBondedTokenChange, int64, error)
	GetValidatorProposedBlocks(pagination dto.PaginationQuery, operatorAddr string) ([]dto.ValidatorProposedBlock, int64, error)
	GetValidatorHistoricalPowers(operatorAddr string) ([]dto.ValidatorHistoricalPower, int64, error)
}
