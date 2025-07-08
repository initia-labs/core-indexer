package db

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/getsentry/sentry-go"
	vmtypes "github.com/initia-labs/movevm/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alleslabs/initia-mono/generic-indexer/common"
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
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
	Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
}

func GetLatestBlockHeight(ctx context.Context, dbClient Queryable) (int64, error) {
	var height int64
	err := QueryRowWithTimeout(ctx, dbClient, "SELECT height FROM blocks ORDER BY height DESC LIMIT 1").Scan(&height)
	if err != nil {
		return 0, err
	}

	return height, nil
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

func GenerateGetAccountsStmt(addresses []string) string {
	return "SELECT address FROM accounts a WHERE address in ('" + strings.Join(addresses, "','") + "')"
}

func GenerateInsertAccountsStmt(notExistAccounts []string, acccountToVmAddress map[string]string) string {
	stmt := "INSERT INTO accounts (address, vm_address_id) VALUES\n"
	notExistAccountsCount := len(notExistAccounts)
	for idx := 0; idx < notExistAccountsCount-1; idx++ {
		stmt += fmt.Sprintf("('%s', '%s'),\n", notExistAccounts[idx], acccountToVmAddress[notExistAccounts[idx]])
	}

	return stmt + fmt.Sprintf("('%s', '%s') ON CONFLICT (address) DO UPDATE SET address  = EXCLUDED.address", notExistAccounts[notExistAccountsCount-1], acccountToVmAddress[notExistAccounts[notExistAccountsCount-1]])
}

func GenerateInsertVmAddressesStmt(notExistVmAddress []string) string {
	stmt := "INSERT INTO vm_addresses (vm_address) VALUES\n"
	notExistAccountsCount := len(notExistVmAddress)
	for idx := 0; idx < notExistAccountsCount-1; idx++ {
		stmt += fmt.Sprintf("('%s'),\n", notExistVmAddress[idx])
	}

	return stmt + fmt.Sprintf("('%s') ON CONFLICT (vm_address) DO NOTHING", notExistVmAddress[notExistAccountsCount-1])
}

func InsertAccounts(ctx context.Context, dbTx Queryable, addresses []string) error {
	span := sentry.StartSpan(ctx, "InsertAccounts")
	span.Description = "Bulk insert accounts into DB"
	defer span.Finish()

	if len(addresses) == 0 {
		return nil
	}

	notExistVmAddress := make([]string, 0)
	accAddressTOVMAddress := make(map[string]string)

	for _, acc := range addresses {
		addr := types.MustAccAddressFromBech32(acc)
		vmAddr, _ := vmtypes.NewAccountAddressFromBytes(addr)
		accAddressTOVMAddress[acc] = vmAddr.String()
		notExistVmAddress = append(notExistVmAddress, vmAddr.String())
	}

	_, err := ExecWithTimeout(
		ctx,
		dbTx,
		GenerateInsertVmAddressesStmt(notExistVmAddress),
	)
	if err != nil {
		return err
	}

	_, err = ExecWithTimeout(
		ctx,
		dbTx,
		GenerateInsertAccountsStmt(addresses, accAddressTOVMAddress),
	)
	if err != nil {
		return err
	}

	return nil
}

func GetAccountsIfNotExist(ctx context.Context, dbTx Queryable, addresses []string) ([]string, error) {
	notExistAccounts := make([]string, 0)

	if len(addresses) == 0 {
		return notExistAccounts, nil
	}

	existAccounts := make(map[string]bool)
	rows, err, cancel := QueryWithTimeout(ctx, dbTx, GenerateGetAccountsStmt(addresses))
	if err != nil {
		return notExistAccounts, err
	}
	defer cancel()
	for rows.Next() {
		var account Account
		err := rows.Scan(&account.Address)

		if err != nil {
			return notExistAccounts, err
		}
		existAccounts[account.Address] = true
	}

	rows.Close()
	if rows.Err() != nil {
		return notExistAccounts, rows.Err()
	}

	for _, address := range addresses {
		if _, ok := existAccounts[address]; !ok {
			notExistAccounts = append(notExistAccounts, address)
		}
	}

	return notExistAccounts, nil
}

func InsertBlockIgnoreConflict(ctx context.Context, dbTx Queryable, block *common.BlockMsg) error {
	var err error
	hashBytes, err := hex.DecodeString(block.Hash)
	if err != nil {
		return ErrorNonRetryable
	}
	if block.Proposer == nil {
		_, err = ExecWithTimeout(ctx, dbTx, "INSERT INTO blocks (height, hash, proposer, timestamp) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
			block.Height, hashBytes, block.Proposer, block.Timestamp,
		)
	} else {
		_, err = ExecWithTimeout(ctx, dbTx, "INSERT INTO blocks (height, hash, proposer, timestamp) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
			block.Height, hashBytes, *block.Proposer, block.Timestamp,
		)
	}

	return err
}

func InsertTransactionsIgnoreConflict(ctx context.Context, dbTx Queryable, transactions []*common.Transaction) error {
	span := sentry.StartSpan(ctx, "InsertTransactions")
	span.Description = "Bulk insert transactions into DB"
	defer span.Finish()

	if len(transactions) == 0 {
		return nil
	}

	columns := []string{
		"id",
		"hash",
		"block_height",
		"block_index",
		"gas_used",
		"gas_limit",
		"gas_fee",
		"err_msg",
		"success",
		"sender",
		"memo",
		"messages",
		"is_ibc",
		"is_send",
		"is_move_publish",
		"is_move_execute_event",
		"is_move_execute",
		"is_move_upgrade",
		"is_move_script",
		"is_nft_transfer",
		"is_nft_mint",
		"is_nft_burn",
		"is_collection_create",
		"is_opinit",
		"is_instantiate",
		"is_migrate",
		"is_update_admin",
		"is_clear_admin",
		"is_store_code",
	}

	var values [][]interface{}
	for _, tx := range transactions {
		values = append(values, []interface{}{
			fmt.Sprintf("%X/%d", tx.Hash, tx.BlockHeight),
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
			tx.IsIBC,
			tx.IsSend,
			tx.IsMovePublish,
			tx.IsMoveExecuteEvent,
			tx.IsMoveExecute,
			tx.IsMoveUpgrade,
			tx.IsMoveScript,
			tx.IsNFTTransfer,
			tx.IsNFTMint,
			tx.IsNFTBurn,
			tx.IsCollectionCreate,
			tx.IsOPInit,
			tx.IsInstantiate,
			tx.IsMigrate,
			tx.IsUpdateAdmin,
			tx.IsClearAdmin,
			tx.IsStoreCode,
		})
	}

	err := BulkInsert(ctx, dbTx, "transactions", columns, values, "ON CONFLICT DO NOTHING")
	if err != nil {
		return err
	}

	return err
}

func UpsertValidators(
	ctx context.Context,
	dbTx Queryable,
	vals *[]Validator,
) error {
	for _, v := range *vals {
		_, err := ExecWithTimeout(ctx, dbTx, "INSERT INTO validators (account_id, operator_address, consensus_address, moniker, identity, website, details, commission_rate, commission_max_rate, commission_max_change, jailed, is_active, consensus_pubkey, voting_power, voting_powers) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)ON CONFLICT (operator_address) DO UPDATE SET (moniker,identity,website,details,commission_rate,commission_max_change,jailed,is_active,voting_power,voting_powers) = (excluded.moniker,excluded.identity,excluded.website,excluded.details,excluded.commission_rate,excluded.commission_max_change,excluded. jailed,excluded.is_active,excluded.voting_power,excluded.voting_powers)",
			v.AccountId,
			v.OperatorAddress,
			v.ConsensusAddress,
			v.Moniker,
			v.Identity,
			v.Website,
			v.Details,
			v.CommissionRate,
			v.CommissionMaxRate,
			v.CommissionMaxChange,
			v.Jailed,
			v.IsActive,
			v.ConsensusPubkey,
			v.VotingPower,
			v.VotingPowers,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
func QueryValidatorRelations(ctx context.Context, dbTx Queryable) ([]ValidatorRelation, error) {
	vals := make([]ValidatorRelation, 0)
	// Retrieve all validator records from the database.
	rows, err, cancel := QueryWithTimeout(ctx, dbTx, "SELECT operator_address, consensus_address FROM validators")
	if err != nil {
		return vals, err
	}
	defer cancel()
	defer rows.Close()

	for rows.Next() {
		var val ValidatorRelation
		err := rows.Scan(&val.OperatorAddress, &val.ConsensusAddress)
		if err != nil {
			return vals, err
		}
		vals = append(vals, val)
	}

	if rows.Err() != nil {
		return vals, rows.Err()
	}

	return vals, nil
}

func QueryLatestValidatorVoteSignature(ctx context.Context, dbClient Queryable) (int64, error) {
	var height int64
	err := QueryRowWithTimeout(ctx, dbClient, "SELECT block_height FROM validator_commit_signatures ORDER BY block_height DESC LIMIT 1").Scan(&height)
	if err != nil {
		return 0, err
	}

	return height, nil
}

func QueryBlockProposers(ctx context.Context, dbTx Queryable) (map[int64]string, error) {
	proposers := make(map[int64]string)
	rows, err, cancel := QueryWithTimeout(
		ctx,
		dbTx,
		"SELECT validator_address, block_height  FROM validator_commit_signatures WHERE validator_commit_signatures.vote = 'PROPOSE' ORDER BY block_height DESC LIMIT 101",
	)
	defer cancel()
	if err != nil {
		return proposers, err
	}

	for rows.Next() {
		var validatorAddress string
		var height int64
		err := rows.Scan(&validatorAddress, &height)
		if err != nil {
			return proposers, err
		}

		proposers[height] = validatorAddress
	}

	rows.Close()
	if rows.Err() != nil {
		return proposers, rows.Err()
	}
	return proposers, nil
}

func DeleteValidatorCommitSignatures(ctx context.Context, dbTx Queryable, height int64) error {
	span := sentry.StartSpan(ctx, "DeleteValidatorCommitSignatures")
	span.Description = "Delete validator commit signatures from DB"
	defer span.Finish()

	_, err := ExecWithTimeout(
		ctx,
		dbTx,
		"DELETE FROM validator_commit_signatures WHERE block_height < $1",
		height,
	)
	return err
}

// FetchValidatorCommitSignatures fetches validator commit signatures from the database.
func FetchValidatorCommitSignatures(ctx context.Context, dbTx Queryable, height, lookbackBlocks int64) ([]ValidatorVote, error) {
	span := sentry.StartSpan(ctx, "FetchValidatorCommitSignatures")
	span.Description = "Fetch validator commit signatures from DB"
	defer span.Finish()

	votes := make([]ValidatorVote, 0)
	query := `
		SELECT validator_address, vote, block_height
		FROM validator_commit_signatures
		WHERE block_height < $1 AND block_height >= $2
	`
	rows, err, cancel := QueryWithTimeout(ctx, dbTx, query, height, height-lookbackBlocks)
	if err != nil {
		return votes, err
	}
	defer rows.Close()
	defer cancel()

	for rows.Next() {
		var vv ValidatorVote
		if err := rows.Scan(&vv.ValidatorAddress, &vv.Vote, &vv.Height); err != nil {
			return votes, err
		}
		votes = append(votes, vv)
	}

	if err := rows.Err(); err != nil {
		return votes, err
	}

	return votes, nil
}

// truncateTable truncates the specified table in the database.
func TruncateTable(ctx context.Context, dbTx Queryable, tableName string) error {
	_, err := ExecWithTimeout(ctx, dbTx, fmt.Sprintf("TRUNCATE %s CASCADE", tableName))
	return err
}

// insertValidatorUptimes inserts the calculated validator uptimes into the database.
func InsertValidatorUptimes(ctx context.Context, dbTx Queryable, validatorUptimes []ValidatorUptime) error {
	span := sentry.StartSpan(ctx, "InsertValidatorUptimes")
	span.Description = "Insert validator uptimes into DB"
	defer span.Finish()

	const queryTemplate = `
		INSERT INTO validator_vote_counts (validator_address, last_100)
		VALUES %s
		ON CONFLICT (validator_address) DO UPDATE SET last_100 = EXCLUDED.last_100
	`
	valueStrings := make([]string, len(validatorUptimes))
	for i, vu := range validatorUptimes {
		valueStrings[i] = vu.String()
	}
	query := fmt.Sprintf(queryTemplate, strings.Join(valueStrings, ",\n"))
	_, err := ExecWithTimeout(ctx, dbTx, query)
	return err
}

func InsertValidatorVote(ctx context.Context, dbTx Queryable, valId int, height int64, vote BlockVote) error {
	_, err := ExecWithTimeout(
		ctx,
		dbTx,
		"INSERT INTO validator_votes (val_id, height, vote) VALUES ($1, $2, $3)",
		valId,
		height,
		vote,
	)
	return err
}

func InsertValidatorCommitSignatures(ctx context.Context, dbTx Queryable, votes *[]ValidatorCommitSignatures) error {
	if len(*votes) == 0 {
		return nil
	}
	stmt := "INSERT INTO validator_commit_signatures (validator_address, block_height, vote) VALUES\n"
	voteCount := len(*votes)
	for idx := 0; idx < voteCount-1; idx++ {
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

func InsertHistoricalVotingPowers(ctx context.Context, dbTx Queryable, hps []ValidatorHistoricalPower) error {
	stmt := "INSERT INTO validator_historical_powers (validator_address, tokens, voting_power, hour_rounded_timestamp, timestamp) VALUES\n"
	count := len(hps)
	for idx := 0; idx < count-1; idx++ {
		stmt += fmt.Sprintf("%s,", hps[idx].String())
	}
	stmt += hps[count-1].String()
	_, err := ExecWithTimeout(
		ctx,
		dbTx,
		stmt,
	)
	return err
}

func GenerateGetAccountTransactionsStmt(accountTransactions []AccountTransaction) string {
	stmt := "INSERT INTO account_transactions (is_signer, block_height, transaction_id, account_id) VALUES\n"
	values := []string{}

	for _, at := range accountTransactions {
		for _, account := range at.Accounts {
			isSigner := account == at.Signer
			values = append(values, fmt.Sprintf("(%t, %d, '%s', '%s')", isSigner, at.BlockHeight, at.TxId, account))
		}
	}
	stmt += strings.Join(values, ",\n")
	stmt += " ON CONFLICT DO NOTHING"
	return stmt
}

func InsertAccountTransactions(ctx context.Context, dbTx Queryable, accountTransactions []AccountTransaction) error {
	span := sentry.StartSpan(ctx, "InsertAccountTransactions")
	span.Description = "Bulk insert account transactions into DB"
	defer span.Finish()

	if len(accountTransactions) == 0 {
		return nil
	}

	_, err := ExecWithTimeout(
		ctx,
		dbTx,
		GenerateGetAccountTransactionsStmt(accountTransactions),
	)
	return err
}
