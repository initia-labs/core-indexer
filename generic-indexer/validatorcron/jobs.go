package validatorcron

import (
	"context"
	"sort"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/mstaking/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/initia-labs/core-indexer/generic-indexer/db"
	"github.com/initia-labs/core-indexer/pkg/cosmosrpc"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
)

// updateValidatorHistoricalPower updates the historical power of validators in the database.
func updateValidatorHistoricalPower(parentCtx context.Context, dbClient *pgxpool.Pool, rpcClient cosmosrpc.CosmosJSONRPCHub, config *ValidatorCronConfig) error {
	transaction, ctx := sentry_integration.StartSentryTransaction(parentCtx, "updateValidatorHistoricalPower", "Update all validator historical power in the database")
	defer transaction.Finish()
	// Create a logger with contextual information
	logger := zerolog.Ctx(log.With().
		Str("component", "validator-cron").
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

	// Begin a database transaction
	dbTx, err := dbClient.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		logger.Error().Msgf("Error beginning transaction: %v", err)
		return err
	}

	// Rollback the transaction in case of error
	defer dbTx.Rollback(ctx)

	span, valInfoSpanCtx := sentry_integration.StartSentrySpan(ctx, "getValidatorInfos", "Get all validator infos from RPC endpoints")
	valInfos, err := rpcClient.ValidatorInfos(valInfoSpanCtx, "BOND_STATUS_BONDED", nil)
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

	// Insert historical voting powers into the database
	err = db.InsertHistoricalVotingPowers(ctx, dbTx, hpInfos)
	if err != nil {
		logger.Error().Msgf("Error inserting historical voting powers into database: %v", err)
		return err
	}

	err = dbTx.Commit(ctx)
	if err != nil {
		logger.Error().Msgf("Error committing transaction: %v", err)
		return err
	}

	// Log a success message
	logger.Info().Msgf("Successfully updated validator historical power")

	return nil
}

// calculateValidatorUptimes calculates the uptime for each validator.
func calculateValidatorUptimes(votes []db.ValidatorVote, maxHeight int64, lookbackBlocks int64) []db.ValidatorUptime {
	// Sort the votes by height in ascending order.
	sort.Slice(votes, func(i, j int) bool {
		return votes[i].Height < votes[j].Height
	})

	voteCount := make(map[string]int) // Map to count votes for each validator.
	front := 0                        // Index to track the current position in the votes slice.
	optimistic := 0                   // Counter for optimistic blocks (blocks assumed to be committed).

	// Iterate over the range of heights within the lookback period.
	for height := maxHeight - lookbackBlocks; height < maxHeight; height++ {
		voteForHeight := false
		// Process all votes for the current height.
		for ; front < len(votes) && votes[front].Height == height; front++ {
			voteForHeight = true
			voteCount[votes[front].ValidatorAddress]++
		}
		// If no votes were found for the current height, increment the optimistic counter.
		if !voteForHeight {
			optimistic++
		}
	}

	// Prepare the result slice for validator uptimes.
	validatorUptimes := make([]db.ValidatorUptime, 0)

	// Calculate the total vote count for each validator, including optimistic blocks.
	for validatorAddress, count := range voteCount {
		totalVotes := count + optimistic
		validatorUptimes = append(validatorUptimes, db.ValidatorUptime{
			Validator: validatorAddress,
			VoteCount: totalVotes,
		})
	}

	return validatorUptimes
}

