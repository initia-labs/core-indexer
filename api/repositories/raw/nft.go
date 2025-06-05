package raw

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
	"gorm.io/gorm"
)

// nftRepository implements repositories.NFTRepository using raw SQL
type nftRepository struct {
	db *gorm.DB
}

// NewNFTRepository creates a new SQL-based NFT repository
func NewNFTRepository(db *gorm.DB) repositories.NFTRepository {
	return &nftRepository{
		db: db,
	}
}

// GetCollections retrieves NFT collections with pagination and search
func (r *nftRepository) GetCollections(pagination dto.PaginationQuery, search string) ([]dto.NFTCollection, int64, error) {
	var collections []dto.NFTCollection
	var total int64

	// NOTE: Do we need to add ctx here?
	query := r.db.Table(db.TableNameCollection).
		Select("id, name, description, creator, uri")

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
