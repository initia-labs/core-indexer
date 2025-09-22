package repositories

import (
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"gocloud.dev/blob"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

var _ TxRepositoryI = &TxRepository{}

// TxRepository implements TxRepositoryI
type TxRepository struct {
	db      *gorm.DB
	buckets []*blob.Bucket
}

// NewTxRepository creates a new SQL-based Nft repository
func NewTxRepository(db *gorm.DB, buckets []*blob.Bucket) *TxRepository {
	return &TxRepository{
		db:      db,
		buckets: buckets,
	}
}

// GetTxByHash retrieves a transaction by hash
func (r *TxRepository) GetTxByHash(hash string) (*dto.TxByHashResponse, error) {
	ctx := context.Background()
	upperHash := strings.ToUpper(hash)

	var largestName string
	var largestNum int64
	var foundBucket *blob.Bucket

	for i, bucket := range r.buckets {
		bucketName := "bucket_" + strconv.Itoa(i)
		log.Debug().Str("bucket", bucketName).Str("hash", upperHash).Msg("Searching in bucket")

		iter := bucket.List(&blob.ListOptions{
			Prefix: upperHash + "/",
		})

		for {
			obj, err := iter.Next(ctx)
			if err != nil {
				if err == io.EOF {
					break
				}

				if strings.Contains(err.Error(), "invalid_grant") {
					log.Warn().Str("bucket", bucketName).Str("hash", upperHash).Msg("Authentication failure")
					continue
				}

				log.Warn().Err(err).Str("bucket", bucketName).Str("hash", upperHash).Msg("Error listing objects")
				continue
			}

			parts := strings.Split(obj.Key, "/")
			if len(parts) != 2 {
				continue
			}

			if num, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				if num > largestNum {
					largestNum = num
					largestName = obj.Key
					foundBucket = bucket
					log.Debug().Str("bucket", bucketName).Str("hash", upperHash).Int64("block_height", num).Msg("Found newer transaction")
				}
			}
		}
	}

	if largestName == "" {
		return nil, apperror.NewNoValidTxFiles(upperHash)
	}

	log.Info().Str("hash", upperHash).Str("block_height", strings.Split(largestName, "/")[1]).Msg("Found latest transaction across all buckets")

	tx, err := foundBucket.NewReader(ctx, largestName, nil)
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
func (r *TxRepository) GetTxCount() (*int64, error) {
	var record db.Tracking

	if err := r.db.Model(&db.Tracking{}).
		Select("tx_count").
		First(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query tracking data for transaction count")
		return nil, err
	}

	return &record.TxCount, nil
}

// GetTxs retrieves a list of transactions with pagination
func (r *TxRepository) GetTxs(pagination dto.PaginationQuery) ([]dto.TxModel, int64, error) {
	record := make([]dto.TxModel, 0)
	total := int64(0)

	if err := r.db.
		Model(&db.Transaction{}).
		Select("transactions.sender, transactions.hash, transactions.success, transactions.messages, transactions.is_send, transactions.is_ibc, transactions.is_opinit, blocks.height, blocks.timestamp").
		Joins("LEFT JOIN blocks ON transactions.block_height = blocks.height").
		Clauses(clause.OrderBy{
			Columns: []clause.OrderByColumn{
				{Column: clause.Column{Name: "transactions.block_height"}, Desc: pagination.Reverse},
				{Column: clause.Column{Name: "transactions.block_index"}, Desc: pagination.Reverse},
			},
		}).
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query transactions")
		return nil, 0, err
	}

	if pagination.CountTotal {
		if err := r.db.Model(&db.Tracking{}).
			Select("tx_count").
			First(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("Failed to get transaction count")
			return nil, 0, err
		}
	}

	return record, total, nil
}
