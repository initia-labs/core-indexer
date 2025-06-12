package repositories

import (
	"strings"

	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/utils"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

// nftRepository implements NFTRepository
type nftRepository struct {
	db *gorm.DB
}

// NewNFTRepository creates a new SQL-based NFT repository
func NewNFTRepository(db *gorm.DB) NFTRepository {
	return &nftRepository{
		db: db,
	}
}

// GetCollections retrieves NFT collections with pagination and search
func (r *nftRepository) GetCollections(pagination dto.PaginationQuery, search string) ([]db.Collection, int64, error) {
	var record []db.Collection
	var total int64

	orderDirection := "asc"
	if pagination.Reverse {
		orderDirection = "desc"
	}

	query := r.db.Model(&db.Collection{}).
		Select("name, uri, description, id, creator").
		Order("name " + orderDirection).
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset))

	search = strings.TrimSpace(search)
	if search != "" {
		query = query.Where(`name ~* ? OR id = ?`, search, strings.ToLower(search))
	}

	err := query.
		Find(&record).
		Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query NFT collections")
		return nil, 0, err
	}

	if pagination.CountTotal {
		err := r.db.Model(&db.Collection{}).
			Where("name ~* ? OR id = ?", search, strings.ToLower(search)).
			Count(&total).Error

		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count NFT collections")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *nftRepository) GetCollectionsByAccountAddress(accountAddress string) ([]dto.CollectionByAccountAddressModel, error) {
	var record []dto.CollectionByAccountAddressModel

	err := r.db.Model(&db.Collection{}).
		Select("collections.name, collections.creator, collections.uri, collections.description, collections.id, COUNT(nfts.id) AS count").
		Joins("JOIN nfts ON nfts.collection = collections.id AND nfts.owner = ? AND nfts.is_burned = false", accountAddress).
		Group("collections.id").
		Order("collections.name ASC").
		Find(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to get collections by account address")
		return nil, err
	}

	return record, nil
}

func (r *nftRepository) GetCollectionsByCollectionAddress(collectionAddress string) (*db.Collection, error) {
	var record db.Collection

	err := r.db.Model(&db.Collection{}).
		Select("name, uri, description, id, creator").
		Where("id = ?", collectionAddress).
		First(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to get collection by address")
		return nil, err
	}

	return &record, nil
}

func (r *nftRepository) GetCollectionActivities(pagination dto.PaginationQuery, collectionAddress string, search string) ([]dto.CollectionActivityModel, int64, error) {
	var record []dto.CollectionActivityModel
	var total int64

	orderDirection := "asc"
	if pagination.Reverse {
		orderDirection = "desc"
	}

	query := r.db.Model(&db.CollectionTransaction{}).
		Joins("LEFT JOIN transactions ON collection_transactions.tx_id = transactions.id").
		Joins("LEFT JOIN blocks ON transactions.block_height = blocks.height").
		Joins("LEFT JOIN nfts ON collection_transactions.nft_id = nfts.id").
		Select(`
			encode(transactions.hash::bytea, 'hex') as hash,
			blocks.timestamp,
			collection_transactions.is_nft_burn,
			collection_transactions.is_nft_mint,
			collection_transactions.is_nft_transfer,
			collection_transactions.nft_id,
			nfts.token_id,
			collection_transactions.is_collection_create
		`).
		Where("collection_transactions.collection_id = ?", collectionAddress).
		Order("collection_transactions.block_height " + orderDirection).
		Order("nfts.token_id " + orderDirection).
		Order("collection_transactions.is_nft_burn " + orderDirection).
		Order("collection_transactions.is_nft_transfer " + orderDirection).
		Order("collection_transactions.is_nft_mint " + orderDirection).
		Order("collection_transactions.is_collection_create " + orderDirection).
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset))

	countQuery := r.db.Model(&db.CollectionTransaction{}).
		Joins("LEFT JOIN transactions ON collection_transactions.tx_id = transactions.id").
		Joins("LEFT JOIN nfts ON collection_transactions.nft_id = nfts.id").
		Where("collection_transactions.collection_id = ?", collectionAddress)

	search = strings.TrimSpace(search)

	if search != "" {
		if utils.IsTxHash(search) {
			query = query.Where("transactions.hash = ?", "\\x"+search)
			countQuery = countQuery.Where("transactions.hash = ?", "\\x"+search)
		} else {
			if utils.IsHex(search) {
				query = query.Where("nfts.token_id = ? OR collection_transactions.nft_id = ?", search, strings.ToLower(search))
				countQuery = countQuery.Where("nfts.token_id = ? OR collection_transactions.nft_id = ?", search, strings.ToLower(search))
			} else {
				query = query.Where("nfts.token_id ~* ?", search)
				countQuery = countQuery.Where("nfts.token_id ~* ?", search)
			}
		}
	}

	err := query.Find(&record).Error
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query collection activities")
		return nil, 0, err
	}

	if pagination.CountTotal {
		err := countQuery.
			Count(&total).Error

		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count collection activities")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *nftRepository) GetCollectionCreator(collectionAddress string) (*dto.CollectionCreatorModel, error) {
	var record dto.CollectionCreatorModel

	err := r.db.Model(&db.CollectionTransaction{}).
		Select(`
			blocks.height,
			blocks.timestamp,
			collections.creator,
			encode(transactions.hash::bytea, 'hex') as hash
		`).
		Joins("LEFT JOIN blocks ON collection_transactions.block_height = blocks.height").
		Joins("LEFT JOIN transactions ON collection_transactions.tx_id = transactions.id").
		Joins("LEFT JOIN collections ON collection_transactions.collection_id = collections.id").
		Where("collection_transactions.collection_id = ?", collectionAddress).
		Order("collection_transactions.block_height asc").
		Order("transactions.block_index asc").
		First(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to get collection creator")
		return nil, err
	}

	return &record, nil
}

