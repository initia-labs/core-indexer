package sweeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/getsentry/sentry-go"
	"github.com/initia-labs/core-indexer/informative-indexer/common"
	"github.com/initia-labs/core-indexer/informative-indexer/cosmosrpc"
	"github.com/initia-labs/core-indexer/informative-indexer/db"
	"github.com/initia-labs/core-indexer/informative-indexer/mq"
	"github.com/initia-labs/core-indexer/informative-indexer/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os/signal"
	"syscall"
	"time"
)

var logger *zerolog.Logger

type Sweeper struct {
	rpcClient     cosmosrpc.CosmosJSONRPCHub
	dbClient      *pgxpool.Pool
	producer      *mq.Producer
	storageClient storage.StorageClient
	config        *SweeperConfig
}

type SweeperConfig struct {
	RPCEndpoints            string
	RPCTimeOutInSeconds     int64
	Chain                   string
	DBConnectionString      string
	NumWorkers              int64
	RebalanceInterval       int64
	KafkaBootstrapServer    string
	KafkaTopic              string
	KafkaAPIKey             string
	KafkaAPISecret          string
	ClaimCheckBucket        string
	ClaimCheckThresholdInMB int64
	AWSAccessKey            string
	AWSSecretKey            string
}

func NewSweeper(config *SweeperConfig) (*Sweeper, error) {
	logger = zerolog.Ctx(log.With().Str("component", "informative-indexer-sweeper").Str("chain", config.Chain).Logger().WithContext(context.Background()))

	if config.RPCEndpoints == "" {
		common.CaptureCurrentHubException(errors.New("PRC: No RPC endpoints provided"), sentry.LevelFatal)
		logger.Fatal().Msgf("RPC: No RPC endpoints provided\n")
		return nil, fmt.Errorf("RPC: No RPC endpoints provided")
	}

	var rpcEndpoints common.RPCEndpoints
	err := json.Unmarshal([]byte(config.RPCEndpoints), &rpcEndpoints)
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

	dbClient, err := db.NewClient(config.DBConnectionString)
	if err != nil {
		common.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("DB: Error creating DB client. Error: %v\n", err)
		return nil, err
	}

	producer, err := mq.NewProducer(&kafka.ConfigMap{
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
	if err != nil {
		common.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("Kafka: Error creating producer Error: %v\n", err)
		return nil, err
	}

	storageClient, err := storage.NewS3Client(config.AWSAccessKey, config.AWSSecretKey)
	if err != nil {
		common.CaptureCurrentHubException(err, sentry.LevelFatal)
		logger.Fatal().Msgf("DB: Error creating Storage client. Error: %v\n", err)
		return nil, err
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
	height, err := db.GetLatestBlockHeight(context.Background(), s.dbClient)
	if err != nil {
		logger.Error().Msgf("DB: Error getting latest block height: %v\n", err)
		panic(err)
	}
	workerChannel := make(chan bool, s.config.NumWorkers)

	for {
		select {
		case <-signalCtx.Done():
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
				s.GetBlockFromRPCAndProduce(context.Background(), lh)
				<-workerChannel
			}(height)
			workerChannel <- true
		}
	}
}

func (s *Sweeper) GetBlockFromRPCAndProduce(parentCtx context.Context, height int64) {
	logger.Info().Msgf("RPC: Getting data from block: %d", height)

	block, err := s.rpcClient.Block(parentCtx, &height)
	if err != nil {
		logger.Error().Msgf("DB: Error getting block %d: %v\n", height, err)
	}

	blockResult, err := s.rpcClient.BlockResults(parentCtx, &height)
	if err != nil {
		logger.Error().Msgf("DB: Error getting block results %d: %v\n", height, err)
	}

	txHashes := make([]string, len(block.Block.Data.Txs))
	for i, tx := range block.Block.Data.Txs {
		hash := sha256.Sum256(tx)
		txHashes[i] = hex.EncodeToString(hash[:])
	}

	//transactionEvents := make([]db.TransactionEvent, 0)
	//
	//for i, txResult := range blockResult.TxsResults {
	//	hash := txHashes[i]
	//	for _, event := range txResult.Events {
	//		for _, attr := range event.Attributes {
	//			transactionEvent := db.TransactionEvent{
	//				TransactionHash: hash,
	//				BlockHeight:     blockResult.Height,
	//				EventKey:        fmt.Sprintf("%s.%s", event.Type, attr.Key),
	//				EventValue:      attr.Value,
	//				EventIndex:      i,
	//			}
	//			transactionEvents = append(transactionEvents, transactionEvent)
	//		}
	//	}
	//}
	//
	//finalizeBlockEvents := make([]db.FinalizeBlockEvent, 0)
	//
	//for i, event := range blockResult.FinalizeBlockEvents {
	//	for _, attr := range event.Attributes {
	//		finalizeBlockEvent := db.FinalizeBlockEvent{
	//			BlockHeight: blockResult.Height,
	//			EventKey:    fmt.Sprintf("%s.%s", event.Type, attr.Key),
	//			EventValue:  attr.Value,
	//			EventIndex:  i,
	//		}
	//
	//		if attr.Key == "mode" {
	//			mode, err := db.ParseMode(attr.Value)
	//			if err != nil {
	//				logger.Error().Msgf("DB: Error parsing mode: %v\n", err)
	//			}
	//			finalizeBlockEvent.Mode = mode
	//
	//		}
	//		finalizeBlockEvents = append(finalizeBlockEvents, finalizeBlockEvent)
	//	}
	//}
	//
	//for _, te := range transactionEvents {
	//	logger.Info().Msgf("tx event: %+v", te)
	//}
	//
	//for _, fe := range finalizeBlockEvents {
	//	logger.Info().Msgf("finalized block event: %+v", fe)
	//}

	err = s.MakeAndSendBlockResultMsg(parentCtx, txHashes, blockResult)
	if err != nil {
		logger.Fatal().Msgf("Kafka: Error producing message at height: %d. Error: %v\n", height, err)
	}
}

func (s *Sweeper) MakeAndSendBlockResultMsg(ctx context.Context, txHashes []string, blockResult *coretypes.ResultBlockResults) error {
	txResults := make([]common.TxResult, len(blockResult.TxsResults))
	for i, txResult := range blockResult.TxsResults {
		txResults[i] = common.TxResult{
			Hash:          txHashes[i],
			ExecTxResults: txResult,
		}
	}

	blockResultMsg := common.BlockResultMsg{
		Height:              blockResult.Height,
		Txs:                 txResults,
		FinalizeBlockEvents: blockResult.FinalizeBlockEvents,
	}

	blockResultMsgBytes, err := common.NewBlockResultMsgBytes(blockResultMsg)
	if err != nil {
		logger.Error().Msgf("Failed to marshal into block result message: %v\n", err)
		return err
	}

	s.producer.ProduceWithClaimCheck(&mq.ProduceWithClaimCheckInput{
		Topic:                   s.config.KafkaTopic,
		Key:                     []byte(fmt.Sprintf("%s_%d", common.NEW_BLOCK_RESULT_KAFKA_MESSAGE_KEY, blockResult.Height)),
		MessageInBytes:          blockResultMsgBytes,
		ClaimCheckKey:           []byte(fmt.Sprintf("%s_%d", common.NEW_BLOCK_RESULT_CLAIM_CHECK_KAFKA_MESSAGE_KEY, blockResult.Height)),
		ClaimCheckThresholdInMB: s.config.ClaimCheckThresholdInMB,
		ClaimCheckBucket:        s.config.ClaimCheckBucket,
		ClaimCheckObjectPath:    fmt.Sprintf("%d", blockResult.Height),
		StorageClient:           s.storageClient,
		Headers:                 []kafka.Header{{Key: "height", Value: []byte(fmt.Sprint(blockResult.Height))}},
	}, logger)

	return nil
}

func (s *Sweeper) Sweep() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer sentry.Flush(2 * time.Second)

	s.StartSweeping(ctx)

	logger.Info().Msgf("Stopping sweeper ...")
	s.close()
}

func (s *Sweeper) close() {
	// TODO: Wrapping up
	s.dbClient.Close()
	s.producer.Flush(30000)
	s.producer.Close()
}
