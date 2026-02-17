package indexercron

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
func calculateValidatorUptimes(votes100 []db.ValidatorCommitSignature, votes10000 []db.ValidatorCommitSignature) []db.ValidatorVoteCount {
	mapValidatorAddressToVoteCount100 := make(map[string]int32)
	for _, vote := range votes100 {
		mapValidatorAddressToVoteCount100[vote.ValidatorAddress]++
	}

	mapValidatorAddressToVoteCount10000 := make(map[string]int32)
	for _, vote := range votes10000 {
		mapValidatorAddressToVoteCount10000[vote.ValidatorAddress]++
	}

	// Merge all validator addresses from both maps
	allValidators := make(map[string]bool)
	for addr := range mapValidatorAddressToVoteCount100 {
		allValidators[addr] = true
	}
	for addr := range mapValidatorAddressToVoteCount10000 {
		allValidators[addr] = true
	}

	validatorVoteCounts := make([]db.ValidatorVoteCount, 0)
	for validatorAddress := range allValidators {
		voteCount100 := mapValidatorAddressToVoteCount100[validatorAddress]
		voteCount10000 := mapValidatorAddressToVoteCount10000[validatorAddress]
		validatorVoteCounts = append(validatorVoteCounts, db.ValidatorVoteCount{
			ValidatorAddress: validatorAddress,
			Last100:          voteCount100,
			Last10000:        voteCount10000,
		})
	}
	return validatorVoteCounts
}

// updateLatest100BlockValidatorUptime updates the latest 100 and 10,000 blocks validator uptime in the database.
func updateLatest100BlockValidatorUptime(parentCtx context.Context, dbClient *gorm.DB, config *IndexerCronConfig) error {
	transaction, ctx := sentry_integration.StartSentryTransaction(parentCtx, "updateLatest100BlockValidatorUptime", "Update latest 100 and 10,000 validator uptime in the database")
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

		// Fetch validator commit signatures for last 100 blocks
		lookbackBlocks100 := int64(100)
		votes100, err := db.QueryValidatorCommitSignatures(ctx, dbTx, height, lookbackBlocks100)
		if err != nil {
			logger.Error().Msgf("Error fetching validator commit signatures for last 100 blocks: %v", err)
			return err
		}

		// Fetch validator commit signatures for last 10,000 blocks
		lookbackBlocks10000 := int64(10000)
		votes10000, err := db.QueryValidatorCommitSignatures(ctx, dbTx, height, lookbackBlocks10000)
		if err != nil {
			logger.Error().Msgf("Error fetching validator commit signatures for last 10,000 blocks: %v", err)
			return err
		}

		// Calculate the validator uptimes based on the votes for both 100 and 10,000 blocks
		validatorUptimes := calculateValidatorUptimes(votes100, votes10000)

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
	logger.Info().Msg("Successfully updated latest 100 and 10,000 block validator uptime")

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

	// Update validator images from Keybase
	if err := updateValidatorImages(ctx, dbClient, logger); err != nil {
		// Log the error but don't fail the entire job if image fetching fails
		logger.Warn().Err(err).Msg("Failed to update validator images")
	}

	return nil
}

// keybaseResponse represents the structure of Keybase API response
type keybaseResponse struct {
	Them []struct {
		Pictures struct {
			Primary struct {
				URL string `json:"url"`
			} `json:"primary"`
		} `json:"pictures"`
	} `json:"them"`
}

// fetchImageDataFromKeybase fetches the validator image from Keybase API and returns it as base64
func fetchImageDataFromKeybase(identity string) string {
	if identity == "" {
		return ""
	}

	// First, get the image URL from Keybase API
	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("https://keybase.io/_/api/1.0/user/lookup.json?key_suffix=%s&fields=pictures", identity)

	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var keybaseResp keybaseResponse
	if err := json.Unmarshal(body, &keybaseResp); err != nil {
		return ""
	}

	imageURL := ""
	if len(keybaseResp.Them) > 0 && keybaseResp.Them[0].Pictures.Primary.URL != "" {
		imageURL = keybaseResp.Them[0].Pictures.Primary.URL
	}

	if imageURL == "" {
		return ""
	}

	// Now fetch the actual image data from the URL
	imageResp, err := client.Get(imageURL)
	if err != nil {
		return ""
	}
	defer imageResp.Body.Close()

	if imageResp.StatusCode != http.StatusOK {
		return ""
	}

	imageData, err := io.ReadAll(imageResp.Body)
	if err != nil {
		return ""
	}

	// Convert to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	return base64Image
}

// updateValidatorImages fetches and updates image data (base64-encoded) for all validators with identity
func updateValidatorImages(ctx context.Context, dbClient *gorm.DB, logger *zerolog.Logger) error {
	// Fetch all validators with non-empty identity
	var validators []db.Validator
	if err := dbClient.WithContext(ctx).
		Model(&db.Validator{}).
		Where("identity != ''").
		Find(&validators).Error; err != nil {
		logger.Error().Err(err).Msg("Failed to query validators for image update")
		return err
	}

	logger.Info().Msgf("Updating images for %d validators with identity", len(validators))

	// Update images in batches to avoid holding locks too long
	for _, val := range validators {
		imageData := fetchImageDataFromKeybase(val.Identity)

		// Only update if we successfully fetched an image or need to clear an old one
		if imageData != val.IdentityImage {
			if err := dbClient.WithContext(ctx).
				Model(&db.Validator{}).
				Where("operator_address = ?", val.OperatorAddress).
				Update("identity_image", imageData).Error; err != nil {
				logger.Warn().
					Err(err).
					Str("operator_address", val.OperatorAddress).
					Str("identity", val.Identity).
					Msg("Failed to update validator image")
				// Continue with other validators even if one fails
				continue
			}
		}

		// Small delay to avoid rate limiting from Keybase API
		time.Sleep(100 * time.Millisecond)
	}

	logger.Info().Msg("Successfully updated validator images")
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
