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
	var collections []db.Collection
	var total int64

	query := r.db.Model(&db.Collection{})

	if search != "" {
		query = query.Where("name ILIKE ? OR id = ?", "%"+search+"%", search)
	}

	orderDirection := "name asc"
	if pagination.Reverse {
		orderDirection = "name desc"
	}

	// Fetch paginated results
	if err := query.
		Order(orderDirection).
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset)).
		Find(&collections).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query NFT collections")
		return nil, 0, err
	}

	if pagination.CountTotal {
		if err := query.Count(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count NFT collections")
			return nil, 0, err
		}
	}

	return collections, total, nil
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
		Select("COUNT(*)").
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
			First(&total).Error

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
		Select("COUNT(*)").
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
			First(&total).Error

		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count NFTs by collection address")
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
