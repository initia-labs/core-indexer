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

// GetCollections retrieves NFT collections with pagination and search
func (r *txRepository) GetTxByHash(hash string) (*dto.RestTxResponse, error) {
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

	txResponse := &dto.RestTxResponse{}
	err = json.NewDecoder(tx).Decode(txResponse)
	if err != nil {
		return nil, err
	}
	return txResponse, nil
}
