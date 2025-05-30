package raw

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/pkg/logger"
	"github.com/rs/zerolog/log"
	"gocloud.dev/blob"
)

// txRepository implements repositories.TxRepository using raw SQL
type txRepository struct {
	db     *sql.DB
	bucket *blob.Bucket
}

func NewTxRepository(db *sql.DB, bucket *blob.Bucket) repositories.TxRepository {
	return &txRepository{
		db:     db,
		bucket: bucket,
	}
}

func (r *txRepository) GetTxByHash(hash string) (*dto.RestTxByHashResponse, error) {
	ctx := context.Background()
	iter := r.bucket.List(&blob.ListOptions{
		Prefix: hash + "/", // Add trailing slash to ensure we only get files under this hash
	})
	var largestName string
	var largestNum int64

	for {
		obj, err := iter.Next(ctx)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Error().Err(err).Msg("Error getting next object")
			return nil, err
		}

		// Extract block height from the path (hash/block_height)
		parts := strings.Split(obj.Key, "/")
		if len(parts) != 2 {
			continue
		}

		// Convert block height to integer for comparison
		if num, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
			if num > largestNum {
				largestNum = num
				largestName = obj.Key
			}
		}
	}

	if largestName == "" {
		return nil, apperror.NewNotFound(fmt.Sprintf("no valid transaction files found for hash %s", hash))
	}

	log.Info().Str("hash", hash).Str("block_height", strings.Split(largestName, "/")[1]).Msg("Found latest transaction")

	tx, err := r.bucket.NewReader(ctx, largestName, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Close()

	txResponse := &dto.RestTxByHashResponse{}
	err = json.NewDecoder(tx).Decode(txResponse)
	if err != nil {
		return nil, err
	}
	return txResponse, nil
}

func (r *txRepository) GetTxCount() (*dto.RestTxCountResponse, error) {
	query := `
		SELECT tx_count FROM tracking
		LIMIT 1
	`

	var txCount int64

	err := r.db.QueryRow(query).Scan(&txCount)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query tracking data for transaction count")
		return nil, err
	}

	return &dto.RestTxCountResponse{
		Count: txCount,
	}, nil
}

func (r *txRepository) GetTxs(pagination dto.PaginationQuery) ([]dto.TxResponse, int64, error) {
	query := `
		SELECT
			"tx"."sender",
			"tx"."hash",
			"tx"."success",
			"tx"."messages",
			"tx"."is_send",
			"tx"."is_ibc",
			"tx"."is_opinit",
			"b"."height",
			"b"."timestamp"
		FROM "transactions" as "tx"
		LEFT JOIN "blocks" as "b" on "tx"."block_height" = "b"."height"
		ORDER BY "tx"."block_height" %s, "tx"."block_index" %s
		LIMIT $1 OFFSET $2
	`

	orderDirection := "ASC"
	if pagination.Reverse {
		orderDirection = "DESC"
	}
	query = strings.ReplaceAll(query, "%s", orderDirection)

	countQuery := `
		SELECT "tracking"."tx_count"
		FROM "tracking"
		LIMIT 1
	`

	rows, err := r.db.Query(query, pagination.Limit, pagination.Offset)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query transactions")
		return nil, 0, err
	}
	defer rows.Close()

	var total int64
	if pagination.CountTotal {
		err := r.db.QueryRow(countQuery).Scan(&total)
		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to get transaction count")
			return nil, 0, err
		}
	}

	var txs []dto.TxResponse
	for rows.Next() {
		var tx dto.TxResponse
		if err := rows.Scan(
			&tx.Sender,
			&tx.Hash,
			&tx.Success,
			&tx.Messages,
			&tx.IsSend,
			&tx.IsIbc,
			&tx.IsOpinit,
			&tx.Height,
			&tx.Timestamp,
		); err != nil {
			logger.Get().Error().Err(err).Msg("Failed to scan transaction")
			return nil, 0, err
		}
		tx.Hash = "\\x" + hex.EncodeToString([]byte(tx.Hash))

		txs = append(txs, tx)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		logger.Get().Error().Err(err).Msg("Error iterating transactions")
		return nil, 0, err
	}

	return txs, total, nil
}
