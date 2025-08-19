package indexercron

import (
	"context"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	vmtypes "github.com/initia-labs/movevm/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/cosmosrpc"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
)

// updateValidatorHistoricalPower updates the historical power of validators in the database.
func updateValidatorHistoricalPower(parentCtx context.Context, dbClient *gorm.DB, rpcClient cosmosrpc.CosmosJSONRPCHub, config *IndexerCronConfig) error {
	transaction, ctx := sentry_integration.StartSentryTransaction(parentCtx, "updateValidatorHistoricalPower", "Update all validator historical power in the database")
	defer transaction.Finish()
	// Create a logger with contextual information
	logger := zerolog.Ctx(log.With().
		Str("component", "indexer-cron").
		Str("function_name", "updateValidatorHistoricalPower").
		Str("chain", config.Chain).
		Str("environment", config.Environment).
		Logger().
		WithContext(ctx))

	logger.Info().Msgf("Starting updateValidatorHistoricalPower task ...")

	err := rpcClient.Rebalance(ctx)
	if err != nil {
		logger.Error().Msgf("Error rebalancing clients: %v", err)
		return err
	}
	timestamp := time.Now()

	span, valInfoSpanCtx := sentry_integration.StartSentrySpan(ctx, "getValidatorInfos", "Get all validator infos from RPC endpoints")
	valInfos, err := rpcClient.Validators(valInfoSpanCtx, "BOND_STATUS_BONDED", nil)
	if err != nil {
		logger.Error().Msgf("Error cannot get Validator info from all RPC endpoints: %v", err)
		return err
	}
	span.Finish()

	hpInfos := make([]db.ValidatorHistoricalPower, 0)
	for _, valInfo := range *valInfos {
		hpInfo, err := db.NewValidatorHistoricalPower(valInfo, timestamp)
		if err != nil {
			logger.Error().Msgf("Error new validator historical info: %v", err)
			return err
		}
		hpInfos = append(hpInfos, hpInfo)
	}

	if err := dbClient.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// Insert historical voting powers into the database
		err = db.InsertHistoricalVotingPowers(ctx, dbTx, hpInfos)
		if err != nil {
			logger.Error().Msgf("Error inserting historical voting powers into database: %v", err)
			return err
		}
		return nil
	}); err != nil {
		logger.Error().Msgf("Error inserting historical voting powers into database: %v", err)
		return err
	}

	// Log a success message
	logger.Info().Msgf("Successfully updated validator historical power")

	return nil
}

// calculateValidatorUptimes calculates the uptime for each validator.
func calculateValidatorUptimes(votes []db.ValidatorCommitSignature) []db.ValidatorVoteCount {
	mapValidatorAddressToVoteCount := make(map[string]int32)
	for _, vote := range votes {
		mapValidatorAddressToVoteCount[vote.ValidatorAddress]++
	}
	validatorVoteCounts := make([]db.ValidatorVoteCount, 0)
	for validatorAddress, voteCount := range mapValidatorAddressToVoteCount {
		validatorVoteCounts = append(validatorVoteCounts, db.ValidatorVoteCount{
			ValidatorAddress: validatorAddress,
			Last100:          voteCount,
		})
	}
	return validatorVoteCounts
}

// updateLatest100BlockValidatorUptime updates the latest 100 blocks validator uptime in the database.
func updateLatest100BlockValidatorUptime(parentCtx context.Context, dbClient *gorm.DB, config *IndexerCronConfig) error {
	transaction, ctx := sentry_integration.StartSentryTransaction(parentCtx, "updateLatest100BlockValidatorUptime", "Update latest 100 validator uptime in the database")
	defer transaction.Finish()
	// Create a logger with contextual information
	logger := zerolog.Ctx(log.With().
		Str("component", "indexer-cron").
		Str("function_name", "updateLatest100BlockValidatorUptime").
		Str("chain", config.Chain).
		Str("environment", config.Environment).
		Logger().
		WithContext(ctx))

	logger.Info().Msgf("Starting updateLatest100BlockValidatorUptime task ...")

	if err := dbClient.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		height, err := db.QueryLatestInformativeBlockHeight(ctx, dbTx)
		if err != nil {
			logger.Error().Msgf("Error querying latest validator vote signature: %v", err)
			return err
		}

		lookbackBlocks := int64(100)
		// Fetch validator commit signatures from the database.
		votes, err := db.QueryValidatorCommitSignatures(ctx, dbTx, height, lookbackBlocks)
		if err != nil {
			logger.Error().Msgf("Error fetching validator commit signatures: %v", err)
			return err
		}

		// Calculate the validator uptimes based on the votes and proposer count.
		validatorUptimes := calculateValidatorUptimes(votes)

		// Truncate the validator_vote_counts table before inserting updated data.
		if err := db.TruncateTable(ctx, dbTx, db.TableNameValidatorVoteCount); err != nil {
			logger.Error().Msgf("Error truncating the validator_vote_counts table before inserting updated data: %v", err)
			return err
		}

		// Insert the calculated validator uptimes into the database.
		if err := db.InsertValidatorVoteCounts(ctx, dbTx, validatorUptimes); err != nil {
			logger.Error().Msgf("Error inserting the calculated validator uptimes into the database: %v", err)
			return err
		}
		return nil
	}); err != nil {
		logger.Error().Msgf("Error inserting the calculated validator uptimes into the database: %v", err)
		return err
	}

	// Log success message
	logger.Info().Msg("Successfully updated latest 100 block validator uptime")

	return nil
}

