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

func GetOperatorAddress(ctx context.Context, dbClient Queryable, consensusAddress string) (*string, error) {
	var operatorAddress string
	err := QueryRowWithTimeout(ctx, dbClient, "SELECT operator_address FROM validators WHERE consensus_address = $1", consensusAddress).Scan(&operatorAddress)
	if err != nil {
		return nil, err
	}

	return &operatorAddress, nil
}

func GetAccountOrInsertIfNotExist(ctx context.Context, dbTx Queryable, address string, vmAddress string) error {
	err := QueryRowWithTimeout(ctx, dbTx, "SELECT address FROM accounts WHERE address = $1", address).Scan(&address)
	if err == pgx.ErrNoRows {
		_, err = ExecWithTimeout(ctx, dbTx, "INSERT INTO vm_addresses (vm_address) VALUES ($1) ON CONFLICT DO NOTHING", vmAddress)
		if err != nil {
			return err
		}
		_, err = ExecWithTimeout(ctx, dbTx, "INSERT INTO accounts (address, vm_address_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", address, vmAddress)
		if err != nil {
			return err
		}

	} else if err != nil {
		return err
	}

	return nil
}

// TODO: use bulk insert
func InsertValidatorCommitSignatureForProposer(ctx context.Context, dbTx Queryable, val string, height int64) error {
	_, err := ExecWithTimeout(
		ctx,
		dbTx,
		"INSERT INTO validator_commit_signatures (validator_address, block_height, vote) VALUES ($1, $2, 'PROPOSE') ON CONFLICT (validator_address, block_height) DO UPDATE SET vote = 'PROPOSE'",
		val,
		height,
	)
	return err
}

// TODO: use bulk insert
func InsertValidatorCommitSignatures(ctx context.Context, dbTx Queryable, votes *[]ValidatorCommitSignatures) error {
	if len(*votes) == 0 {
		return nil
	}
	stmt := "INSERT INTO validator_commit_signatures (validator_address, block_height, vote) VALUES\n"
	voteCount := len(*votes)
	for idx := range voteCount - 1 {
		stmt += fmt.Sprintf("%s,\n", (*votes)[idx].String())
	}
	stmt += fmt.Sprintf("%s ON CONFLICT (validator_address, block_height) DO NOTHING", (*votes)[voteCount-1].String())
	_, err := ExecWithTimeout(
		ctx,
		dbTx,
		stmt,
	)

	return err
}
