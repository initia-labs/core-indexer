package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"gocloud.dev/blob"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

// txRepository implements TxRepository
type txRepository struct {
	db     *gorm.DB
	bucket *blob.Bucket
}

// NewTxRepository creates a new SQL-based NFT repository
func NewTxRepository(db *gorm.DB, bucket *blob.Bucket) TxRepository {
	return &txRepository{
		db:     db,
		bucket: bucket,
	}
}

// GetTxByHash retrieves a transaction by hash
func (r *txRepository) GetTxByHash(hash string) (*dto.TxByHashResponse, error) {
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

	txResponse := &dto.TxByHashResponse{}
	err = json.NewDecoder(tx).Decode(txResponse)
	if err != nil {
		return nil, err
	}
	return txResponse, nil
}

// GetTxCount retrieves the total number of transactions
func (r *txRepository) GetTxCount() (*int64, error) {
	var record db.Tracking

	err := r.db.Model(&db.Tracking{}).
		Select("tx_count").
		First(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query tracking data for transaction count")
		return nil, err
	}

	return &record.TxCount, nil
}

// GetTxs retrieves a list of transactions with pagination
func (r *txRepository) GetTxs(pagination dto.PaginationQuery) ([]dto.TxModel, int64, error) {
	var record []dto.TxModel
	var total int64

	orderDirection := "asc"
	if pagination.Reverse {
		orderDirection = "desc"
	}

	err := r.db.
		Model(&db.Transaction{}).
		Select("transactions.sender, encode(transactions.hash::bytea, 'hex') as hash, transactions.success, transactions.messages, transactions.is_send, transactions.is_ibc, transactions.is_opinit, blocks.height, blocks.timestamp").
		Joins("LEFT JOIN blocks ON transactions.block_height = blocks.height").
		Order("transactions.block_height " + orderDirection).
		Order("transactions.block_index " + orderDirection).
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset)).
		Find(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query transactions")
		return nil, 0, err
	}

	if pagination.CountTotal {
		err := r.db.Model(&db.Tracking{}).
			Select("tx_count").
			First(&total).Error

		if err != nil {
			logger.Get().Error().Err(err).Msg("Failed to get transaction count")
			return nil, 0, err
		}
	}

	return record, total, nil
}
