package repositories

import (
	"regexp"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/utils"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

var _ NftRepositoryI = &NftRepository{}

// NftRepository implements NftRepositoryI
type NftRepository struct {
	db *gorm.DB
}

// NewNftRepository creates a new SQL-based Nft repository
func NewNftRepository(db *gorm.DB) *NftRepository {
	return &NftRepository{
		db: db,
	}
}

// GetCollections retrieves Nft collections with pagination and search
func (r *NftRepository) GetCollections(pagination dto.PaginationQuery, search string) ([]db.Collection, int64, error) {
	record := make([]db.Collection, 0)
	total := int64(0)

	query := r.db.Model(&db.Collection{}).
		Select("name, uri, description, id, creator").
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "name",
			},
			Desc: pagination.Reverse,
		}).
		Limit(pagination.Limit).
		Offset(pagination.Offset)

	search = strings.TrimSpace(search)
	if search != "" {
		safeSearch := regexp.QuoteMeta(search)
		query = query.Where(`name ~* ? OR id = ?`, safeSearch, strings.ToLower(safeSearch))
	}

	if err := query.
		Find(&record).
		Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query Nft collections")
		return nil, 0, err
	}

	// Get total count if requested
	if pagination.CountTotal {
		safeSearch := regexp.QuoteMeta(search)
		if err := r.db.Model(&db.Collection{}).
			Where("name ~* ? OR id = ?", safeSearch, strings.ToLower(safeSearch)).
			Count(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count Nft collections")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *NftRepository) GetCollectionsByAccountAddress(accountAddress string) ([]dto.CollectionByAccountAddressModel, error) {
	record := make([]dto.CollectionByAccountAddressModel, 0)

	if err := r.db.Model(&db.Collection{}).
		Select("collections.name, collections.creator, collections.uri, collections.description, collections.id, COUNT(nfts.id) AS count").
		Joins("JOIN nfts ON nfts.collection = collections.id AND nfts.owner = ? AND nfts.is_burned = false", accountAddress).
		Clauses(
			clause.GroupBy{
				Columns: []clause.Column{
					{Name: "collections.id"},
				},
			},
			clause.OrderBy{
				Columns: []clause.OrderByColumn{
					{Column: clause.Column{Name: "collections.name"}, Desc: false},
				},
			},
		).
		Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to get collections by account address")
		return nil, err
	}

	return record, nil
}

func (r *NftRepository) GetCollectionsByCollectionAddress(collectionAddress string) (*db.Collection, error) {
	var record db.Collection

	if err := r.db.Model(&db.Collection{}).
		Select("name, uri, description, id, creator").
		Where("id = ?", collectionAddress).
		First(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to get collection by address")
		return nil, err
	}

	return &record, nil
}

