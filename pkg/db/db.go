package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/getsentry/sentry-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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



func InsertGenesisBlock(ctx context.Context, dbTx *gorm.DB, timestamp time.Time) error {
	err := dbTx.WithContext(ctx).Exec(`
		INSERT INTO blocks (height, hash, timestamp, proposer) 
		VALUES (?, ?, ?, ?) 
		ON CONFLICT DO NOTHING
	`, 0, []byte("GENESIS"), timestamp, nil).Error
	if err != nil {
		return err
	}

	return nil
}
func InsertBlockIgnoreConflict(ctx context.Context, dbTx *gorm.DB, block Block) error {
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		Create(&block)

	return result.Error
}

func UpsertBlock(ctx context.Context, dbTx *gorm.DB, block Block) error {
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "height"}},
			DoUpdates: clause.AssignmentColumns([]string{"proposer"}),
		}).
		Create(&block)

	return result.Error
}

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

func InsertValidatorsIgnoreConflict(ctx context.Context, dbTx *gorm.DB, validators []Validator) error {
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

func UpsertValidators(ctx context.Context, dbTx *gorm.DB, validators []Validator) error {
	if len(validators) == 0 {
		return nil
	}

	// List all columns you want to update on conflict, except the primary key
	columns := []string{
		"consensus_address",
		"voting_powers",
		"voting_power",
		"moniker",
		"identity",
		"website",
		"details",
		"commission_rate",
		"commission_max_rate",
		"commission_max_change",
		"jailed",
		"is_active",
		"consensus_pubkey",
		"account_id",
	}

	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "operator_address"}},
			DoUpdates: clause.AssignmentColumns(columns),
		}).
		CreateInBatches(&validators, BatchSize)

	return result.Error
}

func QueryValidatorAddresses(dbClient *gorm.DB) ([]ValidatorAddress, error) {
	var validators []ValidatorAddress
	result := dbClient.
		Table(TableNameValidator).
		Select("operator_address, account_id, consensus_address").
		Scan(&validators)

	if result.Error != nil {
		return nil, result.Error
	}

	return validators, nil
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

func InsertProposalsIgnoreConflict(ctx context.Context, dbTx *gorm.DB, proposals []Proposal) error {
	span := sentry.StartSpan(ctx, "InsertProposalsIgnoreConflict")
	span.Description = "Insert proposals into the database"
	defer span.Finish()

	if len(proposals) == 0 {
		return nil
	}

	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(&proposals, BatchSize)

	return result.Error
}

func UpdateProposalStatus(ctx context.Context, dbTx *gorm.DB, proposals []Proposal) error {
	span := sentry.StartSpan(ctx, "UpdateProposalStatus")
	span.Description = "Bulk update proposals into the database"
	defer span.Finish()

	if len(proposals) == 0 {
		return nil
	}

	for _, proposal := range proposals {
		result := dbTx.WithContext(ctx).
			Model(&Proposal{}).
			Where("id = ?", proposal.ID).
			Updates(map[string]any{
				"status":                proposal.Status,
				"voting_time":           proposal.VotingTime,
				"voting_end_time":       proposal.VotingEndTime,
				"resolved_height":       proposal.ResolvedHeight,
				"abstain":               proposal.Abstain,
				"yes":                   proposal.Yes,
				"no":                    proposal.No,
				"no_with_veto":          proposal.NoWithVeto,
				"resolved_voting_power": proposal.ResolvedVotingPower,
				"is_expedited":          proposal.IsExpedited,
			})
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}

func UpdateProposalEmergencyNextTally(ctx context.Context, dbTx *gorm.DB, proposals map[int32]*time.Time) error {
	if len(proposals) == 0 {
		return nil
	}

	for proposal, nextTallyTime := range proposals {
		result := dbTx.WithContext(ctx).
			Model(&Proposal{}).
			Where("id = ?", proposal).
			Updates(map[string]any{
				"IsEmergency":            true,
				"EmergencyNextTallyTime": nextTallyTime,
			})
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}

func UpsertModules(ctx context.Context, dbTx *gorm.DB, modules []Module) error {
	span := sentry.StartSpan(ctx, "UpsertModules")
	span.Description = "Bulk upsert modules into the database"
	defer span.Finish()

	if len(modules) == 0 {
		return nil
	}

	// List all columns to update on conflict, excluding primary key and ModuleEntryExecuted
	columns := []string{
		"upgrade_policy",
		"digest",
	}

	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns(columns),
			UpdateAll: false,
		}).
		CreateInBatches(&modules, BatchSize)

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

func UpsertTransactions(ctx context.Context, dbTx *gorm.DB, txs []Transaction) error {
	span := sentry.StartSpan(ctx, "UpsertTransactions")
	span.Description = "Bulk upsert transactions into the database"
	defer span.Finish()

	if len(txs) == 0 {
		return nil
	}

	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
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
			}),
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

func UpsertCollection(ctx context.Context, dbTx *gorm.DB, collections []Collection) error {
	if len(collections) == 0 {
		return nil
	}

	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name",
				"description",
				"uri",
			}),
		}).
		CreateInBatches(&collections, BatchSize)

	return result.Error
}

