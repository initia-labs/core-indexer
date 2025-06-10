package repositories

import (
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

var _ NFTRepositoryI = &NftRepository{}

// NftRepository implements NFTRepositoryI
type NftRepository struct {
	db *gorm.DB
}

// NewNFTRepository creates a new SQL-based NFT repository
func NewNFTRepository(db *gorm.DB) *NftRepository {
	return &NftRepository{
		db: db,
	}
}

// GetCollections retrieves NFT collections with pagination and search
func (r *NftRepository) GetCollections(pagination dto.PaginationQuery, search string) ([]db.Collection, int64, error) {
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
