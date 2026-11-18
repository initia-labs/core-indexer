package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/certifi/gocertifi"
	"github.com/getsentry/sentry-go"
	initiaapp "github.com/initia-labs/initia/app"
	"github.com/initia-labs/initia/app/params"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/cacher"
	"github.com/initia-labs/core-indexer/informative-indexer/indexer/processors"
	accountprocessor "github.com/initia-labs/core-indexer/informative-indexer/indexer/processors/account"
	bankprocessor "github.com/initia-labs/core-indexer/informative-indexer/indexer/processors/bank"
	ibcprocessor "github.com/initia-labs/core-indexer/informative-indexer/indexer/processors/ibc"
	moveprocessor "github.com/initia-labs/core-indexer/informative-indexer/indexer/processors/move"
	proposalprocessor "github.com/initia-labs/core-indexer/informative-indexer/indexer/processors/proposal"
	validatorprocessor "github.com/initia-labs/core-indexer/informative-indexer/indexer/processors/validator"
	statetracker "github.com/initia-labs/core-indexer/informative-indexer/indexer/state-tracker"
	"github.com/initia-labs/core-indexer/pkg/cosmosrpc"
	"github.com/initia-labs/core-indexer/pkg/db"
	indexererrors "github.com/initia-labs/core-indexer/pkg/errors"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/sdkconfig"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
	"github.com/initia-labs/core-indexer/pkg/storage"
)

var logger *zerolog.Logger

type Indexer struct {
	consumer      *mq.Consumer
	producer      *mq.Producer
	dbClient      *gorm.DB
	storageClient storage.Client
	config        *Config

	encodingConfig *params.EncodingConfig
	rpcClient      cosmosrpc.CosmosJSONRPCHub

	stateUpdateManager *statetracker.StateUpdateManager
	dbBatchInsert      *statetracker.DBBatchInsert
	cacher             *cacher.Cacher

	processors []processors.Processor
}

type Config struct {
	// ID for the current indexer
	ID string

	// Chain id
	Chain string

	DBConnectionString string

	RPCEndpoints        string
	RPCTimeoutInSeconds int64

	// Pub/Sub config
	PubSubProjectID          string
	PubSubEndpoint           string
	TxResponseTopic          string
	BlockResultsSubscription string

	// Claim check config
	ClaimCheckThresholdInMB       int64
	BlockResultsClaimCheckBucket  string
	LCDTxResponseClaimCheckBucket string

	Environment              string
	SentryDSN                string
	CommitSHA                string
	SentryProfilesSampleRate float64
	SentryTracesSampleRate   float64
}

