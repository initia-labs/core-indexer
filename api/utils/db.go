package utils

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// CountWithTimeout executes a count query with a timeout.
// If the query times out, it logs a warning and returns -1 for the count.
// Returns (count, isTimeout, error)
func CountWithTimeout(ctx context.Context, query *gorm.DB, timeout time.Duration) (int64, error) {
	var total int64

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := query.WithContext(ctx).Count(&total).Error; err != nil {
		// Check if the error is a timeout
		if errors.Is(err, context.DeadlineExceeded) {
			return -1, nil
		}
		return 0, err
	}

	return total, nil
}
