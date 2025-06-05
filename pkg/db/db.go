package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	BatchSize = 100
)

var (
	QueryTimeout = 5 * time.Minute
)

func NewClient(databaseURL string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(databaseURL), &gorm.Config{DefaultTransactionTimeout: QueryTimeout})
}

func GetLatestBlockHeight(ctx context.Context, dbClient *gorm.DB) (int64, error) {
	var height int64

	result := dbClient.WithContext(ctx).
		Table(TableNameFinalizeBlockEvent).
		Select("block_height").
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Table: TableNameFinalizeBlockEvent,
				Name:  "block_height",
			},
			Desc: true,
		}).
		Limit(1).
		Scan(&height)

	if result.Error != nil {
		// Handle no rows found in `transaction_events`
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Fallback to the latest indexed height from `blocks`
			result = dbClient.WithContext(ctx).
				Table(TableNameBlock).
				Select("height").
				Order(clause.OrderByColumn{
					Column: clause.Column{
						Table: TableNameBlock,
						Name:  "height",
					},
					Desc: true,
				}).
				Limit(1).
				Scan(&height)

			if result.Error == nil {
				return height, nil
			}
		}
		return 0, fmt.Errorf("failed to get latest block height: %w", result.Error)
	}

	return height, nil
}

func InsertAccountIgnoreConflict(ctx context.Context, dbTx *gorm.DB, accounts []Account) error {
	span := sentry.StartSpan(ctx, "InsertAccount")
	span.Description = "Bulk insert accounts into the database"
	defer span.Finish()

	if len(accounts) == 0 {
		return nil
	}

	// Use GORM's CreateInBatches with on conflict do nothing
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(&accounts, BatchSize)

	return result.Error
}

func InsertVMAddressIgnoreConflict(ctx context.Context, dbTx *gorm.DB, addresses []VMAddress) error {
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

func InsertValidatorIgnoreConflict(ctx context.Context, dbTx *gorm.DB, validators []Validator) error {
	span := sentry.StartSpan(ctx, "InsertValidator")
	span.Description = "Bulk insert validators into the database"
	defer span.Finish()

	if len(validators) == 0 {
		return nil
	}

	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(&validators, BatchSize)

	return result.Error
}

func InsertValidatorBondedTokenChangesIgnoreConflict(ctx context.Context, dbTx *gorm.DB, txs []ValidatorBondedTokenChange) error {
	span := sentry.StartSpan(ctx, "InsertValidatorBondedTokenChanges")
	span.Description = "Bulk insert validator_bonded_token_changes into the database"
	defer span.Finish()

	if len(txs) == 0 {
		return nil
	}

	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(&txs, BatchSize)

	return result.Error
}

func InsertTransactionIgnoreConflict(ctx context.Context, dbTx *gorm.DB, txs []Transaction) error {
	span := sentry.StartSpan(ctx, "InsertTransaction")
	span.Description = "Bulk insert transactions into the database"
	defer span.Finish()

	if len(txs) == 0 {
		return nil
	}

	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(txs, BatchSize)

	return result.Error
}

func InsertAccountTxsIgnoreConflict(ctx context.Context, dbTx *gorm.DB, txs []AccountTransaction) error {
	span := sentry.StartSpan(ctx, "InsertAccountTxs")
	span.Description = "Bulk insert account_txs into the database"
	defer span.Finish()

	if len(txs) == 0 {
		return nil
	}

	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(txs, BatchSize)

	return result.Error
}

func InsertTransactionEventsIgnoreConflict(ctx context.Context, dbTx *gorm.DB, txEvents []TransactionEvent) error {
	span := sentry.StartSpan(ctx, "InsertTransactionEvents")
	span.Description = "Bulk insert transaction_events into the database"
	defer span.Finish()

	if len(txEvents) == 0 {
		return nil
	}

	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(txEvents, BatchSize)

	return result.Error
}

func InsertMoveEventsIgnoreConflict(ctx context.Context, dbTx *gorm.DB, moveEvents []*MoveEvent) error {
	span := sentry.StartSpan(ctx, "InsertMoveEvents")
	span.Description = "Bulk insert move_events into the database"
	defer span.Finish()

	if len(moveEvents) == 0 {
		return nil
	}

	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(moveEvents, BatchSize)

	return result.Error
}

func InsertFinalizeBlockEventsIgnoreConflict(ctx context.Context, dbTx *gorm.DB, blockEvents []*FinalizeBlockEvent) error {
	span := sentry.StartSpan(ctx, "InsertFinalizeBlockEvents")
	span.Description = "Bulk insert finalize_block_events into the database"
	defer span.Finish()

	if len(blockEvents) == 0 {
		return nil
	}

	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(blockEvents, BatchSize)

	return result.Error
}

func GetRowCount(ctx context.Context, dbClient *gorm.DB, table string) (int64, error) {
	if !isValidTableName(table) {
		return 0, fmt.Errorf("invalid table name: %s", table)
	}

	var count int64
	result := dbClient.WithContext(ctx).
		Table(table).
		Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("failed to get row count for table %s: %w", table, result.Error)
	}

	return count, nil
}