func InsertModuleTransactions(ctx context.Context, dbTx *gorm.DB, moduleTransactions []ModuleTransaction) error {
	if len(moduleTransactions) == 0 {
		return nil
	}

	return dbTx.WithContext(ctx).CreateInBatches(moduleTransactions, BatchSize).Error
}

func InsertModuleHistories(ctx context.Context, dbTx *gorm.DB, moduleHistories []ModuleHistory) error {
	if len(moduleHistories) == 0 {
		return nil
	}

	return dbTx.WithContext(ctx).CreateInBatches(moduleHistories, BatchSize).Error
}

func InsertOpinitTransactions(ctx context.Context, dbTx *gorm.DB, opinitTransactions []OpinitTransaction) error {
	if len(opinitTransactions) == 0 {
		return nil
	}

	return dbTx.WithContext(ctx).CreateInBatches(opinitTransactions, BatchSize).Error
}

func InsertCollectionTransactions(ctx context.Context, dbTx *gorm.DB, collectionTransactions []CollectionTransaction) error {
	if len(collectionTransactions) == 0 {
		return nil
	}

	return dbTx.WithContext(ctx).CreateInBatches(collectionTransactions, BatchSize).Error
}

func InsertNftsOnConflictDoUpdate(ctx context.Context, dbTx *gorm.DB, nftTransactions []*Nft) error {
	if len(nftTransactions) == 0 {
		return nil
	}

	return dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"owner",
				"is_burned",
				"description",
				"uri",
			}),
		}).CreateInBatches(nftTransactions, BatchSize).Error
}

func UpdateBurnedNftsOnConflictDoUpdate(ctx context.Context, dbTx *gorm.DB, nftIDs []string) error {
	if len(nftIDs) == 0 {
		return nil
	}

	return dbTx.WithContext(ctx).
		Model(&Nft{}).
		Where("id IN ?", nftIDs).
		Update("is_burned", true).Error
}

func InsertNftTransactions(ctx context.Context, dbTx *gorm.DB, nftTransactions []NftTransaction) error {
	if len(nftTransactions) == 0 {
		return nil
	}

	return dbTx.WithContext(ctx).CreateInBatches(nftTransactions, BatchSize).Error
}

func InsertNftHistories(ctx context.Context, dbTx *gorm.DB, nftHistories []NftHistory) error {
	if len(nftHistories) == 0 {
		return nil
	}

	return dbTx.WithContext(ctx).CreateInBatches(nftHistories, BatchSize).Error
}

func GetNftsByIDs(ctx context.Context, dbTx *gorm.DB, ids []string) ([]*Nft, error) {
	var nfts []*Nft
	result := dbTx.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&nfts)

	if result.Error != nil {
		return nil, result.Error
	}

	return nfts, nil
}

func InsertCollectionMutationEvents(ctx context.Context, dbTx *gorm.DB, collectionMutationEvents []CollectionMutationEvent) error {
	if len(collectionMutationEvents) == 0 {
		return nil
	}

	return dbTx.WithContext(ctx).CreateInBatches(collectionMutationEvents, BatchSize).Error
}

func UpdateCollectionURI(ctx context.Context, dbTx *gorm.DB, collectionID string, uri string) error {
	return dbTx.WithContext(ctx).
		Model(&Collection{}).
		Where("id = ?", collectionID).
		Update("uri", uri).Error
}

