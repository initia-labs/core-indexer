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
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/initia-labs/core-indexer/generic-indexer/common"
	"github.com/initia-labs/core-indexer/generic-indexer/cosmosrpc"
	"github.com/initia-labs/core-indexer/generic-indexer/db"
	"github.com/initia-labs/core-indexer/generic-indexer/mq"
	"github.com/initia-labs/core-indexer/generic-indexer/storage"
)

var logger *zerolog.Logger

type Sweeper struct {
	producer      *mq.Producer
	rpcClient     cosmosrpc.CosmosJSONRPCHub
	dbClient      *pgxpool.Pool
	storageClient storage.StorageClient

	config *SweeperConfig
}

type SweeperConfig struct {
	RPCEndpoints          string
	MaxRetries            int64
	NumWorkers            int64
	PollSleepIntervalInMs int64

	// Chain ID to sweep
	Chain string

	DBConnectionString string

	// Kafka config
	KafkaBootstrapServer string
	KafkaTopic           string
	KafkaAPIKey          string
	KafkaAPISecret       string

	// AWS
	AWSAccessKey            string
	AWSSecretKey            string
	ClaimCheckBucket        string
	ClaimCheckThresholdInMB int64

	Environment         string
	RebalanceInterval   int64
	RPCTimeOutInSeconds int64

	SentryDSN                string
	CommitSHA                string
	SentryProfilesSampleRate float64
	SentryTracesSampleRate   float64
}

func NewSweeper(
	config *SweeperConfig,
) (*Sweeper, error) {
	logger = zerolog.Ctx(log.With().Str("component", "generic-indexer-sweeper").Str("chain", config.Chain).Str("environment", config.Environment).Str("commit_sha", config.CommitSHA).Logger().WithContext(context.Background()))

	sentryClientOptions := sentry.ClientOptions{
		Dsn:                config.SentryDSN,
		ServerName:         config.Chain + "-generic-indexer-sweeper",
		EnableTracing:      true,
		ProfilesSampleRate: config.SentryProfilesSampleRate,
		TracesSampleRate:   config.SentryTracesSampleRate,
		Environment:        config.Environment,
		Release:            config.CommitSHA,
		Tags: map[string]string{
			"chain":       config.Chain,
			"environment": config.Environment,
			"component":   "generic-indexer-sweeper",
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
		common.CaptureCurrentHubException(errors.New("RPC: No RPC endpoints provided"), sentry.LevelFatal)
		logger.Fatal().Msgf("RPC: No RPC endpoints provided\n")
		return nil, fmt.Errorf("RPC: No RPC endpoints provided")
	}

	var rpcEndpoints common.RPCEndpoints
	err = json.Unmarshal([]byte(config.RPCEndpoints), &rpcEndpoints)
	if err != nil {
		common.CaptureCurrentHubException(err, sentry.LevelFatal)
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
		common.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("RPC: Error Rebalancing RPC endpoints: %v\n", err)
		return nil, err
	}

	producer, err := mq.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": config.KafkaBootstrapServer,
		"client.id":         config.Chain + "-generic-indexer-sweeper",
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
		common.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("Kafka: Error creating producer. Error: %v\n", err)
		return nil, err
	}

	dbClient, err := db.NewClient(config.DBConnectionString)
	if err != nil {
		common.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("DB: Error creating DB client. Error: %v\n", err)
		return nil, err
	}

	storageClient, err := storage.NewS3Client(config.AWSAccessKey, config.AWSSecretKey)
	if err != nil {
		common.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("DB: Error creating Storage client. Error: %v\n", err)
		return nil, err
	}

	return &Sweeper{
		producer:      producer,
		rpcClient:     rpcClient,
		dbClient:      dbClient,
		storageClient: storageClient,
		config:        config,
	}, nil
}

