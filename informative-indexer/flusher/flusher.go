package flusher

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
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/getsentry/sentry-go"
	initiaapp "github.com/initia-labs/initia/app"
	"github.com/initia-labs/initia/app/params"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/cosmosrpc"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/sdkconfig"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
	"github.com/initia-labs/core-indexer/pkg/storage"
)

var logger *zerolog.Logger

type Flusher struct {
	consumer      *mq.Consumer
	producer      *mq.Producer
	dbClient      *gorm.DB
	storageClient storage.Client
	config        *Config

	encodingConfig     *params.EncodingConfig
	stateUpdateManager *StateUpdateManager
	rpcClient          cosmosrpc.CosmosJSONRPCHub
	dbBatchInsert      *DBBatchInsert
}

type Config struct {
	// ID for the current flusher
	ID string

	// Chain id
	Chain string

	DBConnectionString string

	RPCEndpoints        string
	RPCTimeoutInSeconds int64

	// Kafka config
	KafkaBootstrapServer           string
	KafkaBlockResultsTopic         string
	KafkaAPIKey                    string
	KafkaAPISecret                 string
	KafkaBlockResultsConsumerGroup string

	// Claim check config
	ClaimCheckThresholdInMB      int64
	BlockResultsClaimCheckBucket string

	Environment              string
	SentryDSN                string
	CommitSHA                string
	SentryProfilesSampleRate float64
	SentryTracesSampleRate   float64
}

func NewFlusher(config *Config) (*Flusher, error) {
	logger = zerolog.Ctx(log.With().Str("component", "informative-indexer-flusher").
		Str("chain", config.Chain).
		Str("id", config.ID).
		Str("environment", config.Environment).
		Str("commit_sha", config.CommitSHA).
		Logger().WithContext(context.Background()),
	)

	sentryClientOptions := sentry.ClientOptions{
		Dsn:                config.SentryDSN,
		ServerName:         config.Chain + "-informative-indexer-flusher",
		EnableTracing:      true,
		ProfilesSampleRate: config.SentryProfilesSampleRate,
		TracesSampleRate:   config.SentryTracesSampleRate,
		Environment:        config.Environment,
		Release:            config.CommitSHA,
		Tags: map[string]string{
			"chain":       config.Chain,
			"environment": config.Environment,
			"component":   "informative-indexer-flusher",
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

	var consumer *mq.Consumer

	if config.Environment == "local" {
		consumer, err = mq.NewConsumer(&kafka.ConfigMap{
			"bootstrap.servers":    config.KafkaBootstrapServer,
			"group.id":             config.KafkaBlockResultsConsumerGroup,
			"client.id":            config.KafkaBlockResultsConsumerGroup + "-" + config.ID,
			"enable.auto.commit":   false,
			"auto.offset.reset":    "earliest",
			"security.protocol":    "PLAINTEXT",
			"max.poll.interval.ms": 600000,
		})
	} else {
		consumer, err = mq.NewConsumer(&kafka.ConfigMap{
			"bootstrap.servers":    config.KafkaBootstrapServer,
			"group.id":             config.KafkaBlockResultsConsumerGroup,
			"client.id":            config.KafkaBlockResultsConsumerGroup + "-" + config.ID,
			"enable.auto.commit":   false,
			"auto.offset.reset":    "earliest",
			"security.protocol":    "SASL_SSL",
			"sasl.mechanisms":      "PLAIN",
			"sasl.username":        config.KafkaAPIKey,
			"sasl.password":        config.KafkaAPISecret,
			"max.poll.interval.ms": 600000,
		})
	}

	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("Kafka: Error creating consumer: %v\n", err)
		return nil, err
	}

	var producer *mq.Producer

	if config.Environment == "local" {
		producer, err = mq.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers": config.KafkaBootstrapServer,
			"client.id":         config.KafkaBlockResultsConsumerGroup + "-" + config.ID,
			"acks":              "all",
			"linger.ms":         200,
			"security.protocol": "PLAINTEXT",
			"message.max.bytes": 7340032,
			"compression.codec": "lz4",
		})
	} else {
		producer, err = mq.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers": config.KafkaBootstrapServer,
			"client.id":         config.KafkaBlockResultsConsumerGroup + "-" + config.ID,
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
	return &Flusher{
		consumer:       consumer,
		producer:       producer,
		dbClient:       dbClient,
		storageClient:  storageClient,
		config:         config,
		encodingConfig: &encodingConfig,
		rpcClient:      rpcClient,
	}, nil
}