// updateLatest100BlockValidatorUptime updates the latest 100 blocks validator uptime in the database.
func updateLatest100BlockValidatorUptime(parentCtx context.Context, dbClient *pgxpool.Pool, config *ValidatorCronConfig) error {
	transaction, ctx := sentry_integration.StartSentryTransaction(parentCtx, "updateLatest100BlockValidatorUptime", "Update latest 100 validator uptime in the database")
	defer transaction.Finish()
	// Create a logger with contextual information
	logger := zerolog.Ctx(log.With().
		Str("component", "validator-cron").
		Str("function_name", "updateLatest100BlockValidatorUptime").
		Str("chain", config.Chain).
		Str("environment", config.Environment).
		Logger().
		WithContext(ctx))

	logger.Info().Msgf("Starting updateLatest100BlockValidatorUptime task ...")

	dbTx, err := dbClient.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		logger.Error().Msgf("Error beginning transaction: %v", err)
		return err
	}
	// Rollback the transaction in case of error
	defer dbTx.Rollback(ctx)

	height, err := db.QueryLatestValidatorVoteSignature(ctx, dbClient)
	if err != nil {
		logger.Error().Msgf("Error querying latest validator vote signature: %v", err)
		return err
	}

	lookbackBlocks := int64(100)
	// Fetch validator commit signatures from the database.
	votes, err := db.FetchValidatorCommitSignatures(ctx, dbTx, height, lookbackBlocks)
	if err != nil {
		logger.Error().Msgf("Error fetching validator commit signatures: %v", err)
		return err
	}

	// Calculate the validator uptimes based on the votes and proposer count.
	validatorUptimes := calculateValidatorUptimes(votes, height, lookbackBlocks)

	// Truncate the validator_vote_counts table before inserting updated data.
	if err := db.TruncateTable(ctx, dbTx, "validator_vote_counts"); err != nil {
		logger.Error().Msgf("Error truncating the validator_vote_counts table before inserting updated data: %v", err)
		return err
	}

	// Insert the calculated validator uptimes into the database.
	if err := db.InsertValidatorUptimes(ctx, dbTx, validatorUptimes); err != nil {
		logger.Error().Msgf("Error inserting the calculated validator uptimes into the database: %v", err)
		return err
	}

	err = dbTx.Commit(ctx)
	if err != nil {
		logger.Error().Msgf("Error committing transaction: %v", err)
		return err
	}

	// Log success message
	logger.Info().Msg("Successfully updated latest 100 block validator uptime")

	return nil
}

func updateValidators(parentCtx context.Context, dbClient *pgxpool.Pool, rpcClient cosmosrpc.CosmosJSONRPCHub, interfaceRegistry codectypes.InterfaceRegistry, config *ValidatorCronConfig) error {
	transaction, ctx := sentry_integration.StartSentryTransaction(parentCtx, "updateValidators", "Update all validator details in the database")
	defer transaction.Finish()
	// Create a logger with contextual information
	logger := zerolog.Ctx(log.With().
		Str("component", "validator-cron").
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
	valInfos, err := rpcClient.ValidatorInfos(valInfoSpanCtx, "BOND_STATUS_BONDED", nil)
	if err != nil {
		logger.Error().Msgf("Error cannot get Validator info from all RPC endpoints: %v", err)
		return err
	}
	span.Finish()

	addresses := make([]string, 0)
	vals := make(map[string]types.Validator)
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

	dbTx, err := dbClient.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		logger.Error().Msgf("Error beginning transaction: %v", err)
		return err
	}
	defer dbTx.Rollback(ctx)

	notExistAccounts, err := db.GetAccountsIfNotExist(ctx, dbTx, addresses)
	if err != nil {
		logger.Error().Msgf("Error: Failed get accounts %v into the database: %v", addresses, err)
		return err
	}

	err = db.InsertAccounts(ctx, dbTx, notExistAccounts)
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
		&dbVals,
	)
	if err != nil {
		logger.Error().Msgf("Error upserting validators into the database: %v", err)
		return err
	}

	err = dbTx.Commit(ctx)
	if err != nil {
		logger.Error().Msgf("Error committing transaction: %v", err)
		return err
	}

	// Log success message
	logger.Info().Msgf("Successfully updated validators")

	return nil
}

func pruneCommitSignatures(parenCtx context.Context, dbClient *pgxpool.Pool, config *ValidatorCronConfig) error {
	transaction, ctx := sentry_integration.StartSentryTransaction(parenCtx, "pruneCommitSignatures", "Prune commit signatures in the database")
	defer transaction.Finish()
	// Create a logger with contextual information
	logger := zerolog.Ctx(log.With().
		Str("component", "validator-cron").
		Str("function_name", "pruneCommitSignatures").
		Str("chain", config.Chain).
		Str("environment", config.Environment).
		Logger().
		WithContext(ctx))

	logger.Info().Msgf("Starting pruneCommitSignatures task ...")

	dbTx, err := dbClient.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		logger.Error().Msgf("Error beginning transaction: %v", err)
		return err
	}

	defer dbTx.Rollback(ctx)

	latest, err := db.QueryLatestValidatorVoteSignature(
		ctx,
		dbClient,
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

	err = dbTx.Commit(ctx)
	if err != nil {
		logger.Error().Msgf("Error committing transaction: %v", err)
		return err
	}

	// Log success message
	logger.Info().Msgf("Successfully pruned validators")

	return nil
}
