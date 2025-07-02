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
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
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
	KafkaBootstrapServer     string
	KafkaTopic               string
	KafkaAPIKey              string
	KafkaAPISecret           string
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

	var producer *mq.Producer

	if config.Environment == "local" {
		producer, err = mq.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers": config.KafkaBootstrapServer,
			"client.id":         config.Chain + "-informative-indexer-sweeper",
			"acks":              "all",
			"linger.ms":         200,
			"security.protocol": "PLAINTEXT",
			"message.max.bytes": 7340032,
			"compression.codec": "lz4",
		})
	} else {
		producer, err = mq.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers": config.KafkaBootstrapServer,
			"client.id":         config.Chain + "-informative-indexer-sweeper",
			"acks":              "all",
			"linger.ms":         200,
			"security.protocol": "SASL_SSL",
			"sasl.mechanisms":   "PLAIN",
			"sasl.username":     config.KafkaAPIKey,
			"sasl.password":     config.KafkaAPISecret,
			"message.max.bytes": 7340032,
			"compression.codec": "lz4",
		})
	}

	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("Kafka: Error creating producer: %v\n", err)
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
	s.producer.ListenToKafkaProduceEvents(logger)

	height, err := db.GetLatestBlockHeight(context.Background(), s.dbClient)
	if err != nil {
		logger.Error().Msgf("DB: Error getting latest block height: %v\n", err)
		panic(err)
	}

	// height = 556037 - 5
	height = 556331 - 5
	for {
		select {
		case <-signalCtx.Done():
			return
		default:
			height = height + 1
			if s.config.RebalanceInterval != 0 && height%s.config.RebalanceInterval == 0 {
				err := s.rpcClient.Rebalance(context.Background())
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
			ctx := sentry.SetHubOnContext(context.Background(), localHub)
			s.GetBlockFromRPCAndProduce(ctx, height)
		}
	}
}

func (s *Sweeper) GetBlockFromRPCAndProduce(parentCtx context.Context, height int64) {
	logger.Info().Msgf("RPC: Getting data from block_results: %d", height)

	transaction, ctx := sentry_integration.StartSentryTransaction(parentCtx, "Sweep", "Sweep block_results from RPC and produce to Kafka")
	defer transaction.Finish()

	block := s.GetBlock(ctx, height)
	blockResult := s.GetBlockResults(ctx, height)

	err := s.MakeAndSendBlockResultMsg(ctx, block, blockResult)
	if err != nil {
		logger.Fatal().Msgf("Kafka: Error producing message at height: %d. Error: %v\n", height, err)
	}
}

func (s *Sweeper) GetBlock(ctx context.Context, height int64) *coretypes.ResultBlock {
	retryCount := 0
	hub := sentry.GetHubFromContext(ctx)
	for {
		block, err := s.rpcClient.Block(ctx, &height)
		// TODO: make a retry count function
		if err != nil {
			if retryCount == 3 {
				sentry_integration.CaptureException(hub, err, sentry.LevelError)
			}
			logger.Error().Msgf("RPC: Error getting block %d: %v\n", height, err)
			time.Sleep(time.Second)

			retryCount++
			continue
		}
		return block
	}
}

func (s *Sweeper) GetBlockResults(ctx context.Context, height int64) *coretypes.ResultBlockResults {
	retryCount := 0
	hub := sentry.GetHubFromContext(ctx)
	for {
		blockResult, err := s.rpcClient.BlockResults(ctx, &height)
		if err != nil {
			if retryCount == 3 {
				sentry_integration.CaptureException(hub, err, sentry.LevelError)
			}
			logger.Error().Msgf("RPC: Error getting block results %d: %v\n", height, err)
			time.Sleep(time.Second)

			retryCount++
			continue
		}
		return blockResult
	}
}

func (s *Sweeper) MakeAndSendBlockResultMsg(ctx context.Context, block *coretypes.ResultBlock, blockResult *coretypes.ResultBlockResults) error {
	span, _ := sentry_integration.StartSentrySpan(ctx, "MakeAndSendBlockResultMsg", "Make and send block results")
	defer span.Finish()

	blockResultMsgBytes, err := mq.NewBlockResultMsgBytes(block, blockResult)
	if err != nil {
		logger.Error().Msgf("Failed to marshal into block result message: %v\n", err)
		return err
	}

	s.producer.ProduceWithClaimCheck(&mq.ProduceWithClaimCheckInput{
		Topic:                   s.config.KafkaTopic,
		Key:                     fmt.Appendf(nil, "%s_%d", mq.NEW_BLOCK_RESULTS_KAFKA_MESSAGE_KEY, blockResult.Height),
		MessageInBytes:          blockResultMsgBytes,
		ClaimCheckKey:           fmt.Appendf(nil, "%s_%d", mq.NEW_BLOCK_RESULTS_CLAIM_CHECK_KAFKA_MESSAGE_KEY, blockResult.Height),
		ClaimCheckThresholdInMB: s.config.ClaimCheckThresholdInMB,
		ClaimCheckBucket:        s.config.ClaimCheckBucket,
		ClaimCheckObjectPath:    fmt.Sprintf("%d", blockResult.Height),
		StorageClient:           s.storageClient,
		Headers:                 []kafka.Header{{Key: "height", Value: fmt.Appendf(nil, "%d", blockResult.Height)}},
	}, logger)

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

	s.producer.Flush(30000)
	s.producer.Close()
}
