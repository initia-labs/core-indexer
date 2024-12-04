package sweeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/initia-labs/core-indexer/informative-indexer/common"
	"github.com/initia-labs/core-indexer/informative-indexer/cosmosrpc"
	"github.com/initia-labs/core-indexer/informative-indexer/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os/signal"
	"syscall"
	"time"
)

var logger *zerolog.Logger

type Sweeper struct {
	rpcClient cosmosrpc.CosmosJSONRPCHub
	dbClient  *pgxpool.Pool
	config    *SweeperConfig
}

type SweeperConfig struct {
	RPCEndpoints        string
	RPCTimeOutInSeconds int64
	Chain               string
	DBConnectionString  string
	NumWorkers          int64
	RebalanceInterval   int64
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

	return &Sweeper{
		rpcClient: rpcClient,
		dbClient:  dbClient,
		config:    config,
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

	transactionEvents := make([]db.TransactionEvent, 0)

	for i, txResult := range blockResult.TxsResults {
		hash := txHashes[i]
		for _, event := range txResult.Events {
			for _, attr := range event.Attributes {
				transactionEvent := db.TransactionEvent{
					TransactionHash: hash,
					BlockHeight:     blockResult.Height,
					EventKey:        fmt.Sprintf("%s.%s", event.Type, attr.Key),
					EventValue:      attr.Value,
					EventIndex:      i,
				}
				transactionEvents = append(transactionEvents, transactionEvent)
			}
		}
	}

	// TODO: for finalize_block_events
	//for i, event := range blockResult.FinalizeBlockEvents {
	//	for _, attr := range event.Attributes {
	//		transactionEvent := db.TransactionEvent{
	//			BlockHeight: blockResult.Height,
	//			EventKey:    attr.Key,
	//			EventValue:  attr.Value,
	//			EventIndex:  i,
	//		}
	//		transactionEvents = append(transactionEvents, transactionEvent)
	//	}
	//}

	// to see result
	for _, te := range transactionEvents {
		logger.Info().Msgf("tx event: %+v", te)
	}
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
}