func UpdateCollectionDescription(ctx context.Context, dbTx *gorm.DB, collectionID string, description string) error {
	return dbTx.WithContext(ctx).
		Model(&Collection{}).
		Where("id = ?", collectionID).
		Update("description", description).Error
}

func UpdateCollectionName(ctx context.Context, dbTx *gorm.DB, collectionID string, name string) error {
	return dbTx.WithContext(ctx).
		Model(&Collection{}).
		Where("id = ?", collectionID).
		Update("name", name).Error
}

func UpdateNftURI(ctx context.Context, dbTx *gorm.DB, nftID string, uri string) error {
	return dbTx.WithContext(ctx).
		Model(&Nft{}).
		Where("id = ?", nftID).
		Update("uri", uri).Error
}

func UpdateNftDescription(ctx context.Context, dbTx *gorm.DB, nftID string, description string) error {
	return dbTx.WithContext(ctx).
		Model(&Nft{}).
		Where("id = ?", nftID).
		Update("description", description).Error
}

func InsertNftMutationEvents(ctx context.Context, dbTx *gorm.DB, nftMutationEvents []NftMutationEvent) error {
	if len(nftMutationEvents) == 0 {
		return nil
	}

	return dbTx.WithContext(ctx).CreateInBatches(nftMutationEvents, BatchSize).Error
}

func InsertProposalDeposits(ctx context.Context, dbTx *gorm.DB, proposalDeposits []ProposalDeposit) error {
	span := sentry.StartSpan(ctx, "InsertProposalDeposits")
	span.Description = "Bulk insert proposal_deposits into the database"
	defer span.Finish()

	return dbTx.WithContext(ctx).CreateInBatches(proposalDeposits, BatchSize).Error
}

func UpdateProposalTotalDeposit(ctx context.Context, dbTx *gorm.DB, totalDepositChanges map[int32][]sdk.Coin) error {
	for proposalID, depositChanges := range totalDepositChanges {
		var proposal Proposal
		result := dbTx.WithContext(ctx).
			Where("id = ?", proposalID).
			First(&proposal)

		if result.Error != nil {
			return result.Error
		}

		var totalDeposit sdk.Coins
		if err := json.Unmarshal(proposal.TotalDeposit, &totalDeposit); err != nil {
			return fmt.Errorf("failed to unmarshal total deposit of proposal %d - %w", proposalID, err)
		}

		totalDeposit = totalDeposit.Add(depositChanges...)
		totalDepositJSON, err := json.Marshal(totalDeposit)
		if err != nil {
			return fmt.Errorf("failed to marshal total deposit of proposal %d - %w", proposalID, err)
		}
		result = dbTx.WithContext(ctx).
			Model(&Proposal{}).
			Where("id = ?", proposalID).
			Updates(map[string]any{
				"total_deposit": totalDepositJSON,
			})
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func UpsertProposalVotes(ctx context.Context, dbTx *gorm.DB, proposalVotes []ProposalVote) error {
	if len(proposalVotes) == 0 {
		return nil
	}
	// TODO: add constraint on proposal_id and voter
	// Process each vote individually like the Python version
	for _, vote := range proposalVotes {
		// Check if vote already exists
		var existingVote ProposalVote
		result := dbTx.WithContext(ctx).
			Where("proposal_id = ? AND voter = ?", vote.ProposalID, vote.Voter).
			First(&existingVote)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// Vote doesn't exist, insert new one
				if err := dbTx.WithContext(ctx).Create(&vote).Error; err != nil {
					return fmt.Errorf("failed to insert proposal vote: %w", err)
				}
			} else {
				return fmt.Errorf("failed to check existing vote: %w", result.Error)
			}
		} else {
			// Vote exists, update it
			if err := dbTx.WithContext(ctx).
				Model(&ProposalVote{}).
				Where("proposal_id = ? AND voter = ?", vote.ProposalID, vote.Voter).
				Updates(map[string]interface{}{
					"is_vote_weighted":  vote.IsVoteWeighted,
					"is_validator":      vote.IsValidator,
					"validator_address": vote.ValidatorAddress,
					"yes":               vote.Yes,
					"no":                vote.No,
					"abstain":           vote.Abstain,
					"no_with_veto":      vote.NoWithVeto,
					"transaction_id":    vote.TransactionID,
				}).Error; err != nil {
				return fmt.Errorf("failed to update proposal vote: %w", err)
			}
		}
	}

	return nil
}

