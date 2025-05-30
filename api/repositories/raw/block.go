package raw

import (
	"database/sql"
	"time"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

type blockRepository struct {
	db *sql.DB
}

func NewBlockRepository(db *sql.DB) repositories.BlockRepository {
	return &blockRepository{
		db: db,
	}
}

func (r *blockRepository) GetBlockHeightLatest() (*dto.RestBlockHeightLatestResponse, error) {
	query := `
		SELECT latest_informative_block_height FROM tracking
		LIMIT 1
	`

	var latestBlockHeight int64

	err := r.db.QueryRow(query).Scan(&latestBlockHeight)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query tracking data")
		return nil, err
	}

	return &dto.RestBlockHeightLatestResponse{
		Height: latestBlockHeight,
	}, nil
}

func (r *blockRepository) GetBlockTimestamp(latestBlockHeight *int64) ([]time.Time, error) {
	query := `
		SELECT timestamp FROM blocks
		WHERE height <= $1
		ORDER BY height DESC
		LIMIT 100
	`

	rows, err := r.db.Query(query, *latestBlockHeight)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query block timestamps")
		return nil, err
	}
	defer rows.Close()

	var timestamps []time.Time
	for rows.Next() {
		var ts time.Time
		if err := rows.Scan(&ts); err != nil {
			logger.Get().Error().Err(err).Msg("Failed to scan timestamp")
			return nil, err
		}
		timestamps = append(timestamps, ts)
	}

	if err := rows.Err(); err != nil {
		logger.Get().Error().Err(err).Msg("Error iterating timestamps")
		return nil, err
	}

	return timestamps, nil
}
