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

// GetTxByHash retrieves a transaction by hash using concurrent search
func (r *TxRepository) GetTxByHash(hash string) (*dto.TxByHashResponse, error) {
	ctx := context.Background()

	type bucketResult struct {
		bucketIndex int
		largestName string
		largestNum  int64
		bucket      *blob.Bucket
		err         error
	}

	results := make(chan bucketResult, len(r.buckets))

	// Search all buckets concurrently
	for i, bucket := range r.buckets {
		go func(bucketIndex int, b *blob.Bucket) {
			result := bucketResult{
				bucketIndex: bucketIndex,
				bucket:      b,
			}

			bucketName := "bucket_" + strconv.Itoa(bucketIndex)
			log.Debug().Str("bucket", bucketName).Str("hash", hash).Msg("Starting concurrent search in bucket")

			iter := b.List(&blob.ListOptions{
				Prefix: hash + "/",
			})

			for {
				obj, err := iter.Next(ctx)
				if err != nil {
					if err == io.EOF {
						break // Normal end of iteration
					}

					if strings.Contains(err.Error(), "invalid_grant") {
						log.Warn().Str("bucket", bucketName).Str("hash", hash).Msg("Authentication failure")
						result.err = err
						break
					}

					log.Warn().Err(err).Str("bucket", bucketName).Str("hash", hash).Msg("Error listing objects")
					result.err = err
					break
				}

				parts := strings.Split(obj.Key, "/")
				if len(parts) != 2 {
					continue
				}

				if num, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					if num > result.largestNum {
						result.largestNum = num
						result.largestName = obj.Key
						log.Debug().Str("bucket", bucketName).Str("hash", hash).Int64("block_height", num).Msg("Found newer transaction")
					}
				}
			}

			log.Debug().Str("bucket", bucketName).Str("hash", hash).Int64("largest_block", result.largestNum).Msg("Completed search in bucket")
			results <- result
		}(i, bucket)
	}

	// Collect results from all goroutines
	var finalLargestName string
	var finalLargestNum int64
	var foundBucket *blob.Bucket
	var searchErrors []error

	for i := 0; i < len(r.buckets); i++ {
		result := <-results

		if result.err != nil {
			searchErrors = append(searchErrors, result.err)
			log.Warn().Err(result.err).Int("bucket", result.bucketIndex).Str("hash", hash).Msg("Search failed in bucket")
			continue
		}

		// Check if this bucket has a newer transaction
		if result.largestNum > finalLargestNum {
			finalLargestNum = result.largestNum
			finalLargestName = result.largestName
			foundBucket = result.bucket
			log.Debug().Int("bucket", result.bucketIndex).Str("hash", hash).Int64("block_height", result.largestNum).Msg("New latest transaction found")
		}
	}

	// Check if we found any transaction
	if finalLargestName == "" {
		if len(searchErrors) > 0 {
			log.Warn().Int("error_count", len(searchErrors)).Str("hash", hash).Msg("All bucket searches failed")
		}
		return nil, apperror.NewNoValidTxFiles(hash)
	}

	log.Info().Str("hash", hash).Str("block_height", strings.Split(finalLargestName, "/")[1]).Msg("Found latest transaction across all buckets (concurrent)")

	// Read the transaction data
	tx, err := foundBucket.NewReader(ctx, finalLargestName, nil)
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
