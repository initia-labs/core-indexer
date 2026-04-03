package db

import (
	"context"

	"github.com/getsentry/sentry-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func InsertAccountsIgnoreConflict(ctx context.Context, dbTx *gorm.DB, accounts []Account) error {
	span := sentry.StartSpan(ctx, "InsertAccount")
	span.Description = "Bulk insert accounts into the database"
	defer span.Finish()

	if len(accounts) == 0 {
		return nil
	}
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(&accounts, BatchSize)
	return result.Error
}

func InsertVMAddressesIgnoreConflict(ctx context.Context, dbTx *gorm.DB, addresses []VMAddress) error {
	span := sentry.StartSpan(ctx, "InsertVMAddress")
	span.Description = "Bulk insert VM addresses into the database"
	defer span.Finish()

	if len(addresses) == 0 {
		return nil
	}
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).CreateInBatches(&addresses, BatchSize)
	return result.Error
}

func GetAccountOrInsertIfNotExist(ctx context.Context, dbTx *gorm.DB, address string, vmAddress string) error {
	var account Account
	result := dbTx.WithContext(ctx).
		Table(TableNameAccount).
		Where("address = ?", address).
		First(&account)

	if result.Error == gorm.ErrRecordNotFound {
		vmAddr := VMAddress{VMAddress: vmAddress}
		if err := dbTx.WithContext(ctx).
			Clauses(clause.OnConflict{
				DoNothing: true,
			}).
			Create(&vmAddr).Error; err != nil {
			return err
		}
		newAccount := Account{
			Address:     address,
			VMAddressID: vmAddress,
			Type:        string(BaseAccount),
		}
		if err := dbTx.WithContext(ctx).
			Clauses(clause.OnConflict{
				DoNothing: true,
			}).
			Create(&newAccount).Error; err != nil {
			return err
		}
	} else if result.Error != nil {
		return result.Error
	}
	return nil
}
