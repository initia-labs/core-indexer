package prunner

import (
	"context"
	"github.com/initia-labs/core-indexer/informative-indexer/db"
	"github.com/initia-labs/core-indexer/informative-indexer/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os/signal"
	"syscall"
	"time"
)

var logger *zerolog.Logger

type Prunner struct {
	dbClient      *pgxpool.Pool
	storageClient storage.Client
	config        *PrunnerConfig
}

type PrunnerConfig struct {
	DBConnectionString string
	BackupBucketName   string
	BackupFilePrefix   string
	PruningBlockHeight int64
	//PruningAgeDays     int64
	PruningInterval int64
	Chain           string
	Environment     string
	CommitSHA       string
}

func NewPrunner(config *PrunnerConfig) (*Prunner, error) {
	logger = zerolog.Ctx(log.With().Str("component", "informative-indexer-prunner").Str("chain", config.Chain).Str("environment", config.Environment).Str("commit_sha", config.CommitSHA).Logger().WithContext(context.Background()))

	dbClient, err := db.NewClient(config.DBConnectionString)
	if err != nil {
		logger.Fatal().Msgf("DB: Error creating DB client: %v", err)
		return nil, err
	}

	storageClient, err := storage.NewGCSClient()
	if err != nil {
		logger.Fatal().Msgf("Storage: Error creating storage client: %v", err)
		return nil, err
	}

	return &Prunner{
		dbClient:      dbClient,
		storageClient: storageClient,
		config:        config,
	}, nil

}

func (p *Prunner) StartPruning(signalCtx context.Context) {
	logger.Info().Msg("Prunner: Starting pruning process ...")

	for {
		select {
		case <-signalCtx.Done():
			logger.Info().Msg("Prunner: Received stop signla. Exiting pruning loop ...")
			return
		default:
			height, err := db.GetLatestBlockHeight(context.Background(), p.dbClient)
			if err != nil {
				logger.Error().Msgf("DB: Error getting latest block height: %v", err)
				panic(err)
			}
			logger.Info().Msgf("Current block height: %d", height)

			// interval
			time.Sleep(5 * time.Second)
		}
	}
}

func (p *Prunner) Prune() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go p.StartPruning(ctx)

	<-ctx.Done()

	logger.Info().Msgf("Stopping prunner ...")
	p.close()
}

func (p *Prunner) close() {
	p.dbClient.Close()
}
