package repositories

import "time"

type BlockRepository interface {
	GetBlockHeightLatest() (*int64, error)
	GetBlockTimestamp(latestBlockHeight *int64) ([]time.Time, error)
}
