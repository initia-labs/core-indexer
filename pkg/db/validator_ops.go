package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/getsentry/sentry-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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

func UpsertValidatorIdentityImages(ctx context.Context, dbTx *gorm.DB, validators []Validator) error {
	if len(validators) == 0 {
		return nil
	}
	for i := 0; i < len(validators); i += BatchSize {
		end := i + BatchSize
		if end > len(validators) {
			end = len(validators)
		}
		batch := validators[i:end]
		var values []any
		var placeholders []string
		for idx, val := range batch {
			paramNum := idx*2 + 1
			placeholders = append(placeholders, fmt.Sprintf("($%d, $%d)", paramNum, paramNum+1))
			values = append(values, val.OperatorAddress, val.IdentityImage)
		}
		query := fmt.Sprintf(`
			UPDATE validators 
			SET identity_image = v.identity_image
			FROM (VALUES %s) AS v(operator_address, identity_image)
			WHERE validators.operator_address = v.operator_address
		`, strings.Join(placeholders, ", "))
		if err := dbTx.WithContext(ctx).Exec(query, values...).Error; err != nil {
			return err
		}
	}
	return nil
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

func InsertHistoricalVotingPowers(ctx context.Context, dbTx *gorm.DB, historicalVotingPowers []ValidatorHistoricalPower) error {
	if len(historicalVotingPowers) == 0 {
		return nil
	}
	return dbTx.WithContext(ctx).CreateInBatches(historicalVotingPowers, BatchSize).Error
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

func InsertValidatorVoteCounts(ctx context.Context, dbTx *gorm.DB, validatorVoteCounts []ValidatorVoteCount) error {
	if len(validatorVoteCounts) == 0 {
		return nil
	}
	return dbTx.WithContext(ctx).CreateInBatches(validatorVoteCounts, BatchSize).Error
}

func UpsertValidatorVoteCountLast10000(ctx context.Context, dbTx *gorm.DB, validatorVoteCounts []ValidatorVoteCount) error {
	if len(validatorVoteCounts) == 0 {
		return nil
	}
	return dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "validator_address"}},
			DoUpdates: clause.AssignmentColumns([]string{"last_10000"}),
		}).
		CreateInBatches(validatorVoteCounts, BatchSize).Error
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