func BuildPruneQuery(ctx context.Context, dbClient *gorm.DB, table string, threshold int64) (*gorm.DB, error) {
	if !isValidTableName(table) {
		return nil, fmt.Errorf("invalid table name: %s", table)
	}

	query := dbClient.WithContext(ctx).
		Table(table).
		Where("block_height <= ?", threshold)

	return query, nil
}

func DeleteRowsToPrune(ctx context.Context, dbClient *gorm.DB, table string, threshold int64) error {
	if !isValidTableName(table) {
		return fmt.Errorf("invalid table name: %s", table)
	}

	result := dbClient.WithContext(ctx).
		Table(table).
		Where("block_height <= ?", threshold).
		Delete(nil)

	return result.Error
}

func GetOperatorAddress(ctx context.Context, dbClient *gorm.DB, consensusAddress string) (*string, error) {
	var operatorAddress string
	result := dbClient.WithContext(ctx).
		Table(TableNameValidator).
		Select("operator_address").
		Where("consensus_address = ?", consensusAddress).
		Scan(&operatorAddress)
	if result.Error != nil {
		return nil, result.Error
	}

	return &operatorAddress, nil
}

func GetAccountOrInsertIfNotExist(ctx context.Context, dbTx *gorm.DB, address string, vmAddress string) error {
	var account Account
	result := dbTx.WithContext(ctx).
		Table(TableNameAccount).
		Where("address = ?", address).
		First(&account)

	if result.Error == gorm.ErrRecordNotFound {
		// First insert the VM address
		vmAddr := VMAddress{VMAddress: vmAddress}
		if err := dbTx.WithContext(ctx).
			Clauses(clause.OnConflict{
				DoNothing: true,
			}).
			Create(&vmAddr).Error; err != nil {
			return err
		}

		// Then insert the account
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

// InsertValidatorCommitSignatureForProposer inserts a validator commit signature for a proposer
func InsertValidatorCommitSignatureForProposer(ctx context.Context, dbTx *gorm.DB, val string, height int64) error {
	signature := ValidatorCommitSignature{
		ValidatorAddress: val,
		BlockHeight:      height,
		Vote:             string(Propose),
	}

	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "validator_address"}, {Name: "block_height"}},
			DoUpdates: clause.Assignments(map[string]any{"vote": string(Propose)}),
		}).
		Create(&signature)

	return result.Error
}

func InsertValidatorCommitSignatures(ctx context.Context, dbTx *gorm.DB, votes *[]ValidatorCommitSignature) error {
	if len(*votes) == 0 {
		return nil
	}

	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
			Columns:   []clause.Column{{Name: "validator_address"}, {Name: "block_height"}},
		}).
		CreateInBatches(votes, BatchSize)

	return result.Error
}