func NewIndexer(config *Config) (*Indexer, error) {
	logger = zerolog.Ctx(log.With().Str("component", "informative-indexer-indexer").
		Str("chain", config.Chain).
		Str("id", config.ID).
		Str("environment", config.Environment).
		Str("commit_sha", config.CommitSHA).
		Logger().WithContext(context.Background()),
	)

	sentryClientOptions := sentry.ClientOptions{
		Dsn:                config.SentryDSN,
		ServerName:         config.Chain + "-informative-indexer-indexer",
		EnableTracing:      true,
		ProfilesSampleRate: config.SentryProfilesSampleRate,
		TracesSampleRate:   config.SentryTracesSampleRate,
		Environment:        config.Environment,
		Release:            config.CommitSHA,
		Tags: map[string]string{
			"chain":       config.Chain,
			"environment": config.Environment,
			"component":   "informative-indexer-indexer",
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

	consumer, err := mq.NewConsumer(context.Background(), config.PubSubProjectID, config.BlockResultsSubscription, config.PubSubEndpoint)
	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("Pub/Sub: Error creating consumer: %v\n", err)
		return nil, err
	}

	producer, err := mq.NewProducer(context.Background(), config.PubSubProjectID, config.PubSubEndpoint)
	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("Pub/Sub: Error creating producer: %v\n", err)
		return nil, err
	}

	dbClient, err := db.NewClient(config.DBConnectionString)
	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("DB: Error creating DB client: %v\n", err)
		return nil, err
	}

	var storageClient storage.Client

	if config.Environment == "local" {
		storageClient, err = storage.NewGCSFakeClient()
		if err != nil {
			logger.Info().Msgf("Local: Error creating storage client: %v\n", err)
			return nil, err
		}
	} else {
		storageClient, err = storage.NewGCSClient()
		if err != nil {
			sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
			logger.Fatal().Msgf("Storage: Error creating storage client: %v\n", err)
			return nil, err
		}
	}

	if config.RPCEndpoints == "" {
		sentry_integration.CaptureCurrentHubException(errors.New("RPC: No RPC endpoints provided"), sentry.LevelFatal)
		logger.Fatal().Msgf("RPC: No RPC endpoints provided\n")
		return nil, fmt.Errorf("RPC: No RPC endpoints provided")
	}

	var rpcEndpoints mq.RPCEndpoints
	err = json.Unmarshal([]byte(config.RPCEndpoints), &rpcEndpoints)
	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
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

	rpcClient := cosmosrpc.NewHub(clientConfigs, logger, time.Duration(config.RPCTimeoutInSeconds)*time.Second)
	err = rpcClient.Rebalance(context.Background())
	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("RPC: Error Rebalancing RPC endpoints: %v\n", err)
		return nil, err
	}

	sdkconfig.ConfigureSDK()
	encodingConfig := initiaapp.MakeEncodingConfig()
	return &Indexer{
		consumer:       consumer,
		producer:       producer,
		dbClient:       dbClient,
		storageClient:  storageClient,
		config:         config,
		encodingConfig: &encodingConfig,
		rpcClient:      rpcClient,
		cacher:         cacher.NewCacher(),
		processors: []processors.Processor{
			&accountprocessor.Processor{},
			&bankprocessor.Processor{},
			&ibcprocessor.Processor{},
			&moveprocessor.Processor{},
			&proposalprocessor.Processor{},
			&validatorprocessor.Processor{},
		},
	}, nil
}

func (f *Indexer) RunPatcher(ctx context.Context) {
	err := runPatcher(ctx, f.dbClient, f.config.Chain)
	if err != nil {
		logger.Error().Msgf("Error running patcher: %v", err)
	}
}

func (f *Indexer) parseBlockResults(parentCtx context.Context, blockResultsBytes []byte) (mq.BlockResultMsg, error) {
	span, _ := sentry_integration.StartSentrySpan(parentCtx, "parseBlockResults", "Parsing block_results")
	defer span.Finish()

	var blockResultsMsg mq.BlockResultMsg
	err := json.Unmarshal(blockResultsBytes, &blockResultsMsg)
	if err != nil {
		logger.Error().Msgf("Error unmarshalling message: %v", err)
		return blockResultsMsg, err
	}

	return blockResultsMsg, err
}

func (f *Indexer) processUntilSucceeds(ctx context.Context, blockResults mq.BlockResultMsg) error {
	proposer, ok := f.cacher.GetValidatorByConsAddr(blockResults.ProposerConsensusAddress)
	if !ok {
		logger.Error().Msgf("Failed to get proposer operator address")
	}

	// Process the block_results until success
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := f.processBlockResults(ctx, &blockResults, &proposer)
		if err != nil {
			if errors.Is(err, indexererrors.ErrorNonRetryable) {
				return err
			}

			sentry_integration.CaptureCurrentHubException(err, sentry.LevelWarning)
			logger.Error().Msgf("Error processing block_results: %v, retrying...", err)
			if errors.Is(err, errValidatorProcessing) || errors.Is(err, errTxResponsePublish) {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(10 * time.Second):
				}
			}
			continue
		}
		break
	}

	return nil
}

