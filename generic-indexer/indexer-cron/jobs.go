package indexercron

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/h2non/bimg"
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
	// Use "" to fetch all validators (bonded, unbonding, unbonded) for historical power
	valInfos, err := rpcClient.Validators(valInfoSpanCtx, "", nil)
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

// calculateValidatorVoteCountsFromVotes aggregates votes per validator and returns ValidatorVoteCount with Last10000 set.
// Result is sorted by ValidatorAddress for determinism.
func calculateValidatorVoteCountsFromVotes(votes []db.ValidatorCommitSignature) []db.ValidatorVoteCount {
	m := make(map[string]int32)
	for _, vote := range votes {
		m[vote.ValidatorAddress]++
	}
	addrs := make([]string, 0, len(m))
	for addr := range m {
		addrs = append(addrs, addr)
	}
	sort.Strings(addrs)
	out := make([]db.ValidatorVoteCount, 0, len(addrs))
	for _, addr := range addrs {
		out = append(out, db.ValidatorVoteCount{ValidatorAddress: addr, Last10000: m[addr]})
	}
	return out
}

// updateValidatorUptimeLast10000 updates validator_vote_counts.last_10000 (signed blocks in last 10,000). API uses for SignedBlocks and uptime.
func updateValidatorUptimeLast10000(parentCtx context.Context, dbClient *gorm.DB, config *IndexerCronConfig) error {
	transaction, ctx := sentry_integration.StartSentryTransaction(parentCtx, "updateValidatorUptimeLast10000", "Update validator vote counts for last 10,000 blocks")
	defer transaction.Finish()
	logger := zerolog.Ctx(log.With().
		Str("component", "indexer-cron").
		Str("function_name", "updateValidatorUptimeLast10000").
		Str("chain", config.Chain).
		Str("environment", config.Environment).
		Logger().
		WithContext(ctx))

	logger.Info().Msg("Starting updateValidatorUptimeLast10000 task ...")

	if err := dbClient.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		height, err := db.QueryLatestInformativeBlockHeight(ctx, dbTx)
		if err != nil {
			logger.Error().Msgf("Error querying latest block height: %v", err)
			return err
		}
		votes, err := db.QueryValidatorCommitSignatures(ctx, dbTx, height, 10000)
		if err != nil {
			logger.Error().Msgf("Error fetching validator commit signatures for last 10,000 blocks: %v", err)
			return err
		}
		counts := calculateValidatorVoteCountsFromVotes(votes)
		if err := db.UpsertValidatorVoteCountLast10000(ctx, dbTx, counts); err != nil {
			logger.Error().Msgf("Error upserting validator vote counts: %v", err)
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	logger.Info().Msg("Successfully updated validator vote counts for latest 10,000 blocks")
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
	// Use "" to fetch all validators (bonded, unbonding, unbonded) so DB stays in sync
	valInfos, err := rpcClient.Validators(valInfoSpanCtx, "", nil)
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

const avatarSize = 36

// normalizeImageToJPEG uses bimg (libvips) to convert the image to JPEG and resize to avatarSize×avatarSize.
// Returns nil on failure so caller can keep the original bytes.
// Requires libvips to be installed on the system (e.g. apt install libvips-dev, brew install vips).
func normalizeImageToJPEG(data []byte) []byte {
	opts := bimg.Options{
		Width:   avatarSize,
		Height:  avatarSize,
		Type:    bimg.JPEG,
		Quality: 80,
		Crop:    true, // crop to exact 36×36 (center crop)
	}
	out, err := bimg.NewImage(data).Process(opts)
	if err != nil {
		return nil
	}
	return out
}

// keybaseImageCache caches Keybase identity -> base64 image in memory to avoid repeated API calls.
var (
	keybaseImageCache   = make(map[string]string)
	keybaseImageCacheMu sync.RWMutex
)

// fetchImageDataFromKeybase fetches the validator image from Keybase API and returns it as base64.
// Results are cached in memory by identity so each identity is only fetched once per process.
// Second return is true if the result was from cache (no Keybase call).
func fetchImageDataFromKeybase(identity string) (string, bool) {
	if identity == "" {
		return "", false
	}
	keybaseImageCacheMu.RLock()
	cached, ok := keybaseImageCache[identity]
	keybaseImageCacheMu.RUnlock()
	if ok {
		return cached, true
	}

	// First, get the image URL from Keybase API
	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("https://keybase.io/_/api/1.0/user/lookup.json?key_suffix=%s&fields=pictures", identity)

	resp, err := client.Get(url)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false
	}

	var keybaseResp keybaseResponse
	if err := json.Unmarshal(body, &keybaseResp); err != nil {
		return "", false
	}

	imageURL := ""
	if len(keybaseResp.Them) > 0 && keybaseResp.Them[0].Pictures.Primary.URL != "" {
		imageURL = keybaseResp.Them[0].Pictures.Primary.URL
	}

	if imageURL == "" {
		return "", false
	}

	// Now fetch the actual image data from the URL
	imageResp, err := client.Get(imageURL)
	if err != nil {
		return "", false
	}
	defer imageResp.Body.Close()

	if imageResp.StatusCode != http.StatusOK {
		return "", false
	}

	imageData, err := io.ReadAll(imageResp.Body)
	if err != nil {
		return "", false
	}

	// Normalize to JPEG to reduce stored size (avatars from Keybase may be PNG or large JPEGs)
	if normalized := normalizeImageToJPEG(imageData); normalized != nil {
		imageData = normalized
	}

	// Convert to base64 and cache
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	keybaseImageCacheMu.Lock()
	keybaseImageCache[identity] = base64Image
	keybaseImageCacheMu.Unlock()
	return base64Image, false
}

// updateValidatorImages fetches and updates image data (base64-encoded) only for validators that have
// identity but no cached image in DB. Keybase results are cached in memory by identity (one fetch per identity per process).
func updateValidatorImages(ctx context.Context, dbClient *gorm.DB, logger *zerolog.Logger) error {
	// Only validators with identity and no image yet — DB is source of truth; memory cache avoids duplicate Keybase calls
	var validators []db.Validator
	if err := dbClient.WithContext(ctx).
		Model(&db.Validator{}).
		Where("identity != '' AND identity_image = ''").
		Find(&validators).Error; err != nil {
		logger.Error().Err(err).Msg("Failed to query validators for image update")
		return err
	}

	if len(validators) == 0 {
		logger.Debug().Msg("No validators missing cached images, skipping")
		return nil
	}

	logger.Info().Msgf("Updating images for %d validators (in-memory cache used per identity)", len(validators))

	var toUpsert []db.Validator
	for _, val := range validators {
		imageData, _ := fetchImageDataFromKeybase(val.Identity)
		if imageData == "" {
			continue
		}
		toUpsert = append(toUpsert, db.Validator{OperatorAddress: val.OperatorAddress, IdentityImage: imageData})
	}

	if len(toUpsert) > 0 {
		if err := db.UpsertValidatorIdentityImages(ctx, dbClient, toUpsert); err != nil {
			logger.Warn().Err(err).Msg("Failed to upsert validator identity images")
		}
	}

	logger.Info().Msg("Validator image update completed")
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