func (r *nftRepository) GetCollectionMutateEvents(pagination dto.PaginationQuery, collectionAddress string) ([]dto.CollectionMutateEventResponse, int64, error) {
	var record []dto.CollectionMutateEventResponse
	var total int64

	err := r.db.Model(&db.CollectionMutationEvent{}).
		Select(`
			collection_mutation_events.mutated_field_name,
			collection_mutation_events.new_value,
			collection_mutation_events.old_value,
			collection_mutation_events.remark,
			blocks.timestamp
		`).
		Joins("LEFT JOIN blocks ON collection_mutation_events.block_height = blocks.height").
		Where("collection_mutation_events.collection_id = ?", collectionAddress).
		Order("collection_mutation_events.block_height desc").
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset)).
		Find(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query collection mutate events")
		return nil, 0, err
	}

	if pagination.CountTotal {
		err := r.db.Model(&db.CollectionMutationEvent{}).
			Where("collection_mutation_events.collection_id = ?", collectionAddress).
			Count(&total).Error

		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count collection mutate events")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *nftRepository) GetNFTByNFTAddress(collectionAddress string, nftAddress string) (*dto.NFTByAddressModel, error) {
	var record dto.NFTByAddressModel

	err := r.db.Model(&db.Nft{}).
		Select(`
			nfts.token_id,
			nfts.uri,
			nfts.description,
			nfts.is_burned,
			nfts.owner,
			nfts.id,
			nfts.collection,
			collections.name AS collection_name
		`).
		Joins("LEFT JOIN collections ON nfts.collection = collections.id").
		Where("nfts.collection = ? AND nfts.id = ?", collectionAddress, nftAddress).
		First(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to get NFT by address")
		return nil, err
	}

	return &record, nil
}

func (r *nftRepository) GetNFTsByAccountAddress(pagination dto.PaginationQuery, accountAddress string, collectionAddress string, search string) ([]dto.NFTByAddressModel, int64, error) {
	var record []dto.NFTByAddressModel
	var total int64

	query := r.db.Model(&db.Nft{}).
		Select(`
			nfts.token_id,
			nfts.uri,
			nfts.description,
			nfts.is_burned,
			nfts.owner,
			nfts.id,
			nfts.collection,
			collections.name AS collection_name
		`).
		Joins("LEFT JOIN collections ON nfts.collection = collections.id").
		Where("nfts.owner = ? AND nfts.is_burned = false", accountAddress).
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset))

	countQuery := r.db.Model(&db.Nft{}).
		Where("nfts.owner = ? AND nfts.is_burned = false", accountAddress)

	applyNFTFilters(query, collectionAddress, search)
	applyNFTFilters(countQuery, collectionAddress, search)

	err := query.Find(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query NFTs by account address")
		return nil, 0, err
	}

	if pagination.CountTotal {
		err := countQuery.
			Count(&total).Error

		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count NFTs by account address")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *nftRepository) GetNFTsByCollectionAddress(pagination dto.PaginationQuery, collectionAddress string, search string) ([]dto.NFTByAddressModel, int64, error) {
	var record []dto.NFTByAddressModel
	var total int64

	query := r.db.Model(&db.Nft{}).
		Select(`
			nfts.token_id,
			nfts.uri,
			nfts.description,
			nfts.is_burned,
			nfts.owner,
			nfts.id,
			nfts.collection,
			collections.name AS collection_name
		`).
		Joins("LEFT JOIN collections ON nfts.collection = collections.id").
		Where("nfts.collection = ? AND nfts.is_burned = false", collectionAddress).
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset))

	countQuery := r.db.Model(&db.Nft{}).
		Where("nfts.collection = ? AND nfts.is_burned = false", collectionAddress)

	applyNFTFilters(query, collectionAddress, search)
	applyNFTFilters(countQuery, collectionAddress, search)

	err := query.Find(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query NFTs by collection address")
		return nil, 0, err
	}

	if pagination.CountTotal {
		err := countQuery.
			Count(&total).Error

		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count NFTs by collection address")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *nftRepository) GetNFTMintInfo(nftAddress string) (*dto.NFTMintInfoModel, error) {
	var record dto.NFTMintInfoModel

	query := r.db.Model(&db.NftTransaction{}).
		Select(`
			accounts.address,
			encode(transactions.hash::bytea, 'hex') as hash,
			blocks.height,
			blocks.timestamp
		`).
		Joins("LEFT JOIN transactions ON nft_transactions.tx_id = transactions.id").
		Joins("LEFT JOIN blocks ON transactions.block_height = blocks.height").
		Joins("LEFT JOIN accounts ON transactions.sender = accounts.address").
		Where("nft_transactions.is_nft_mint = true AND nft_transactions.nft_id = ?", nftAddress)

	err := query.First(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to get NFT mint info")
		return nil, err
	}

	return &record, nil
}