func (s *Sweeper) parseAndProduceBlock(parentCtx context.Context, resultBlock *coretypes.ResultBlock) error {
	span, ctx := common.StartSentrySpan(parentCtx, "parseAndProduceBlock", "Parsing raw rpc response into Kafka block message")
	defer span.Finish()

	consensusAddress, err := bech32.ConvertAndEncode("initvalcons", resultBlock.Block.ProposerAddress)
	if err != nil {
		logger.Error().Msgf("Failed to convert and encode Bech32: %v\n", err)
		return err
	}

	var proposerAddress *string
	proposerAddress, err = db.GetOperatorAddress(ctx, s.dbClient, consensusAddress)
	if err != nil {
		logger.Warn().Msgf("Failed to get proposer address: %v\n", err)
		proposerAddress = nil
	}

	blockMsgBytes, err := common.NewBlockMsgBytes(resultBlock, proposerAddress, &consensusAddress)
	if err != nil {
		logger.Error().Msgf("Failed to marshal into block message: %v\n", err)
		return err
	}

	s.producer.ProduceWithClaimCheck(&mq.ProduceWithClaimCheckInput{
		Topic:          s.config.KafkaTopic,
		Key:            []byte(fmt.Sprintf("%s_%d", common.NEW_BLOCK_KAFKA_MESSAGE_KEY, resultBlock.Block.Height)),
		MessageInBytes: blockMsgBytes,

		ClaimCheckKey:           []byte(fmt.Sprintf("%s_%d", common.NEW_BLOCK_CLAIM_CHECK_KAFKA_MESSAGE_KEY, resultBlock.Block.Height)),
		ClaimCheckThresholdInMB: s.config.ClaimCheckThresholdInMB,
		ClaimCheckBucket:        s.config.ClaimCheckBucket,
		ClaimCheckObjectPath:    fmt.Sprintf("%d", resultBlock.Block.Height),

		StorageClient: s.storageClient,

		Headers: []kafka.Header{{Key: "height", Value: []byte(fmt.Sprint(resultBlock.Block.Height))}},
	}, logger)

	return nil
}

func (s *Sweeper) getBlockFromRPCs(parentCtx context.Context, height int64) *coretypes.ResultBlock {
	span, ctx := common.StartSentrySpan(parentCtx, "getBlockFromRPCs", "Calling /block from RPCs")
	defer span.Finish()

	var res *coretypes.ResultBlock
	var err error
	for {
		res, err = s.rpcClient.Block(ctx, &height)
		if err == nil {
			return res
		}

		time.Sleep(time.Second * time.Duration(1))
		logger.Warn().Msgf("RPC: Error getting block: %d. Error: %v. Retrying\n", height, err)
	}
}

func (s *Sweeper) GetBlockFromRPCAndProduce(parentCtx context.Context, height int64) {
	hub := sentry.GetHubFromContext(parentCtx)

	transaction, ctx := common.StartSentryTransaction(parentCtx, "Sweep", "Sweep blocks from RPC and produce to Kafka")
	defer transaction.Finish()

	logger.Info().Msgf("RPC: getting data from block %d", height)
	resultBlock := s.getBlockFromRPCs(ctx, height)
	err := s.parseAndProduceBlock(ctx, resultBlock)
	if err != nil {
		common.CaptureException(hub, err, sentry.LevelFatal)
		logger.Fatal().Msgf("Kafka: Error producing message at height: %d. Error: %v\n", height, err)
	}
	logger.Info().Msgf("Successfully sweeped block: %d", height)
}

func (s *Sweeper) close() {
	s.dbClient.Close()

	s.producer.Flush(30000)
	s.producer.Close()
}

func (s *Sweeper) StartSweeping(signalCtx context.Context) {
	s.producer.ListenToKafkaProduceEvents(logger)
	height, err := db.GetLatestBlockHeight(context.Background(), s.dbClient)
	if err != nil {
		logger.Error().Msgf("DB: Error getting latest block height: %v\n", err)
		panic(err)
	}
	workerChannel := make(chan bool, s.config.NumWorkers)

	for {
		select {
		case <-signalCtx.Done():
			// wait for all workers to finish
			for i := 0; i < len(workerChannel); i++ {
				workerChannel <- true
			}
			return
		default:
			height = height + 1
			if s.config.RebalanceInterval != 0 && height%s.config.RebalanceInterval == 0 {
				err := s.rpcClient.Rebalance(context.Background())
				if err != nil {
					common.CaptureCurrentHubException(err, sentry.LevelWarning)
					logger.Error().Msgf("Error rebalancing clients: %v", err)
				}
				actives := s.rpcClient.GetActiveClients()
				for _, active := range actives {
					logger.Info().Msgf("Active client url: %s, latest height: %d", active.Client.GetIdentifier(), active.Height)
				}
			}
			go func(lh int64) {
				localHub := sentry.CurrentHub().Clone()
				localHub.ConfigureScope(func(scope *sentry.Scope) {
					scope.SetTag("height", fmt.Sprint(lh))
				})
				ctx := sentry.SetHubOnContext(context.Background(), localHub)
				s.GetBlockFromRPCAndProduce(ctx, lh)
				<-workerChannel
			}(height)
			workerChannel <- true
		}
	}
}

func (s *Sweeper) Sweep() {
	// graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer sentry.Flush(2 * time.Second)

	s.StartSweeping(ctx)

	logger.Info().Msgf("Shutting down ...")
	s.close()
}

func Max(l, r int64) int64 {
	if l > r {
		return l
	}
	return r
}
