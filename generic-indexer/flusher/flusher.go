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
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/getsentry/sentry-go"
	initiaapp "github.com/initia-labs/initia/app"
	"github.com/initia-labs/initia/app/params"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/initia-labs/core-indexer/generic-indexer/common"
	"github.com/initia-labs/core-indexer/generic-indexer/db"
	"github.com/initia-labs/core-indexer/generic-indexer/storage"
	"github.com/initia-labs/core-indexer/pkg/cosmosrpc"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
)

var logger *zerolog.Logger

type Flusher struct {
	consumer      *mq.Consumer
	producer      *mq.Producer
	rpcClient     cosmosrpc.CosmosJSONRPCHub
	dbClient      *pgxpool.Pool
	storageClient storage.StorageClient
	validators    map[string]db.ValidatorRelation

	config         *FlusherConfig
	encodingConfig params.EncodingConfig
}

type FlusherConfig struct {
	// Worker ID. There supposed to be multiple workers running in parallel
	ID string

	RPCEndpoints string
	NumWorkers   int64

	// Chain ID to sweep
	Chain string

	DBConnectionString string

	// Kafka config
	KafkaBootstrapServer    string
	KafkaBlockTopic         string
	KafkaTxResponseTopic    string
	KafkaAPIKey             string
	KafkaAPISecret          string
	KafkaBlockConsumerGroup string

	// Claim check config
	ClaimCheckThresholdInMB       int64
	BlockClaimCheckBucket         string
	LCDTxResponseClaimCheckBucket string
	BlockResultsClaimCheckBucket  string

	// AWS
	AWSAccessKey string
	AWSSecretKey string

	// Functionality control
	DisableLCDTXResponse              bool
	DisableIndexingAccountTransaction bool

	Environment         string
	RebalanceInterval   int64
	RPCTimeOutInSeconds int64

	SentryDSN                string
	CommitSHA                string
	SentryProfilesSampleRate float64
	SentryTracesSampleRate   float64
}

func NewFlusher(config *FlusherConfig) (*Flusher, error) {
	logger = zerolog.Ctx(log.With().Str("component", "generic-indexer-flusher").
		Str("chain", config.Chain).
		Str("id", config.ID).
		Str("environment", config.Environment).
		Str("commit_sha", config.CommitSHA).
		Logger().WithContext(context.Background()),
	)

	sentryClientOptions := sentry.ClientOptions{
		Dsn:                config.SentryDSN,
		ServerName:         config.Chain + "-generic-indexer-flusher",
		EnableTracing:      true,
		ProfilesSampleRate: config.SentryProfilesSampleRate,
		TracesSampleRate:   config.SentryTracesSampleRate,
		Environment:        config.Environment,
		Release:            config.CommitSHA,
		Tags: map[string]string{
			"chain":       config.Chain,
			"environment": config.Environment,
			"component":   "generic-indexer-flusher",
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

	var rpcEndpoints common.RPCEndpoints
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
		logger.Fatal().Msgf("RPC: Error rebalancing RPC endpoints: %v\n", err)
		return nil, err
	}

	consumer, err := mq.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":    config.KafkaBootstrapServer,
		"group.id":             config.KafkaBlockConsumerGroup,
		"client.id":            config.KafkaBlockConsumerGroup + "-" + config.ID,
		"enable.auto.commit":   false,
		"auto.offset.reset":    "earliest",
		"security.protocol":    "SASL_SSL",
		"sasl.mechanisms":      "PLAIN",
		"sasl.username":        config.KafkaAPIKey,
		"sasl.password":        config.KafkaAPISecret,
		"max.poll.interval.ms": 600000,
	})

	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("Kafka: Error creating consumer. Error: %v\n", err)
		return nil, err
	}

	producer, err := mq.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": config.KafkaBootstrapServer,
		"client.id":         config.KafkaBlockConsumerGroup + "-" + config.ID,
		"acks":              "all",
		"linger.ms":         200,
		"security.protocol": "SASL_SSL",
		"sasl.mechanisms":   "PLAIN",
		"sasl.username":     config.KafkaAPIKey,
		"sasl.password":     config.KafkaAPISecret,
		"message.max.bytes": 7340032,
		"compression.codec": "lz4",
	})

	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("Kafka: Error creating producer. Error: %v\n", err)
		return nil, err
	}

	dbClient, err := db.NewClient(config.DBConnectionString)
	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("DB: Error creating DB client. Error: %v\n", err)
		return nil, err
	}

	storageClient, err := storage.NewS3Client(config.AWSAccessKey, config.AWSSecretKey)
	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("Storage: Error creating storage client. Error: %v\n", err)
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
	return &Flusher{
		consumer:       consumer,
		producer:       producer,
		rpcClient:      rpcClient,
		dbClient:       dbClient,
		storageClient:  storageClient,
		config:         config,
		encodingConfig: initiaapp.MakeEncodingConfig(),
		validators:     make(map[string]db.ValidatorRelation),
	}, nil
}