func (f *Indexer) processClaimCheckMessage(ctx context.Context, key []byte, messageValue []byte) ([]byte, error) {
	if strings.HasPrefix(string(key), mq.NEW_BLOCK_RESULTS_CLAIM_CHECK_MESSAGE_KEY) {
		var claimCheckBlockResultsMsg mq.ClaimCheckMsg
		err := json.Unmarshal(messageValue, &claimCheckBlockResultsMsg)
		if err != nil {
			logger.Fatal().Msgf("Error unmarshalling message: %v", err)
			return nil, err
		}

		var claimCheckBlockResultsMsgBytes []byte
		for {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			claimCheckBlockResultsMsgBytes, err = f.storageClient.ReadFile(f.config.BlockResultsClaimCheckBucket, claimCheckBlockResultsMsg.ObjectPath)
			if err == nil {
				break
			}

			logger.Error().Msgf("Error reading block_results from storage: %v", err)
		}

		messageValue = claimCheckBlockResultsMsgBytes
	}
	return messageValue, nil
}

func (f *Indexer) processMessage(ctx context.Context, message *mq.Message) error {
	messageValue, err := f.processClaimCheckMessage(ctx, []byte(message.Attributes["key"]), message.Data)
	if err != nil {
		logger.Error().Msgf("Error processing claim check message: %v", err)
		return err
	}
	blockResultsMsg, err := f.parseBlockResults(ctx, messageValue)
	if err != nil {
		logger.Error().Msgf("Error processing block_results message: %v", err)
		return err
	}

	latestInformativeBlockHeight, err := db.GetLatestInformativeBlockHeight(ctx, f.dbClient)
	if err != nil {
		return err
	}
	if blockResultsMsg.Height <= latestInformativeBlockHeight {
		logger.Info().Msgf("Skipping block_results message at height %d because it's already processed", blockResultsMsg.Height)
		return nil
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("height", fmt.Sprint(blockResultsMsg.Height))
	})

	err = f.processUntilSucceeds(ctx, blockResultsMsg)
	if err != nil {
		logger.Error().Msgf("Error processing block_results: %v", err)
		return err
	}

	return nil
}

func (f *Indexer) close() {
	sqlDB, err := f.dbClient.DB()
	if err == nil {
		sqlDB.Close()
	}

	f.producer.Close()

	f.consumer.Close()
}

func (f *Indexer) StartIndexing(stopCtx context.Context) {
	logger.Info().Msgf("Starting indexer...")

	trackingInit, err := db.IsTrackingInit(context.Background(), f.dbClient)
	if err != nil {
		logger.Fatal().Msgf("Error checking if tracking is initialized: %v", err)
		panic(err)
	}

	if !trackingInit {
		logger.Info().Msgf("Initializing tracking...")
		err := f.StartFromGenesis(stopCtx, logger)
		if err != nil {
			logger.Fatal().Msgf("Error starting from genesis: %v", err)
			panic(err)
		}
	}
	logger.Info().Str("subscription", f.config.BlockResultsSubscription).Msg("Starting Pub/Sub consumer")
	validatorAddresses, err := db.QueryValidatorAddresses(f.dbClient)
	if err != nil {
		logger.Fatal().Msgf("Error getting validators from db: %v", err)
		return
	}
	f.cacher.SetValidatorAddresses(validatorAddresses)

	err = f.consumer.Receive(stopCtx, func(ctx context.Context, message *mq.Message) error {
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("id", f.config.ID)
			scope.SetTag("message_id", message.ID)
		})
		transaction, ctx := sentry_integration.StartSentryTransaction(ctx, "Index", "Process informative-indexer block results")
		defer transaction.Finish()
		return f.processMessage(ctx, message)
	})
	if err != nil && stopCtx.Err() == nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Err(err).Msg("Pub/Sub consumer stopped")
	}
}

func (f *Indexer) Index(ctx context.Context) {
	// graceful shutdown
	defer sentry.Flush(2 * time.Second)

	f.StartIndexing(ctx)

	logger.Info().Msgf("Shutting down ...")
	f.close()
}
