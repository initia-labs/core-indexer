package db

import (
	"time"

	"gorm.io/gorm"
)

// ListWithCount runs a list query (Find into dest) and optionally a count query with timeout.
// listQuery should already have Limit and Offset applied. countQuery should be the same
// base query without Limit/Offset, used only for counting.
// Returns total count (from count query when countTotal is true, else 0) and any error.
func ListWithCount(listQuery, countQuery *gorm.DB, dest interface{}, countTotal bool, timeout time.Duration) (total int64, err error) {
	if err = listQuery.Find(dest).Error; err != nil {
		return 0, err
	}
	if countTotal {
		total, err = CountWithTimeout(countQuery, timeout)
		if err != nil {
			return 0, err
		}
	}
	return total, nil
}