func (f Flusher) WithCustomRPCClient(rpcClient cosmosrpc.CosmosJSONRPCHub) *Flusher {
	f.rpcClient = rpcClient
	return &f
}

func (f *Flusher) parseBlockAndRebalanceRPCClient(parentCtx context.Context, blockBytes []byte) (mq.BlockResultMsg, error) {
	span, ctx := sentry_integration.StartSentrySpan(parentCtx, "parseBlockAndRebalanceRPCClient", "Parsing block and rebalancing RPC clients")
	defer span.Finish()

	var blockMsg mq.BlockResultMsg
	err := json.Unmarshal(blockBytes, &blockMsg)
	if err != nil {
		logger.Error().Msgf("Error unmarshalling message: %v", err)
		return blockMsg, err
	}

	if f.config.RebalanceInterval != 0 && blockMsg.Height%f.config.RebalanceInterval == 0 {
		err := f.rpcClient.Rebalance(ctx)
		if err != nil {
			logger.Error().Msgf("Error rebalancing clients: %v", err)
			return blockMsg, err
		}
		actives := f.rpcClient.GetActiveClients()
		for _, active := range actives {
			logger.Info().Msgf("Active client url: %s, latest height: %d", active.Client.GetIdentifier(), active.Height)
		}
	}
	return blockMsg, err
}

func (f *Flusher) processUntilSucceeds(ctx context.Context, blockMsg mq.BlockResultMsg) error {
	// Process the block until success
	for {
		err := f.processBlock(ctx, &blockMsg)
		if err != nil {
			if errors.Is(err, ErrorNonRetryable) {
				return err
			}

			sentry_integration.CaptureCurrentHubException(err, sentry.LevelWarning)
			logger.Error().Msgf("Error processing block: %v, retrying...", err)
			continue
		}
		break
	}

	// Validate the block until success
	for {
		err := f.processValidator(ctx, &blockMsg)
		if err != nil {
			if errors.Is(err, ErrorNonRetryable) {
				return err
			}
			sentry_integration.CaptureCurrentHubException(err, sentry.LevelWarning)
			logger.Error().Msgf("Error validating block: %v, retrying...", err)
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
	blockMsg, err := f.parseBlockAndRebalanceRPCClient(ctx, messageValue)
	if err != nil {
		logger.Error().Msgf("Error processing block message: %v", err)
		return err
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("height", fmt.Sprint(blockMsg.Height))
	})

	err = f.processUntilSucceeds(ctx, blockMsg)
	if err != nil {
		logger.Error().Msgf("Error processing block: %v", err)
		return err
	}

	return nil
}

func (f *Flusher) close() {
	f.dbClient.Close()

	f.producer.Flush(30000)
	f.producer.Close()

	f.consumer.Close()
}

func (f *Flusher) loadAllValidatorToCache() error {
	ctx := context.Background()
	dbTx, err := f.dbClient.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	defer dbTx.Rollback(ctx)

	vals, err := db.QueryValidatorRelations(ctx, dbTx)
	if err != nil {
		return err
	}
	for _, val := range vals {
		f.validators[val.ConsensusAddress] = val
	}

	return nil
}

func (f *Flusher) StartFlushing(stopCtx context.Context) {
	logger.Info().Msgf("Starting flusher...")

	f.producer.ListenToKafkaProduceEvents(logger)

	err := f.consumer.SubscribeTopics([]string{f.config.KafkaBlockTopic}, nil)
	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("Failed to subscribe to topic: %s\n", err)
	}

	logger.Info().Msgf("Subscribed to topic: %s\n", f.config.KafkaBlockTopic)

	err = f.loadAllValidatorToCache()
	if err != nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("Error loading validators to cache: %v", err)
	}

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

			transaction, ctx := sentry_integration.StartSentryTransaction(ctx, "Flush", "Process and flush generic block messages")
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
