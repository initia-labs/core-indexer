package validatorcron

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/certifi/gocertifi"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/getsentry/sentry-go"
	initiaapp "github.com/initia-labs/initia/app"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/initia-labs/core-indexer/generic-indexer/common"
	"github.com/initia-labs/core-indexer/generic-indexer/db"
	"github.com/initia-labs/core-indexer/pkg/cosmosrpc"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
)

type ValidatorCron struct {
	dbClient           *pgxpool.Pool
	rpcClient          cosmosrpc.CosmosJSONRPCHub
	interfaceRegistry  codectypes.InterfaceRegistry
	DBConnectionString string
	config             *ValidatorCronConfig
}

type ValidatorCronConfig struct {
	RPCEndpoints                           string
	Chain                                  string
	DBConnectionString                     string
	ValidatorUpdateIntervalInSeconds       int64
	ValidatorUptimeUpdateIntervalInSeconds int64
	Environment                            string
	KeepLatestCommitSignatures             int64
	RPCTimeOutInSeconds                    int64
	SentryDSN                              string
	CommitSHA                              string
	SentryProfilesSampleRate               float64
	SentryTracesSampleRate                 float64
}

func NewValidatorCronFlusher(config *ValidatorCronConfig) (*ValidatorCron, error) {
	logger := zerolog.Ctx(log.With().
		Str("component", "validator-cron").Str("environment", config.Environment).
		Str("commit_sha", config.CommitSHA).
		Logger().WithContext(context.Background()))

	sentryClientOptions := sentry.ClientOptions{
		Dsn:                config.SentryDSN,
		ServerName:         config.Chain + "-validator-cron",
		EnableTracing:      true,
		ProfilesSampleRate: config.SentryProfilesSampleRate,
		TracesSampleRate:   config.SentryTracesSampleRate,
		Environment:        config.Environment,
		Release:            config.CommitSHA,
		Tags: map[string]string{
			"chain":       config.Chain,
			"environment": config.Environment,
			"component":   "validator-cron",
			"commit_sha":  config.CommitSHA,
		},
	}

	rootCAs, err := gocertifi.CACerts()
	if err != nil {
		logger.Fatal().Msgf("Sentry: Error getting root CAs: %v\n", err)
	} else {
		sentryClientOptions.CaCerts = rootCAs
	}

	err = sentry.Init(sentryClientOptions)
	if err != nil {
		logger.Fatal().Msgf("Sentry: Error initializing sentry: %v\n", err)
		return nil, err
	}

	if config.RPCEndpoints == "" {

		logger.Fatal().Msgf("RPC: No RPC endpoints provided\n")
		return nil, fmt.Errorf("RPC: No RPC endpoints provided")
	}

	var rpcEndpoints common.RPCEndpoints
	err = json.Unmarshal([]byte(config.RPCEndpoints), &rpcEndpoints)
	if err != nil {
		sentry_integration.CaptureCurrentHubException(errors.New("RPC: No RPC endpoints provided"), sentry.LevelFatal)
		logger.Fatal().Msgf("RPC: Error unmarshalling RPC endpoints: %v\n", err)
		return nil, err
	}

	clientConfigs := make([]cosmosrpc.ClientConfig, 0)
	for _, rpc := range rpcEndpoints.RPCs {
		clientConfigs = append(clientConfigs, cosmosrpc.ClientConfig{
			URL:          rpc.URL,
			ClientOption: &cosmosrpc.ClientOption{CustomHeaders: rpc.Headers},
		})
	}
	rpcClient := cosmosrpc.NewHub(clientConfigs, logger, time.Duration(config.RPCTimeOutInSeconds)*time.Second)
	dbClient, err := db.NewClient(config.DBConnectionString)
	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("DB: Error creating DB client. Error: %v\n", err)
		return nil, err
	}

	sdkConfig := types.GetConfig()
	sdkConfig.SetCoinType(initiaapp.CoinType)

	accountPubKeyPrefix := initiaapp.AccountAddressPrefix + "pub"
	validatorAddressPrefix := initiaapp.AccountAddressPrefix + "valoper"
	validatorPubKeyPrefix := initiaapp.AccountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := initiaapp.AccountAddressPrefix + "valcons"
	consNodePubKeyPrefix := initiaapp.AccountAddressPrefix + "valconspub"

	sdkConfig.SetBech32PrefixForAccount(initiaapp.AccountAddressPrefix, accountPubKeyPrefix)
	sdkConfig.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	sdkConfig.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
	sdkConfig.SetAddressVerifier(initiaapp.VerifyAddressLen())
	sdkConfig.Seal()

	// log the configuration: To be removed
	logger.Info().Msgf("Configuration: %+v\n", config)

	return &ValidatorCron{
		rpcClient:         rpcClient,
		dbClient:          dbClient,
		config:            config,
		interfaceRegistry: initiaapp.MakeEncodingConfig().InterfaceRegistry,
	}, nil
}

func createCronHubAndContext(name string) (*sentry.Hub, context.Context) {
	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("function_name", name)
	})

	ctx := sentry.SetHubOnContext(context.Background(), hub)
	return hub, ctx
}

func (v *ValidatorCron) Run() {
	c := cron.New()
	updateValidatorsHub, updateValidatorsCtx := createCronHubAndContext("updateValidators")
	c.AddFunc(fmt.Sprintf("@every %ds", v.config.ValidatorUpdateIntervalInSeconds), func() {
		err := updateValidators(
			updateValidatorsCtx,
			v.dbClient,
			v.rpcClient,
			v.interfaceRegistry,
			v.config,
		)

		if err != nil {
			sentry_integration.CaptureException(updateValidatorsHub, err, sentry.LevelError)
		}
	})

	updateLatest100BlockValidatorUptimeHub, updateLatest100BlockValidatorUptimeCtx := createCronHubAndContext("updateLatest100BlockValidatorUptime")
	c.AddFunc(fmt.Sprintf("@every %ds", v.config.ValidatorUptimeUpdateIntervalInSeconds), func() {
		err := updateLatest100BlockValidatorUptime(updateLatest100BlockValidatorUptimeCtx, v.dbClient, v.config)

		if err != nil {
			sentry_integration.CaptureException(updateLatest100BlockValidatorUptimeHub, err, sentry.LevelError)
		}
	})

	updateValidatorHistoricalPowerHub, updateValidatorHistoricalPowerCtx := createCronHubAndContext("updateValidatorHistoricalPower")
	c.AddFunc("0 * * * *", func() {
		err := updateValidatorHistoricalPower(updateValidatorHistoricalPowerCtx, v.dbClient, v.rpcClient, v.config)

		if err != nil {
			sentry_integration.CaptureException(updateValidatorHistoricalPowerHub, err, sentry.LevelError)
		}
	})

	pruneCommitSignaturesHub, pruneCommitSignaturesCtx := createCronHubAndContext("pruneCommitSignatures")
	c.AddFunc("0 * * * *", func() {
		err := pruneCommitSignatures(pruneCommitSignaturesCtx, v.dbClient, v.config)

		if err != nil {
			sentry_integration.CaptureException(pruneCommitSignaturesHub, err, sentry.LevelError)
		}
	})

	// Start the Cron job scheduler
	c.Start()

	// wait for the scheduler to be terminated by signals
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer sentry.Flush(2 * time.Second)
	<-ctx.Done()

	// wait for all running crons to finish
	stopCtx := c.Stop()
	<-stopCtx.Done()

	v.dbClient.Close()
}
