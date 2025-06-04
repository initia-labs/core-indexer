package raw

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/utils"
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

func (r *nftRepository) GetNFTByNFTAddress(collectionAddress string, nftAddress string) (*dto.NFTByAddress, error) {
	query := `
		SELECT
			"nfts"."token_id",
			"nfts"."uri",
			"nfts"."description",
			"nfts"."is_burned",
			"nfts"."owner",
			"nfts"."id",
			"nfts"."collection",
			"collections"."name"
		FROM "nfts"
		LEFT JOIN "collections" ON "nfts"."collection" = "collections"."id"
		WHERE "nfts"."collection" = $1 AND "nfts"."id" = $2
		LIMIT 1
	`

	var foundNft dto.NFTByAddress

	err := r.db.QueryRow(query, collectionAddress, nftAddress).Scan(
		&foundNft.TokenID,
		&foundNft.URI,
		&foundNft.Description,
		&foundNft.IsBurned,
		&foundNft.Owner,
		&foundNft.ID,
		&foundNft.Collection,
		&foundNft.CollectionName,
	)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query NFT by address")
		return nil, err
	}

	return &dto.NFTByAddress{
		TokenID:        foundNft.TokenID,
		URI:            foundNft.URI,
		Description:    foundNft.Description,
		IsBurned:       foundNft.IsBurned,
		Owner:          foundNft.Owner,
		ID:             foundNft.ID,
		Collection:     foundNft.Collection,
		CollectionName: foundNft.CollectionName,
	}, nil
}

func (r *nftRepository) GetNFTsByAccountAddress(pagination dto.PaginationQuery, accountAddress string, collectionAddress *string, search *string) ([]dto.NFTByAddress, int64, error) {
	query := `
		SELECT
			"nfts"."token_id",
			"nfts"."uri",
			"nfts"."description",
			"nfts"."is_burned",
			"nfts"."owner",
			"nfts"."id",
			"nfts"."collection",
			"collections"."name"
		FROM "nfts"
		LEFT JOIN "collections" ON "nfts"."collection" = "collections"."id"
		WHERE "nfts"."owner" = $1 AND "nfts"."is_burned" = false
		LIMIT $2 OFFSET $3
	`

	countQuery := `
		SELECT COUNT(*)
		FROM "nfts"
		WHERE "nfts"."owner" = $1 AND "nfts"."is_burned" = false
	`

	queryArgs := []interface{}{accountAddress, pagination.Limit, pagination.Offset}
	queryArgIdx := 4

	countQueryArgs := []interface{}{accountAddress}
	countQueryArgIdx := 2

	if collectionAddress != nil && *collectionAddress != "" {
		query += fmt.Sprintf(` AND "nfts"."collection" = $%d`, queryArgIdx)
		queryArgs = append(queryArgs, *collectionAddress)
		queryArgIdx++

		countQuery += fmt.Sprintf(` AND "nfts"."collection" = $%d`, countQueryArgIdx)
		countQueryArgs = append(countQueryArgs, *collectionAddress)
		countQueryArgIdx++
	}

	if search != nil && *search != "" {
		query += fmt.Sprintf(` AND "nfts"."token_id" ~* $%d`, queryArgIdx)
		queryArgs = append(queryArgs, *search)
		queryArgIdx++

		countQuery += fmt.Sprintf(` AND "nfts"."token_id" ~* $%d`, countQueryArgIdx)
		countQueryArgs = append(countQueryArgs, search)
		countQueryArgIdx++

		if utils.IsHex(*search) {
			query += fmt.Sprintf(` OR "nfts"."id" = $%d`, queryArgIdx)
			queryArgs = append(queryArgs, strings.ToLower(*search))
			queryArgIdx++

			countQuery += fmt.Sprintf(` OR "nfts"."id" = $%d`, countQueryArgIdx)
			countQueryArgs = append(countQueryArgs, strings.ToLower(*search))
			countQueryArgIdx++
		}
	}

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query NFTs by account address")
		return nil, 0, err
	}
	defer rows.Close()

	var total int64
	if pagination.CountTotal {
		err = r.db.QueryRow(countQuery, countQueryArgs...).Scan(&total)
		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to get NFT count by account address")
			return nil, 0, err
		}
	}

	var nfts []dto.NFTByAddress
	for rows.Next() {
		var nft dto.NFTByAddress
		if err := rows.Scan(
			&nft.TokenID,
			&nft.URI,
			&nft.Description,
			&nft.IsBurned,
			&nft.Owner,
			&nft.ID,
			&nft.Collection,
			&nft.CollectionName,
		); err != nil {
			logger.Get().Error().Err(err).Msg("Failed to scan NFT by account address")
			return nil, 0, err
		}

		nfts = append(nfts, nft)
	}

	if err := rows.Err(); err != nil {
		logger.Get().Error().Err(err).Msg("Error iterating NFTs by account address")
		return nil, 0, err
	}

	return nfts, total, nil
}

