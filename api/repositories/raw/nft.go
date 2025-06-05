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
	orderDirection := "ASC"
	if pagination.Reverse {
		orderDirection = "DESC"
	}

	query := fmt.Sprintf(`
		SELECT "name", "uri", "description", "id", "creator"
		FROM "collections"
		ORDER BY "name" %s
		LIMIT $1 OFFSET $2
	`, orderDirection)

	countQuery := `
		SELECT COUNT(*)
		FROM collections
	`

	queryArgs := []interface{}{pagination.Limit, pagination.Offset}
	queryArgIndex := 2

	if search != "" {
		query += fmt.Sprintf(` AND ("name" ~* $%d OR "id" = $%d)`, queryArgIndex, queryArgIndex+1)
		queryArgs = append(queryArgs, search)
		queryArgs = append(queryArgs, strings.ToLower(search))
		queryArgIndex += 2
	}

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query NFT collections")
		return nil, 0, err
	}
	defer rows.Close()

	var total int64
	if pagination.CountTotal {
		err = r.db.QueryRow(countQuery).Scan(&total)
		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to get NFT collections count")
			return nil, 0, err
		}
	}

	var collections []dto.NFTCollection
	for rows.Next() {
		var collection dto.NFTCollection
		if err := rows.Scan(&collection.Name, &collection.URI, &collection.Description, &collection.ID, &collection.Creator); err != nil {
			logger.Get().Error().Err(err).Msg("Failed to scan NFT collection")
			return nil, 0, err
		}
		collections = append(collections, collection)
	}

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

	return &foundNft, nil
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

	for i, arg := range queryArgs {
		fmt.Printf("arg[%d]: %v (type: %T)\n", i+1, arg, arg)
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

func (r *nftRepository) GetNFTMintInfo(nftAddress string) (*dto.NFTMintInfo, error) {
	query := `
		SELECT 
			"accounts"."address",
			"transactions"."hash",
			"blocks"."height",
			"blocks"."timestamp"
		FROM "nft_transactions"
		LEFT JOIN "transactions" ON "nft_transactions"."tx_id" = "transactions"."id"
		LEFT JOIN "blocks" ON "transactions"."block_height" = "blocks"."height"
		LEFT JOIN "accounts" ON "transactions"."sender" = "accounts"."address"
		WHERE "nft_transactions"."is_nft_mint" = true
		AND "nft_transactions"."nft_id" = $1
		LIMIT 1
	`

	var foundNft dto.NFTMintInfo

	err := r.db.QueryRow(query, nftAddress).Scan(
		&foundNft.Address,
		&foundNft.Hash,
		&foundNft.Height,
		&foundNft.Timestamp,
	)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query NFT mint info")
		return nil, err
	}

	return &foundNft, nil
}

func (r *nftRepository) GetNFTMutateEvents(pagination dto.PaginationQuery, nftAddress string) ([]dto.NFTMutateEventResponse, int64, error) {
	query := `
		SELECT
			"nft_mutation_events"."old_value",
			"nft_mutation_events"."remark",
			"nft_mutation_events"."mutated_field_name",
			"nft_mutation_events"."new_value",
			"blocks"."timestamp"
		FROM "nft_mutation_events"
		LEFT JOIN "blocks" ON "nft_mutation_events"."block_height" = "blocks"."height"
		WHERE "nft_mutation_events"."nft_id" = $1
		LIMIT $2 OFFSET $3
	`

	countQuery := `
		SELECT COUNT(*)
		FROM "nft_mutation_events"
		WHERE "nft_mutation_events"."nft_id" = $1
	`

	rows, err := r.db.Query(query, nftAddress, pagination.Limit, pagination.Offset)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query NFT mutate events")
		return nil, 0, err
	}
	defer rows.Close()

	var total int64
	if pagination.CountTotal {
		err = r.db.QueryRow(countQuery, nftAddress).Scan(&total)
		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to get NFT mutate events count")
			return nil, 0, err
		}
	}

	var events []dto.NFTMutateEventResponse
	for rows.Next() {
		var event dto.NFTMutateEventResponse
		if err := rows.Scan(
			&event.OldValue,
			&event.NewValue,
			&event.Remark,
			&event.MutatedFieldName,
			&event.Timestamp,
		); err != nil {
			logger.Get().Error().Err(err).Msg("Failed to scan NFT mutate events")
			return nil, 0, err
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		logger.Get().Error().Err(err).Msg("Error iterating NFT mutate events")
		return nil, 0, err
	}

	return events, total, nil
}

func (r *nftRepository) GetNFTTxs(pagination dto.PaginationQuery, nftAddress string) ([]dto.NFTTx, int64, error) {
	orderDirection := "ASC"
	if pagination.Reverse {
		orderDirection = "DESC"
	}

	query := fmt.Sprintf(`
		SELECT 
			"nft_transactions"."is_nft_burn",
			"nft_transactions"."is_nft_mint",
			"nft_transactions"."is_nft_transfer",
			"transactions"."hash",
			"blocks"."height",
			"blocks"."timestamp"
		FROM "nft_transactions"
		LEFT JOIN "nfts" ON "nft_transactions"."nft_id" = "nfts"."id"
		LEFT JOIN "transactions" ON "nft_transactions"."tx_id" = "transactions"."id"
		LEFT JOIN "blocks" ON "transactions"."block_height" = "blocks"."height"
		WHERE "nft_transactions"."nft_id" = $1
		ORDER BY
			"nft_transactions"."block_height" %s,
			"nfts"."token_id" %s,
			"nft_transactions"."is_nft_burn" %s,
			"nft_transactions"."is_nft_transfer" %s,
			"nft_transactions"."is_nft_mint" %s
		LIMIT $2 OFFSET $3
	`, orderDirection, orderDirection, orderDirection, orderDirection, orderDirection)

	countQuery := `
		SELECT COUNT(*)
		FROM "nft_transactions"
		WHERE "nft_transactions"."nft_id" = $1
	`

	rows, err := r.db.Query(query, nftAddress, pagination.Limit, pagination.Offset)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query NFT txs")
		return nil, 0, err
	}
	defer rows.Close()

	var total int64
	if pagination.CountTotal {
		err = r.db.QueryRow(countQuery, nftAddress).Scan(&total)
		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to get NFT txs count")
			return nil, 0, err
		}
	}

	var txs []dto.NFTTx
	for rows.Next() {
		var tx dto.NFTTx
		if err := rows.Scan(
			&tx.IsNFTBurn,
			&tx.IsNFTMint,
			&tx.IsNFTTransfer,
			&tx.Hash,
			&tx.Height,
			&tx.Timestamp,
		); err != nil {
			logger.Get().Error().Err(err).Msg("Failed to scan NFT txs")
			return nil, 0, err
		}
		txs = append(txs, tx)
	}

	if err := rows.Err(); err != nil {
		logger.Get().Error().Err(err).Msg("Error iterating NFT txs")
		return nil, 0, err
	}

	return txs, total, nil
}
