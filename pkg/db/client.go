package db

import (
	"context"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	BatchSize = 100
)

var QueryTimeout = 5 * time.Minute

type ValidatorAddress struct {
	OperatorAddress  string `gorm:"column:operator_address"`
	AccountID        string `gorm:"column:account_id"`
	ConsensusAddress string `gorm:"column:consensus_address"`
}

func NewClient(databaseURL string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(databaseURL), &gorm.Config{DefaultTransactionTimeout: QueryTimeout})
}

func Ping(ctx context.Context, dbClient *gorm.DB) error {
	return dbClient.WithContext(ctx).Exec("SELECT 1").Error
}
