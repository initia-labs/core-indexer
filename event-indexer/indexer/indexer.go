package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/certifi/gocertifi"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/cosmosrpc"
	"github.com/initia-labs/core-indexer/pkg/db"
	indexererrors "github.com/initia-labs/core-indexer/pkg/errors"
	"github.com/initia-labs/core-indexer/pkg/mq"
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

	rpcClient cosmosrpc.CosmosJSONRPCHub
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
	BlockResultsSubscription string

	// Claim check config
	ClaimCheckThresholdInMB      int64
	BlockResultsClaimCheckBucket string

	Environment              string
	SentryDSN                string
	CommitSHA                string
	SentryProfilesSampleRate float64
	SentryTracesSampleRate   float64
}

func New(config *Config) (*Indexer, error) {
	logger = zerolog.Ctx(log.With().Str("component", "event-indexer").
		Str("chain", config.Chain).
		Str("id", config.ID).
		Str("environment", config.Environment).
		Str("commit_sha", config.CommitSHA).
		Logger().WithContext(context.Background()),
	)

	sentryClientOptions := sentry.ClientOptions{
		Dsn:                config.SentryDSN,
		ServerName:         config.Chain + "-event-indexer",
		EnableTracing:      true,
		ProfilesSampleRate: config.SentryProfilesSampleRate,
		TracesSampleRate:   config.SentryTracesSampleRate,
		Environment:        config.Environment,
		Release:            config.CommitSHA,
		Tags: map[string]string{
			"chain":       config.Chain,
			"environment": config.Environment,
			"component":   "event-indexer",
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

	return &Indexer{
		consumer:      consumer,
		producer:      producer,
		dbClient:      dbClient,
		storageClient: storageClient,
		config:        config,
		rpcClient:     rpcClient,
	}, nil
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
	//	// Process the block_results until success
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := f.processBlockResults(ctx, &blockResults)
		if err != nil {
			if errors.Is(err, indexererrors.ErrorNonRetryable) {
				return err
			}

			sentry_integration.CaptureCurrentHubException(err, sentry.LevelWarning)
			logger.Error().Msgf("Error processing block_results: %v, retrying...", err)
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

	logger.Info().Str("subscription", f.config.BlockResultsSubscription).Msg("Starting Pub/Sub consumer")

	err := f.consumer.Receive(stopCtx, func(ctx context.Context, message *mq.Message) error {
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("id", f.config.ID)
			scope.SetTag("message_id", message.ID)
		})
		transaction, ctx := sentry_integration.StartSentryTransaction(ctx, "Index", "Process event-indexer block results")
		defer transaction.Finish()
		if err := f.processMessage(ctx, message); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			sentry_integration.CaptureCurrentHubException(err, sentry.LevelError)
			logger.Warn().Err(err).Str("message_id", message.ID).Msg("Producing message to DLQ")
			return f.producer.ProduceToDLQ(ctx, f.config.Chain, "event-indexer-block-results", message, err)
		}
		return nil
	})
	if err != nil && stopCtx.Err() == nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Err(err).Msg("Pub/Sub consumer stopped")
	}
}

func (f *Indexer) Run() {
	// graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer sentry.Flush(2 * time.Second)

	f.StartIndexing(ctx)

	logger.Info().Msgf("Shutting down ...")
	f.close()
}