func (r *NftRepository) GetCollectionActivities(pagination dto.PaginationQuery, collectionAddress string, search string) ([]dto.CollectionActivityModel, int64, error) {
	record := make([]dto.CollectionActivityModel, 0)
	total := int64(0)

	query := r.db.Model(&db.CollectionTransaction{}).
		Joins("LEFT JOIN transactions ON collection_transactions.tx_id = transactions.id").
		Joins("LEFT JOIN blocks ON transactions.block_height = blocks.height").
		Joins("LEFT JOIN nfts ON collection_transactions.nft_id = nfts.id").
		Select(`
			transactions.hash,
			blocks.timestamp,
			collection_transactions.is_nft_burn,
			collection_transactions.is_nft_mint,
			collection_transactions.is_nft_transfer,
			collection_transactions.nft_id,
			nfts.token_id,
			collection_transactions.is_collection_create
		`).
		Where("collection_transactions.collection_id = ?", collectionAddress).
		Clauses(clause.OrderBy{
			Columns: []clause.OrderByColumn{
				{Column: clause.Column{Name: "collection_transactions.block_height"}, Desc: pagination.Reverse},
				{Column: clause.Column{Name: "nfts.token_id"}, Desc: pagination.Reverse},
				{Column: clause.Column{Name: "collection_transactions.is_nft_burn"}, Desc: pagination.Reverse},
				{Column: clause.Column{Name: "collection_transactions.is_nft_transfer"}, Desc: pagination.Reverse},
				{Column: clause.Column{Name: "collection_transactions.is_nft_mint"}, Desc: pagination.Reverse},
				{Column: clause.Column{Name: "collection_transactions.is_collection_create"}, Desc: pagination.Reverse},
			},
		}).
		Limit(pagination.Limit).
		Offset(pagination.Offset)

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
				safeSearch := regexp.QuoteMeta(search)
				query = query.Where("nfts.token_id ~* ?", safeSearch)
				countQuery = countQuery.Where("nfts.token_id ~* ?", safeSearch)
			}
		}
	}

	if err := query.Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query collection activities")
		return nil, 0, err
	}

	if pagination.CountTotal {
		if err := countQuery.
			Count(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count collection activities")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *NftRepository) GetCollectionCreator(collectionAddress string) (*dto.CollectionCreatorModel, error) {
	var record dto.CollectionCreatorModel

	if err := r.db.Model(&db.CollectionTransaction{}).
		Select(`
			blocks.height,
			blocks.timestamp,
			collections.creator,
			transactions.hash
		`).
		Joins("LEFT JOIN blocks ON collection_transactions.block_height = blocks.height").
		Joins("LEFT JOIN transactions ON collection_transactions.tx_id = transactions.id").
		Joins("LEFT JOIN collections ON collection_transactions.collection_id = collections.id").
		Where("collection_transactions.collection_id = ?", collectionAddress).
		Clauses(clause.OrderBy{
			Columns: []clause.OrderByColumn{
				{Column: clause.Column{Name: "collection_transactions.block_height"}, Desc: false},
				{Column: clause.Column{Name: "transactions.block_index"}, Desc: false},
			},
		}).
		First(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to get collection creator")
		return nil, err
	}

	return &record, nil
}

func (r *NftRepository) GetCollectionMutateEvents(pagination dto.PaginationQuery, collectionAddress string) ([]dto.MutateEventModel, int64, error) {
	record := make([]dto.MutateEventModel, 0)
	total := int64(0)

	if err := r.db.Model(&db.CollectionMutationEvent{}).
		Select(`
			collection_mutation_events.mutated_field_name,
			collection_mutation_events.new_value,
			collection_mutation_events.old_value,
			collection_mutation_events.remark,
			blocks.timestamp
		`).
		Joins("LEFT JOIN blocks ON collection_mutation_events.block_height = blocks.height").
		Where("collection_mutation_events.collection_id = ?", collectionAddress).
		Clauses(clause.OrderBy{
			Columns: []clause.OrderByColumn{
				{Column: clause.Column{Name: "collection_mutation_events.block_height"}, Desc: pagination.Reverse},
			},
		}).
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query collection mutate events")
		return nil, 0, err
	}

	if pagination.CountTotal {
		if err := r.db.Model(&db.CollectionMutationEvent{}).
			Where("collection_mutation_events.collection_id = ?", collectionAddress).
			Count(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count collection mutate events")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *NftRepository) GetNftByNftAddress(collectionAddress string, nftAddress string) (*dto.NftByAddressModel, error) {
	var record dto.NftByAddressModel

	if err := r.db.Model(&db.Nft{}).
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
		First(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to get Nft by address")
		return nil, err
	}

	return &record, nil
}

func (r *NftRepository) GetNftsByAccountAddress(pagination dto.PaginationQuery, accountAddress string, collectionAddress string, search string) ([]dto.NftByAddressModel, int64, error) {
	record := make([]dto.NftByAddressModel, 0)
	total := int64(0)

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
		Limit(pagination.Limit).
		Offset(pagination.Offset)

	countQuery := r.db.Model(&db.Nft{}).
		Where("nfts.owner = ? AND nfts.is_burned = false", accountAddress)

	applyNftFilters(query, collectionAddress, search)
	applyNftFilters(countQuery, collectionAddress, search)

	if err := query.Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query Nfts by account address")
		return nil, 0, err
	}

	if pagination.CountTotal {
		if err := countQuery.
			Count(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count Nfts by account address")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *NftRepository) GetNftsByCollectionAddress(pagination dto.PaginationQuery, collectionAddress string, search string) ([]dto.NftByAddressModel, int64, error) {
	record := make([]dto.NftByAddressModel, 0)
	total := int64(0)

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
		Limit(pagination.Limit).
		Offset(pagination.Offset)

	countQuery := r.db.Model(&db.Nft{}).
		Where("nfts.collection = ? AND nfts.is_burned = false", collectionAddress)

	applyNftFilters(query, collectionAddress, search)
	applyNftFilters(countQuery, collectionAddress, search)

	if err := query.Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query Nfts by collection address")
		return nil, 0, err
	}

	if pagination.CountTotal {
		if err := countQuery.
			Count(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count Nfts by collection address")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *NftRepository) GetNftMintInfo(nftAddress string) (*dto.NftMintInfoModel, error) {
	var record dto.NftMintInfoModel

	query := r.db.Model(&db.NftTransaction{}).
		Select(`
			accounts.address,
			transactions.hash,
			blocks.height,
			blocks.timestamp
		`).
		Joins("LEFT JOIN transactions ON nft_transactions.tx_id = transactions.id").
		Joins("LEFT JOIN blocks ON transactions.block_height = blocks.height").
		Joins("LEFT JOIN accounts ON transactions.sender = accounts.address").
		Where("nft_transactions.is_nft_mint = true AND nft_transactions.nft_id = ?", nftAddress)

	if err := query.First(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to get Nft mint info")
		return nil, err
	}

	return &record, nil
}

func (r *NftRepository) GetNftMutateEvents(pagination dto.PaginationQuery, nftAddress string) ([]dto.MutateEventModel, int64, error) {
	record := make([]dto.MutateEventModel, 0)
	total := int64(0)

	if err := r.db.Model(&db.NftMutationEvent{}).
		Select(`
			nft_mutation_events.old_value,
			nft_mutation_events.new_value,
			nft_mutation_events.remark,
			nft_mutation_events.mutated_field_name,
			blocks.timestamp
		`).
		Joins("LEFT JOIN blocks ON nft_mutation_events.block_height = blocks.height").
		Where("nft_mutation_events.nft_id = ?", nftAddress).
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query Nft mutate events")
		return nil, 0, err
	}

	if pagination.CountTotal {
		if err := r.db.Model(&db.NftMutationEvent{}).
			Where("nft_mutation_events.nft_id = ?", nftAddress).
			Count(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count Nft mutate events")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *NftRepository) GetNftTxs(pagination dto.PaginationQuery, nftAddress string) ([]dto.NftTxModel, int64, error) {
	record := make([]dto.NftTxModel, 0)
	total := int64(0)

	if err := r.db.Model(&db.NftTransaction{}).
		Select(`
			nft_transactions.is_nft_burn,
			nft_transactions.is_nft_mint,
			nft_transactions.is_nft_transfer,
			transactions.hash,
			blocks.height,
			blocks.timestamp
		`).
		Joins("LEFT JOIN nfts ON nft_transactions.nft_id = nfts.id").
		Joins("LEFT JOIN transactions ON nft_transactions.tx_id = transactions.id").
		Joins("LEFT JOIN blocks ON transactions.block_height = blocks.height").
		Where("nft_transactions.nft_id = ?", nftAddress).
		Clauses(clause.OrderBy{
			Columns: []clause.OrderByColumn{
				{Column: clause.Column{Name: "nft_transactions.block_height"}, Desc: pagination.Reverse},
				{Column: clause.Column{Name: "nfts.token_id"}, Desc: pagination.Reverse},
				{Column: clause.Column{Name: "nft_transactions.is_nft_burn"}, Desc: pagination.Reverse},
				{Column: clause.Column{Name: "nft_transactions.is_nft_transfer"}, Desc: pagination.Reverse},
				{Column: clause.Column{Name: "nft_transactions.is_nft_mint"}, Desc: pagination.Reverse},
			},
		}).
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query Nft transactions")
		return nil, 0, err
	}

	if pagination.CountTotal {
		if err := r.db.Model(&db.NftTransaction{}).
			Where("nft_transactions.nft_id = ?", nftAddress).
			Count(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count Nft transactions")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func applyNftFilters(query *gorm.DB, collectionAddress string, search string) *gorm.DB {
	if collectionAddress != "" {
		query = query.Where("nfts.collection = ?", collectionAddress)
	}

	search = strings.TrimSpace(search)
	if search != "" {
		safeSearch := regexp.QuoteMeta(search)
		if utils.IsHex(search) {
			query = query.Where(`("nfts"."token_id" ~* ? OR "nfts"."id" = ?)`, safeSearch, strings.ToLower(safeSearch))
		} else {
			query = query.Where(`"nfts"."token_id" ~* ?`, safeSearch)
		}
	}

	return query
}