func (r *nftRepository) GetNFTsByCollectionAddress(pagination dto.PaginationQuery, collectionAddress string, search *string) ([]dto.NFTByAddress, int64, error) {
	query := `
		SELECT
			"nfts"."token_id",
			"nfts"."uri",
			"nfts"."description",
			"nfts"."is_burned",
			"nfts"."owner",
			"nfts"."id",
			"nfts"."collection",
			"collections"."name"
		FROM "nfts"
		LEFT JOIN "collections" ON "nfts"."collection" = "collections"."id"
		WHERE "nfts"."collection" = $1 AND "nfts"."is_burned" = false
		LIMIT $2 OFFSET $3
	`

	countQuery := `
		SELECT COUNT(*)
		FROM "nfts"
		WHERE "nfts"."collection" = $1 AND "nfts"."is_burned" = false
	`

	queryArgs := []interface{}{collectionAddress, pagination.Limit, pagination.Offset}
	queryArgIndex := 4

	countQueryArgs := []interface{}{collectionAddress}
	countQueryArgIndex := 2

	if search != nil && *search != "" {
		query += fmt.Sprintf(` AND "nfts"."token_id" ~* $%d`, queryArgIndex)
		queryArgs = append(queryArgs, *search)
		queryArgIndex++

		countQuery += fmt.Sprintf(` AND "nfts"."token_id" ~* $%d`, countQueryArgIndex)
		countQueryArgs = append(countQueryArgs, search)
		countQueryArgIndex++

		if utils.IsHex(*search) {
			query += fmt.Sprintf(` OR "nfts"."id" = $%d`, queryArgIndex)
			queryArgs = append(queryArgs, strings.ToLower(*search))
			queryArgIndex++

			countQuery += fmt.Sprintf(` OR "nfts"."id" = $%d`, countQueryArgIndex)
			countQueryArgs = append(countQueryArgs, strings.ToLower(*search))
			countQueryArgIndex++
		}
	}

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query NFTs by collection address")
		return nil, 0, err
	}
	defer rows.Close()

	var total int64
	if pagination.CountTotal {
		err = r.db.QueryRow(countQuery, countQueryArgs...).Scan(&total)
		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to get NFT count by collection address")
			return nil, 0, err
		}
	}

	var nfts []dto.NFTByAddress
	for rows.Next() {
		var nft dto.NFTByAddress
		if err := rows.Scan(
			&nft.TokenID,
			&nft.URI,
			&nft.Description,
			&nft.IsBurned,
			&nft.Owner,
			&nft.ID,
			&nft.Collection,
			&nft.CollectionName,
		); err != nil {
			logger.Get().Error().Err(err).Msg("Failed to scan NFT by collection address")
			return nil, 0, err
		}

		nfts = append(nfts, nft)
	}

	if err := rows.Err(); err != nil {
		logger.Get().Error().Err(err).Msg("Error iterating NFTs by collection address")
		return nil, 0, err
	}

	return nfts, total, nil
}