func (f *Flusher) parseBlockResults(parentCtx context.Context, blockResultsBytes []byte) (mq.BlockResultMsg, error) {
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

func (f *Flusher) processUntilSucceeds(ctx context.Context, blockResultsMsg mq.BlockResultMsg) error {
	// Process the block_results until success
	for {
		err := f.processBlockResults(ctx, &blockResultsMsg)
		if err != nil {
			if errors.Is(err, ErrorNonRetryable) {
				return err
			}

			sentry_integration.CaptureCurrentHubException(err, sentry.LevelWarning)
			logger.Error().Msgf("Error processing block_results: %v, retrying...", err)
			continue
		}
		break
	}

	// Validate the block validators until success
	// TODO: Add flag for disable
	for {
		err := f.processValidator(ctx, &blockResultsMsg)
		if err != nil {
			if errors.Is(err, ErrorNonRetryable) {
				return err
			}

			sentry_integration.CaptureCurrentHubException(err, sentry.LevelWarning)
			logger.Error().Msgf("Error validating block validators: %v, retrying...", err)
			continue
		}
		break
	}

	return nil
}

func (f *Flusher) processClaimCheckMessage(key []byte, messageValue []byte) ([]byte, error) {
	if strings.HasPrefix(string(key), mq.NEW_BLOCK_RESULTS_CLAIM_CHECK_KAFKA_MESSAGE_KEY) {
		var claimCheckBlockResultsMsg mq.ClaimCheckMsg
		err := json.Unmarshal(messageValue, &claimCheckBlockResultsMsg)
		if err != nil {
			logger.Fatal().Msgf("Error unmarshalling message: %v", err)
			return nil, err
		}

		var claimCheckBlockResultsMsgBytes []byte
		for {
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

func (f *Flusher) processKafkaMessage(ctx context.Context, message *kafka.Message) error {
	messageValue, err := f.processClaimCheckMessage(message.Key, message.Value)
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

func (f *Flusher) close() {
	sqlDB, err := f.dbClient.DB()
	if err == nil {
		sqlDB.Close()
	}

	f.producer.Flush(30000)
	f.producer.Close()

	f.consumer.Close()
}

func (f *Flusher) StartFlushing(stopCtx context.Context) {
	logger.Info().Msgf("Starting flusher...")

	f.producer.ListenToKafkaProduceEvents(logger)

	err := f.consumer.SubscribeTopics([]string{f.config.KafkaBlockResultsTopic}, nil)
	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("Failed to subscribe to topic: %s\n", err)
	}

	logger.Info().Msgf("Subscribed to topic: %s\n", f.config.KafkaBlockResultsTopic)

	for {
		select {
		case <-stopCtx.Done():
			return
		default:
			ctx := context.Background()

			message, err := f.consumer.ReadMessage(10 * time.Second)
			if err != nil {
				if err.(kafka.Error).IsTimeout() {
					continue
				}

				sentry_integration.CaptureCurrentHubException(err, sentry.LevelWarning)
				logger.Error().Msgf("Error reading message: %v", err)
				continue
			}

			sentry.ConfigureScope(func(scope *sentry.Scope) {
				scope.SetTag("id", f.config.ID)
				scope.SetTag("partition", fmt.Sprint(message.TopicPartition.Partition))
				scope.SetTag("offset", message.TopicPartition.Offset.String())
			})

			transaction, ctx := sentry_integration.StartSentryTransaction(ctx, "Flush", "Process and flush informative block_results messages")
			err = f.processKafkaMessage(ctx, message)
			if err != nil {
				sentry_integration.CaptureCurrentHubException(err, sentry.LevelError)
				logger.Warn().Msgf("Producing message to DLQ: %d, %d, %v", message.TopicPartition.Partition, message.TopicPartition.Offset, err)
				f.producer.ProduceToDLQ(message, err, logger)
			}

			_, err = f.consumer.CommitMessage(message)
			if err != nil {
				sentry_integration.CaptureCurrentHubException(err, sentry.LevelError)
				logger.Error().Msgf("Non-retryable Error committing message: %v", err)
			}
			transaction.Finish()
		}
	}
}

func (f *Flusher) Flush() {
	// graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer sentry.Flush(2 * time.Second)

	f.StartFlushing(ctx)

	logger.Info().Msgf("Shutting down ...")
	f.close()
}
