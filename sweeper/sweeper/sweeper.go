package sweeper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/certifi/gocertifi"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/cosmosrpc"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
	"github.com/initia-labs/core-indexer/pkg/storage"
)

var logger *zerolog.Logger

type Sweeper struct {
	rpcClient     cosmosrpc.CosmosJSONRPCHub
	dbClient      *gorm.DB
	producer      *mq.Producer
	storageClient storage.Client
	config        *SweeperConfig
}

type SweeperConfig struct {
	RPCEndpoints             string
	RPCTimeOutInSeconds      int64
	Chain                    string
	DBConnectionString       string
	NumWorkers               int64
	RebalanceInterval        int64
	PubSubProjectID          string
	PubSubEndpoint           string
	BlockResultsTopics       []string
	ClaimCheckBucket         string
	ClaimCheckThresholdInMB  int64
	Environment              string
	CommitSHA                string
	SentryDSN                string
	SentryProfilesSampleRate float64
	SentryTracesSampleRate   float64
	MigrationsDir            string
}

func NewSweeper(config *SweeperConfig) (*Sweeper, error) {
	logger = zerolog.Ctx(log.With().Str("component", "informative-indexer-sweeper").Str("chain", config.Chain).Str("environment", config.Environment).Str("commit_sha", config.CommitSHA).Logger().WithContext(context.Background()))

	sentryClientOptions := sentry.ClientOptions{
		Dsn:                config.SentryDSN,
		ServerName:         config.Chain + "-informative-indexer-sweeper",
		EnableTracing:      true,
		ProfilesSampleRate: config.SentryProfilesSampleRate,
		TracesSampleRate:   config.SentryTracesSampleRate,
		Environment:        config.Environment,
		Release:            config.CommitSHA,
		Tags: map[string]string{
			"chain":       config.Chain,
			"environment": config.Environment,
			"component":   "informative-indexer-sweeper",
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

	rpcClient := cosmosrpc.NewHub(clientConfigs, logger, time.Duration(config.RPCTimeOutInSeconds)*time.Second)
	err = rpcClient.Rebalance(context.Background())
	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("RPC: Error Rebalancing RPC endpoints: %v\n", err)
		return nil, err
	}

	dbClient, err := db.NewClient(config.DBConnectionString)
	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("DB: Error creating DB client: %v\n", err)
		return nil, err
	}

	producer, err := mq.NewProducer(context.Background(), config.PubSubProjectID, config.PubSubEndpoint)
	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("Pub/Sub: Error creating producer: %v\n", err)
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

	return &Sweeper{
		rpcClient:     rpcClient,
		dbClient:      dbClient,
		producer:      producer,
		storageClient: storageClient,
		config:        config,
	}, nil
}

func (s *Sweeper) StartSweeping(signalCtx context.Context) {

	height, err := db.GetLatestInformativeBlockHeight(signalCtx, s.dbClient)
	if err != nil {
		if signalCtx.Err() != nil {
			return
		}
		logger.Error().Msgf("DB: Error getting latest block height: %v\n", err)
		panic(err)
	}

	retryCount := 0
	for {
		select {
		case <-signalCtx.Done():
			return
		default:
			height = height + 1
			if (retryCount == 0 && s.config.RebalanceInterval != 0 && height%s.config.RebalanceInterval == 0) || (retryCount > 0 && retryCount%10 == 0) {
				err := s.rpcClient.Rebalance(signalCtx)
				if err != nil {
					sentry_integration.CaptureCurrentHubException(err, sentry.LevelWarning)
					logger.Error().Msgf("Error rebalancing clients: %v", err)
				}
				actives := s.rpcClient.GetActiveClients()
				for _, active := range actives {
					logger.Info().Msgf("Active client url: %s, latest height: %d", active.Client.GetIdentifier(), active.Height)
				}
			}
			localHub := sentry.CurrentHub().Clone()
			localHub.ConfigureScope(func(scope *sentry.Scope) {
				scope.SetTag("height", fmt.Sprint(height))
			})
			ctx := sentry.SetHubOnContext(signalCtx, localHub)
			err := s.GetBlockFromRPCAndProduce(ctx, height)
			if err != nil {
				if signalCtx.Err() != nil {
					return
				}
				sentry_integration.CaptureCurrentHubException(err, sentry.LevelWarning)
				logger.Error().Err(err).Int64("height", height).Msg("Failed to sweep block; retrying")
				retryCount++
				height-- // Retry the same height after serialization or fetch failure.
				select {
				case <-signalCtx.Done():
					return
				case <-time.After(time.Second):
				}
			} else {
				retryCount = 0
			}
		}
	}
}

func (s *Sweeper) GetBlockFromRPCAndProduce(parentCtx context.Context, height int64) error {
	logger.Info().Msgf("RPC: Getting data from block_results: %d", height)

	transaction, ctx := sentry_integration.StartSentryTransaction(parentCtx, "Sweep", "Sweep block_results from RPC and produce to Pub/Sub")
	defer transaction.Finish()

	var blockResultMsgBytes []byte
	var resultHeight int64
	err := s.rpcClient.WithBlockResults(ctx, height, func(block *coretypes.ResultBlock, results *coretypes.ResultBlockResults) error {
		var err error
		blockResultMsgBytes, err = mq.NewBlockResultMsgBytes(block, results)
		resultHeight = results.Height
		return err
	})
	if err != nil {
		return err
	}

	for _, topic := range s.config.BlockResultsTopics {
		if err := s.producer.ProduceWithClaimCheck(ctx, &mq.ProduceWithClaimCheckInput{
			Topic:                   topic,
			Key:                     fmt.Appendf(nil, "%s_%d", mq.NEW_BLOCK_RESULTS_MESSAGE_KEY, resultHeight),
			MessageInBytes:          blockResultMsgBytes,
			ClaimCheckKey:           fmt.Appendf(nil, "%s_%d", mq.NEW_BLOCK_RESULTS_CLAIM_CHECK_MESSAGE_KEY, resultHeight),
			ClaimCheckThresholdInMB: s.config.ClaimCheckThresholdInMB,
			ClaimCheckBucket:        s.config.ClaimCheckBucket,
			ClaimCheckObjectPath:    fmt.Sprintf("%d", resultHeight),
			StorageClient:           s.storageClient,
			Attributes:              map[string]string{"height": fmt.Sprint(resultHeight)},
			OrderingKey:             s.config.Chain,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Sweeper) Sweep() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer sentry.Flush(2 * time.Second)

	if err := db.ApplyMigrationFiles(logger, s.dbClient, s.config.MigrationsDir); err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("Failed to apply migrations: %v\n", err)
		return
	}

	s.StartSweeping(ctx)

	logger.Info().Msgf("Stopping sweeper ...")
	s.close()
}

func (s *Sweeper) close() {
	db, err := s.dbClient.DB()
	if err == nil {
		db.Close()
	}

	s.producer.Close()
}