func updateValidators(parentCtx context.Context, dbClient *gorm.DB, rpcClient cosmosrpc.CosmosJSONRPCHub, interfaceRegistry codectypes.InterfaceRegistry, config *IndexerCronConfig) error {
	transaction, ctx := sentry_integration.StartSentryTransaction(parentCtx, "updateValidators", "Update all validator details in the database")
	defer transaction.Finish()
	// Create a logger with contextual information
	logger := zerolog.Ctx(log.With().
		Str("component", "indexer-cron").
		Str("function_name", "updateValidators").
		Str("chain", config.Chain).
		Str("environment", config.Environment).
		Logger().
		WithContext(ctx))

	logger.Info().Msgf("Starting updateValidators task ...")

	err := rpcClient.Rebalance(ctx)
	if err != nil {
		logger.Error().Msgf("Error rebalancing clients: %v", err)
		return err
	}

	span, valInfoSpanCtx := sentry_integration.StartSentrySpan(ctx, "getValidatorInfos", "Get all validator infos from RPC endpoints")
	valInfos, err := rpcClient.Validators(valInfoSpanCtx, "BOND_STATUS_BONDED", nil)
	if err != nil {
		logger.Error().Msgf("Error cannot get Validator info from all RPC endpoints: %v", err)
		return err
	}
	span.Finish()

	addresses := make([]string, 0)
	vals := make(map[string]mstakingtypes.Validator)
	for _, valInfo := range *valInfos {
		bz, err := sdk.GetFromBech32(valInfo.OperatorAddress, "initvaloper")
		if err != nil {
			logger.Error().Msgf("Error getting bech32 from validator address: %v", err)
			return err
		}
		acc := sdk.ValAddress(bz)
		address := sdk.AccAddress(acc).String()
		addresses = append(addresses, address)
		vals[address] = valInfo
	}

	if err := dbClient.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		accounts := make([]db.Account, 0, len(addresses))
		vmAddresses := make([]db.VMAddress, 0, len(addresses))
		for _, address := range addresses {
			accAddr, err := sdk.AccAddressFromBech32(address)
			if err != nil {
				logger.Error().Msgf("Error getting account address from validator address: %v", err)
				return err
			}
			vmAddr, _ := vmtypes.NewAccountAddressFromBytes(accAddr)
			vmAddresses = append(vmAddresses, db.VMAddress{VMAddress: vmAddr.String()})
			accounts = append(accounts, db.Account{
				Address:     address,
				VMAddressID: vmAddr.String(),
				Type:        string(db.BaseAccount),
			})
		}

		err = db.InsertVMAddressesIgnoreConflict(ctx, dbTx, vmAddresses)
		if err != nil {
			logger.Error().Msgf("Error: Failed to insert vm addresses %v into the database: %v", addresses, err)
			return err
		}
		err = db.InsertAccountsIgnoreConflict(ctx, dbTx, accounts)
		if err != nil {
			logger.Error().Msgf("Error: Failed to insert accounts %v into the database: %v", addresses, err)
			return err
		}

		dbVals := make([]db.Validator, 0)
		for _, acc := range addresses {
			valInfo := vals[acc]
			valInfo.UnpackInterfaces(interfaceRegistry)
			consAddr, err := valInfo.GetConsAddr()
			if err != nil {
				logger.Error().Msgf("Error getting consensus address for validator: %v", err)
				return err
			}
			dbVals = append(dbVals, db.NewValidator(valInfo, acc, consAddr))
		}

		err = db.UpsertValidators(
			ctx,
			dbTx,
			dbVals,
		)
		if err != nil {
			logger.Error().Msgf("Error upserting validators into the database: %v", err)
			return err
		}
		return nil
	}); err != nil {
		logger.Error().Msgf("Error upserting validators into the database: %v", err)
		return err
	}

	// Log success message
	logger.Info().Msgf("Successfully updated validators")

	return nil
}

func pruneCommitSignatures(parenCtx context.Context, dbClient *gorm.DB, config *IndexerCronConfig) error {
	transaction, ctx := sentry_integration.StartSentryTransaction(parenCtx, "pruneCommitSignatures", "Prune commit signatures in the database")
	defer transaction.Finish()
	// Create a logger with contextual information
	logger := zerolog.Ctx(log.With().
		Str("component", "indexer-cron").
		Str("function_name", "pruneCommitSignatures").
		Str("chain", config.Chain).
		Str("environment", config.Environment).
		Logger().
		WithContext(ctx))

	logger.Info().Msgf("Starting pruneCommitSignatures task ...")

	err := dbClient.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		latest, err := db.QueryLatestInformativeBlockHeight(
			ctx,
			dbTx,
		)
		if err != nil {
			logger.Error().Msgf("Error querying latest validator vote signature: %v", err)
			return err
		}

		err = db.DeleteValidatorCommitSignatures(ctx, dbTx, latest-config.KeepLatestCommitSignatures)
		if err != nil {
			logger.Error().Msgf("Error pruning validator vote signatures: %v", err)
			return err
		}

		// Log success message
		logger.Info().Msgf("Successfully pruned validators")

		return nil
	})
	if err != nil {
		logger.Error().Msgf("Error pruning validator vote signatures: %v", err)
		return err
	}

	return nil
}
