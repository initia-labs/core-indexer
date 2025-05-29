package raw

import (
	"database/sql"

	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

type txRepository struct {
	db *sql.DB
}

func NewTxRepository(db *sql.DB) repositories.TxRepository {
	return &txRepository{
		db: db,
	}
}

func (r *txRepository) GetTxCount() (*int64, error) {
	query := `
		SELECT tx_count FROM tracking
		LIMIT 1
	`

	var txCount *int64

	err := r.db.QueryRow(query).Scan(&txCount)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query tracking data for transaction count")
		return nil, err
	}

	return txCount, nil
}
