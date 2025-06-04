package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	QueryTimeout = 5 * time.Minute

	ErrorNonRetryable   = errors.New("non-retryable error")
	ErrorLengthMismatch = errors.New("length mismatch")
)

func NewClient(databaseURL string) (*pgxpool.Pool, error) {
	return pgxpool.New(context.Background(), databaseURL)
}

type Queryable interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
}

func GetLatestBlockHeight(ctx context.Context, dbClient Queryable) (int64, error) {
	var height int64
	err := QueryRowWithTimeout(ctx, dbClient, "SELECT block_height FROM finalize_block_events ORDER BY block_height DESC LIMIT 1").Scan(&height)
	if err != nil {
		// Handle no rows found in `transaction_events`
		if errors.Is(err, pgx.ErrNoRows) {
			// Fallback to the latest indexed height from `blocks`
			err = QueryRowWithTimeout(ctx, dbClient, "SELECT height FROM blocks ORDER BY height DESC LIMIT 1").Scan(&height)
			if err == nil {
				return height, nil
			}
		}
		return 0, fmt.Errorf("failed to get latest block height: %w", err)
	}

	return height, nil
}

func InsertAccountIgnoreConflict(ctx context.Context, dbTx Queryable, accounts []Account) error {
	span := sentry.StartSpan(ctx, "InsertAccount")
	span.Description = "Bulk insert accounts into the database"
	defer span.Finish()

	if len(accounts) == 0 {
		return nil
	}

	columns := getColumns[Account]()
	var values [][]any
	for _, account := range accounts {
		values = append(values, []any{
			account.Address,
			account.VMAddress,
		})
	}

	return BulkInsert(ctx, dbTx, "accounts", columns, values, "ON CONFLICT DO NOTHING")
}

func InsertValidatorIgnoreConflict(ctx context.Context, dbTx Queryable, validators []Validator) error {
	span := sentry.StartSpan(ctx, "InsertValidator")
	span.Description = "Bulk insert validators into the database"
	defer span.Finish()

	if len(validators) == 0 {
		return nil
	}

	columns := getColumns[Validator]()
	var values [][]any
	for _, validator := range validators {
		values = append(values, []any{
			validator.AccountId,
			validator.OperatorAddress,
			validator.ConsensusAddress,
			validator.Moniker,
			validator.Identity,
			validator.Website,
			validator.Details,
			validator.CommissionRate,
			validator.CommissionMaxRate,
			validator.CommissionMaxChange,
			validator.Jailed,
			validator.IsActive,
			validator.ConsensusPubkey,
			validator.VotingPower,
			validator.VotingPowers,
		})
	}

	return BulkInsert(ctx, dbTx, "validators", columns, values, "ON CONFLICT DO NOTHING")
}

func InsertValidatorBondedTokenChangesIgnoreConflict(ctx context.Context, dbTx Queryable, txs []ValidatorBondedTokenChange) error {
	span := sentry.StartSpan(ctx, "InsertValidatorBondedTokenChanges")
	span.Description = "Bulk insert validator_bonded_token_changes into the database"
	defer span.Finish()

	if len(txs) == 0 {
		return nil
	}

	columns := getColumns[ValidatorBondedTokenChange]()
	var values [][]any
	for _, tx := range txs {
		values = append(values, []any{
			tx.ValidatorAddr,
			tx.TxId,
			tx.BlockHeight,
			tx.Tokens,
		})
	}

	return BulkInsert(ctx, dbTx, "validator_bonded_token_changes", columns, values, "ON CONFLICT DO NOTHING")
}

func InsertTransactionIgnoreConflict(ctx context.Context, dbTx Queryable, txs []*Transaction) error {
	span := sentry.StartSpan(ctx, "InsertTransaction")
	span.Description = "Bulk insert transactions into the database"
	defer span.Finish()

	if len(txs) == 0 {
		return nil
	}

	columns := getColumns[Transaction]()
	var values [][]any
	for _, tx := range txs {
		values = append(values, []any{
			tx.ID,
			tx.Hash,
			tx.BlockHeight,
			tx.BlockIndex,
			tx.GasUsed,
			tx.GasLimit,
			tx.GasFee,
			tx.ErrMsg,
			tx.Success,
			tx.Sender,
			tx.Memo,
			tx.Messages,
		})
	}

	return BulkInsert(ctx, dbTx, "transactions", columns, values, "ON CONFLICT DO NOTHING")
}