func GetLatestInformativeBlockHeight(ctx context.Context, dbClient *gorm.DB) (int64, error) {
	var tracking Tracking
	if err := dbClient.WithContext(ctx).First(&tracking).Error; err != nil {
		return 0, err
	}

	return tracking.LatestInformativeBlockHeight, nil
}

func IsTrackingInit(ctx context.Context, dbTx *gorm.DB) (bool, error) {
	var tracking Tracking
	if err := dbTx.WithContext(ctx).First(&tracking).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func InitTracking(ctx context.Context, dbTx *gorm.DB) error {
	tracking := Tracking{
		TxCount:                      0,
		LatestInformativeBlockHeight: 0,
	}
	return dbTx.WithContext(ctx).Create(&tracking).Error
}

func UpdateTxCount(ctx context.Context, dbTx *gorm.DB, txCount int64, height int64) error {
	var tracking Tracking
	if err := dbTx.WithContext(ctx).First(&tracking).Error; err != nil {
		return err
	}

	return dbTx.WithContext(ctx).
		Model(&tracking).
		Where("1 = 1").
		Update("tx_count", gorm.Expr("tx_count + ?", txCount)).
		Update("latest_informative_block_height", height).Error
}

func InsertHistoricalVotingPowers(ctx context.Context, dbTx *gorm.DB, historicalVotingPowers []ValidatorHistoricalPower) error {
	if len(historicalVotingPowers) == 0 {
		return nil
	}

	return dbTx.WithContext(ctx).CreateInBatches(historicalVotingPowers, BatchSize).Error
}

func QueryLatestInformativeBlockHeight(ctx context.Context, dbTx *gorm.DB) (int64, error) {
	var tracking Tracking
	if err := dbTx.WithContext(ctx).First(&tracking).Error; err != nil {
		return 0, err
	}

	return int64(tracking.LatestInformativeBlockHeight), nil
}

func QueryValidatorCommitSignatures(ctx context.Context, dbTx *gorm.DB, height, lookbackBlocks int64) ([]ValidatorCommitSignature, error) {
	var votes []ValidatorCommitSignature
	result := dbTx.WithContext(ctx).
		Model(&ValidatorCommitSignature{}).
		Select("validator_address, vote, block_height").
		Where("block_height < ? AND block_height >= ?", height, height-lookbackBlocks).
		Scan(&votes)

	return votes, result.Error
}

func TruncateTable(ctx context.Context, dbTx *gorm.DB, table string) error {
	return dbTx.WithContext(ctx).Exec(fmt.Sprintf("TRUNCATE TABLE %s", table)).Error
}

func InsertValidatorVoteCounts(ctx context.Context, dbTx *gorm.DB, validatorVoteCounts []ValidatorVoteCount) error {
	if len(validatorVoteCounts) == 0 {
		return nil
	}

	return dbTx.WithContext(ctx).CreateInBatches(validatorVoteCounts, BatchSize).Error
}

func DeleteValidatorCommitSignatures(ctx context.Context, dbTx *gorm.DB, height int64) error {
	return dbTx.WithContext(ctx).
		Model(&ValidatorCommitSignature{}).
		Where("block_height < ?", height).
		Delete(&ValidatorCommitSignature{}).Error
}

func QueryValidatorAddress(ctx context.Context, dbTx *gorm.DB, consensusAddress string) (*string, error) {
	var validator ValidatorAddress
	result := dbTx.WithContext(ctx).
		Table(TableNameValidator).
		Select("operator_address, account_id, consensus_address").
		Where("consensus_address = ?", consensusAddress).
		Limit(1).
		Scan(&validator)

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		return nil, nil
	}

	return &validator.OperatorAddress, nil
}

func InsertValidatorSlashEvents(ctx context.Context, dbTx *gorm.DB, validatorSlashEvents []ValidatorSlashEvent) error {
	if len(validatorSlashEvents) == 0 {
		return nil
	}

	return dbTx.WithContext(ctx).CreateInBatches(validatorSlashEvents, BatchSize).Error
}

func InsertModuleProposalsOnConflictDoUpdate(ctx context.Context, dbTx *gorm.DB, moduleProposals []ModuleProposal) error {
	if len(moduleProposals) == 0 {
		return nil
	}

	return dbTx.WithContext(ctx).CreateInBatches(moduleProposals, BatchSize).Error
}
