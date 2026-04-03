package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

func TruncateTable(ctx context.Context, dbTx *gorm.DB, table string) error {
	return dbTx.WithContext(ctx).Exec(fmt.Sprintf("TRUNCATE TABLE %s", table)).Error
}