func InsertAccountTxsIgnoreConflict(ctx context.Context, dbTx Queryable, txs []AccountTx) error {
	span := sentry.StartSpan(ctx, "InsertAccountTxs")
	span.Description = "Bulk insert account_txs into the database"
	defer span.Finish()

	if len(txs) == 0 {
		return nil
	}

	columns := getColumns[AccountTx]()
	var values [][]any
	for _, tx := range txs {
		values = append(values, []any{
			tx.TxId,
			tx.BlockHeight,
			tx.Account,
			tx.IsSigner,
		})
	}

	return BulkInsert(ctx, dbTx, "account_transactions", columns, values, "ON CONFLICT DO NOTHING")
}

func InsertTransactionEventsIgnoreConflict(ctx context.Context, dbTx Queryable, txEvents []*TransactionEvent) error {
	span := sentry.StartSpan(ctx, "InsertTransactionEvents")
	span.Description = "Bulk insert transaction_events into the database"
	defer span.Finish()

	if len(txEvents) == 0 {
		return nil
	}

	columns := getColumns[TransactionEvent]()
	var values [][]any
	for _, txEvent := range txEvents {
		values = append(values, []any{
			txEvent.TransactionHash,
			txEvent.BlockHeight,
			txEvent.EventKey,
			txEvent.EventValue,
			txEvent.EventIndex,
		})
	}

	return BulkInsert(ctx, dbTx, "transaction_events", columns, values, "ON CONFLICT DO NOTHING")
}

func InsertMoveEventsIgnoreConflict(ctx context.Context, dbTx Queryable, moveEvents []*MoveEvent) error {
	span := sentry.StartSpan(ctx, "InsertMoveEvents")
	span.Description = "Bulk insert move_events into the database"
	defer span.Finish()

	if len(moveEvents) == 0 {
		return nil
	}

	columns := getColumns[MoveEvent]()
	var values [][]any
	for _, moveEvent := range moveEvents {
		values = append(values, []any{
			moveEvent.TypeTag,
			moveEvent.Data,
			moveEvent.BlockHeight,
			moveEvent.TransactionHash,
			moveEvent.EventIndex,
		})
	}

	return BulkInsert(ctx, dbTx, "move_events", columns, values, "ON CONFLICT DO NOTHING")
}

func InsertFinalizeBlockEventsIgnoreConflict(ctx context.Context, dbTx Queryable, blockEvents []*FinalizeBlockEvent) error {
	span := sentry.StartSpan(ctx, "InsertFinalizeBlockEvents")
	span.Description = "Bulk insert finalize_block_events into the database"
	defer span.Finish()

	if len(blockEvents) == 0 {
		return nil
	}

	columns := getColumns[FinalizeBlockEvent]()
	var values [][]any
	for _, blockEvent := range blockEvents {
		values = append(values, []any{
			blockEvent.BlockHeight,
			blockEvent.EventKey,
			blockEvent.EventValue,
			blockEvent.EventIndex,
			blockEvent.Mode,
		})
	}

	return BulkInsert(ctx, dbTx, "finalize_block_events", columns, values, "ON CONFLICT DO NOTHING")
}

func GetRowCount(ctx context.Context, dbClient Queryable, table string) (int64, error) {
	if !isValidTableName(table) {
		return 0, fmt.Errorf("invalid table name: %s", table)
	}

	var count int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	err := QueryRowWithTimeout(ctx, dbClient, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get row count for table %s: %w", table, err)
	}
	return count, nil
}

func GetRowsToPruneByBlockHeight(ctx context.Context, dbClient Queryable, table string, threshold int64) (pgx.Rows, error) {
	columns, err := GetColumnsFromValidTable(table)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns from valid table: %w", err)
	}
	query := fmt.Sprintf("SELECT %s FROM %s WHERE block_height <= $1", strings.Join(columns, ", "), table)
	rows, err := QueryRowsWithTimeout(ctx, dbClient, query, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows to prune from table %s: %w", table, err)
	}
	return rows, err
}

func DeleteRowsToPrune(ctx context.Context, dbClient Queryable, table string, threshold int64) error {
	if !isValidTableName(table) {
		return fmt.Errorf("invalid table name: %s", table)
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE block_height <= $1", table)

	_, err := ExecWithTimeout(ctx, dbClient, query, threshold)
	return err
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
