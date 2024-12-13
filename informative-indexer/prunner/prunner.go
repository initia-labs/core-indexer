package prunner

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/initia-labs/core-indexer/informative-indexer/db"
	"github.com/initia-labs/core-indexer/informative-indexer/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os/signal"
	"strings"
	"sync"
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
	DBConnectionString   string
	BackupBucketName     string
	BackupFilePrefix     string
	PruningKeepBlock     int64
	PruningBlockInterval int64
	PruningInterval      int64
	Chain                string
	Environment          string
	CommitSHA            string
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
			//pruningInterval := time.Duration(p.config.PruningInterval) * 24 * time.Hour
			pruningInterval := 10 * time.Second // for local test

			rows, err := db.GetRowCount(context.Background(), p.dbClient)
			if err != nil {
				logger.Error().Msgf("DB: Error getting row count: %v", err)
				panic(err)
			}

			if rows <= p.config.PruningBlockInterval {
				logger.Info().Msg("Pruning not required: # of rows are below the block interval.")
				time.Sleep(pruningInterval)
				continue
			}

			height, err := db.GetLatestBlockHeight(context.Background(), p.dbClient)
			if err != nil {
				logger.Error().Msgf("DB: Error getting latest block height: %v", err)
				panic(err)
			}

			pruningThreshold := height - p.config.PruningKeepBlock
			if pruningThreshold <= 0 {
				logger.Info().Msgf("Pruning not required: threshold is too low: %d", pruningThreshold)
				time.Sleep(pruningInterval)
				continue
			}

			logger.Info().Msgf("Pruning rows with block_height below: %d", pruningThreshold)

			rowsToPrune, err := fetchRowsToPrune(context.Background(), p.dbClient, pruningThreshold)
			if err != nil {
				logger.Error().Msgf("DB: Failed to fetch rows to prune: %v", err)
				time.Sleep(pruningInterval)
				continue
			}

			backupFileName := fmt.Sprintf("%s-%d.zip", p.config.BackupFilePrefix, time.Now().Unix())
			if err := backupToGCS(context.Background(), p.storageClient, p.config.BackupBucketName, "transaction_events", backupFileName, rowsToPrune); err != nil {
				logger.Error().Msgf("GCS: Failed to backup data to GCS: %v", err)
				panic(err)
			}

			if err = prunegRows(context.Background(), p.dbClient, pruningThreshold); err != nil {
				logger.Error().Msgf("DB: Failed to pruning rows: %v", err)
				panic(err)
			}

			logger.Info().Msg("Pruning completed ...")

			time.Sleep(pruningInterval)
		}
	}
}

func fetchRowsToPrune(ctx context.Context, dbClient db.Queryable, pruningThreshold int64) ([]interface{}, error) {
	rows, err := db.GetRowsToPrune(ctx, dbClient, pruningThreshold)
	if err != nil {
		logger.Error().Msgf("DB: Failed to fetch rows to prune: %v", err)
	}

	var result []interface{}
	for rows.Next() {
		var hash string
		var blockHeight int64
		var eventKey string
		var eventValue string
		var eventIndex int64

		err := rows.Scan(&hash, &blockHeight, &eventKey, &eventValue, &eventIndex)
		if err != nil {
			return nil, err
		}

		result = append(result, map[string]interface{}{
			"transaction_hash": hash,
			"block_height":     blockHeight,
			"event_key":        eventKey,
			"event_value":      eventValue,
			"event_index":      eventIndex,
		})
	}

	return result, nil
}

func backupToGCS(ctx context.Context, storageClient storage.Client, bucketName string, folderName string, fileName string, data []interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// create zip
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	w, err := zipWriter.Create(fmt.Sprintf("%s.json", strings.Split(fileName, ".")[0]))
	if err != nil {
		return err
	}
	_, err = w.Write(jsonData)
	if err != nil {
		return err
	}
	zipWriter.Close()

	objectName := fmt.Sprintf("%s/%s", folderName, fileName)

	// upload to GCS
	err = storageClient.UploadFile(bucketName, objectName, buffer.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func prunegRows(ctx context.Context, dbClient db.Queryable, pruningThreshold int64) error {
	err := db.DeleteRowsToPrune(ctx, dbClient, pruningThreshold)
	if err != nil {
		return err
	}
	return nil
}

func (p *Prunner) Prune() {
	// Mutex to avoid multiple prunner instances
	var once sync.Once
	once.Do(func() {
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		go p.StartPruning(ctx)

		<-ctx.Done()

		logger.Info().Msgf("Stopping prunner ...")
		p.close()
	})
}

func (p *Prunner) close() {
	p.dbClient.Close()
}
