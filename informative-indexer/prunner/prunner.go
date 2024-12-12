package prunner

import (
	"context"
	"github.com/initia-labs/core-indexer/informative-indexer/db"
	"github.com/initia-labs/core-indexer/informative-indexer/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

func (p *Prunner) Prune() {
	logger.Log().Msg("PRUNE ENTRY")

	logger.Log().Msg("PRUNE EXIT")
}