func (r *nftRepository) GetNFTMutateEvents(pagination dto.PaginationQuery, nftAddress string) ([]dto.NFTMutateEventResponse, int64, error) {
	var record []dto.NFTMutateEventResponse
	var total int64

	err := r.db.Model(&db.NftMutationEvent{}).
		Select(`
			nft_mutation_events.old_value,
			nft_mutation_events.new_value,
			nft_mutation_events.remark,
			nft_mutation_events.mutated_field_name,
			blocks.timestamp
		`).
		Joins("LEFT JOIN blocks ON nft_mutation_events.block_height = blocks.height").
		Where("nft_mutation_events.nft_id = ?", nftAddress).
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset)).
		Find(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query NFT mutate events")
		return nil, 0, err
	}

	if pagination.CountTotal {
		err := r.db.Model(&db.NftMutationEvent{}).
			Where("nft_mutation_events.nft_id = ?", nftAddress).
			Count(&total).Error

		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count NFT mutate events")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *nftRepository) GetNFTTxs(pagination dto.PaginationQuery, nftAddress string) ([]dto.NFTTx, int64, error) {
	var record []dto.NFTTx
	var total int64

	orderDirection := "asc"
	if pagination.Reverse {
		orderDirection = "desc"
	}

	err := r.db.Model(&db.NftTransaction{}).
		Select(`
			nft_transactions.is_nft_burn,
			nft_transactions.is_nft_mint,
			nft_transactions.is_nft_transfer,
			encode(transactions.hash::bytea, 'hex') as hash,
			blocks.height,
			blocks.timestamp
		`).
		Joins("LEFT JOIN nfts ON nft_transactions.nft_id = nfts.id").
		Joins("LEFT JOIN transactions ON nft_transactions.tx_id = transactions.id").
		Joins("LEFT JOIN blocks ON transactions.block_height = blocks.height").
		Where("nft_transactions.nft_id = ?", nftAddress).
		Order("nft_transactions.block_height " + orderDirection).
		Order("nfts.token_id " + orderDirection).
		Order("nft_transactions.is_nft_burn " + orderDirection).
		Order("nft_transactions.is_nft_transfer " + orderDirection).
		Order("nft_transactions.is_nft_mint " + orderDirection).
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset)).
		Find(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query NFT transactions")
		return nil, 0, err
	}

	if pagination.CountTotal {
		err := r.db.Model(&db.NftTransaction{}).
			Where("nft_transactions.nft_id = ?", nftAddress).
			Count(&total).Error

		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count NFT transactions")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func applyNFTFilters(query *gorm.DB, collectionAddress string, search string) *gorm.DB {
	if collectionAddress != "" {
		query = query.Where("nfts.collection = ?", collectionAddress)
	}

	search = strings.TrimSpace(search)
	if search != "" {
		if utils.IsHex(search) {
			query = query.Where(`("nfts"."token_id" ~* ? OR "nfts"."id" = ?)`, search, strings.ToLower(search))
		} else {
			query = query.Where(`"nfts"."token_id" ~* ?`, search)
		}
	}

	return query
}
