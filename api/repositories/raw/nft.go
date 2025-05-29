package raw

import (
	"database/sql"
	"strings"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

// nftRepository implements repositories.NFTRepository using raw SQL
type nftRepository struct {
	db *sql.DB
}

// NewNFTRepository creates a new SQL-based NFT repository
func NewNFTRepository(db *sql.DB) repositories.NFTRepository {
	return &nftRepository{
		db: db,
	}
}

// GetCollections retrieves NFT collections with pagination and search
func (r *nftRepository) GetCollections(pagination dto.PaginationQuery, search string) ([]dto.NFTCollection, int64, error) {
	// Build the query
	query := `
		SELECT name, uri, description, id, creator
		FROM collections
		WHERE ($1 = '' OR name ILIKE $2 OR id = $3)
		ORDER BY name %s
		LIMIT $4 OFFSET $5
	`

	// Set order direction based on reverse flag
	orderDirection := "ASC"
	if pagination.Reverse {
		orderDirection = "DESC"
	}
	query = strings.Replace(query, "%s", orderDirection, 1)

	// Build the count query
	countQuery := `
		SELECT COUNT(*)
		FROM collections
		WHERE ($1 = '' OR name ILIKE $2 OR id = $3)
	`

	// Prepare search parameters
	searchPattern := "%" + search + "%"
	searchLower := strings.ToLower(search)

	// Execute queries
	rows, err := r.db.Query(query, search, searchPattern, searchLower, pagination.Limit, pagination.Offset)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query NFT collections")
		return nil, 0, err
	}
	defer rows.Close()

	// Get total count if requested
	var total int64
	if pagination.CountTotal {
		err = r.db.QueryRow(countQuery, search, searchPattern, searchLower).Scan(&total)
		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to get NFT collections count")
			return nil, 0, err
		}
	}

	// Scan results
	var collections []dto.NFTCollection
	for rows.Next() {
		var collection dto.NFTCollection
		if err := rows.Scan(&collection.Name, &collection.URI, &collection.Description, &collection.ID, &collection.Creator); err != nil {
			logger.Get().Error().Err(err).Msg("Failed to scan NFT collection")
			return nil, 0, err
		}
		collections = append(collections, collection)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		logger.Get().Error().Err(err).Msg("Error iterating NFT collections")
		return nil, 0, err
	}

	return collections, total, nil
}
