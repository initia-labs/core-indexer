package db

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// CountWithTimeout executes a count query with a timeout.
// If the query times out, it returns -1 for the count with no error.
// Returns (count int64, error).
func CountWithTimeout(query *gorm.DB, timeout time.Duration) (int64, error) {
	var total int64

	// Use a transaction with statement_timeout to avoid connection corruption
	if err := query.Transaction(func(tx *gorm.DB) error {
		// Set timeout only for this transaction
		if err := tx.Exec(fmt.Sprintf("SET LOCAL statement_timeout = '%dms'", timeout.Milliseconds())).Error; err != nil {
			return err
		}

		countErr := tx.Count(&total).Error

		// Reset timeout before committing to prevent leakage to parent transaction
		if resetErr := tx.Exec("RESET statement_timeout").Error; resetErr != nil {
			// Log the reset error but don't override the count error
			if countErr == nil {
				return resetErr
			}
		}

		return countErr
	}); err != nil {
		// Check for statement timeout
		if strings.Contains(err.Error(), "statement timeout") {
			return -1, nil
		}
		return 0, err
	}

	return total, nil
}
