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
	DBConnectionString string
	BackupBucketName   string
	BackupFilePrefix   string
	PruningKeepBlock   int64
	PruningInterval    int64
	Chain              string
	Environment        string
	CommitSHA          string
}

func NewPrunner(config *PrunnerConfig) (*Prunner, error) {
	logger = zerolog.Ctx(log.With().Str("component", "informative-indexer-prunner").Str("chain", config.Chain).Str("environment", config.Environment).Str("commit_sha", config.CommitSHA).Logger().WithContext(context.Background()))

	dbClient, err := db.NewClient(config.DBConnectionString)
	if err != nil {
		logger.Fatal().Msgf("DB: Error creating DB client: %v", err)
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
			logger.Fatal().Msgf("Storage: Error creating storage client: %v\n", err)
			return nil, err
		}
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
			logger.Info().Msg("Prunner: Received stop signal. Exiting pruning loop ...")
			return
		default:
			pruningInterval := time.Duration(p.config.PruningInterval) * 24 * time.Hour
			//pruningInterval := 10 * time.Second // for local test

			for _, table := range db.GetValidTableNames() {
				if err := p.pruningTable(signalCtx, table); err != nil {
					logger.Error().Msgf("Error during pruning for table %s: %v", table, err)
				}
			}

			time.Sleep(pruningInterval)
		}
	}
}

func (p *Prunner) pruningTable(ctx context.Context, tableName string) error {
	height, err := db.GetLatestBlockHeight(ctx, p.dbClient)
	if err != nil {
		return fmt.Errorf("DB: failed to get latest block height: %w", err)
	}

	pruningThreshold := height - p.config.PruningKeepBlock
	if pruningThreshold <= 0 {
		logger.Info().Msgf("Pruning not required for table %s: threshold is too low: %d", tableName, pruningThreshold)
		return nil
	}

	logger.Info().Msgf("Pruning rows from table %s with block_height below: %d", tableName, pruningThreshold)

	rowsToPrune, err := fetchRowsToPrune(ctx, p.dbClient, tableName, pruningThreshold)
	if err != nil {
		return fmt.Errorf("DB: Failed to fetch rows to prune from table %s: %w", tableName, err)
	}
	// no rows to prune
	if len(rowsToPrune) == 0 {
		logger.Info().Msgf("Pruning not required for table %s: no rows to prune", tableName)
		return nil
	}

	backupFileName := fmt.Sprintf("%s-%d.zip", p.config.BackupFilePrefix, time.Now().Unix())
	if err := backupToGCS(ctx, p.storageClient, p.config.BackupBucketName, tableName, backupFileName, rowsToPrune); err != nil {
		return fmt.Errorf("GCS: Failed to backup data from table %s to GCS: %w", tableName, err)
	}

	if err = pruneRows(ctx, p.dbClient, tableName, pruningThreshold); err != nil {
		return fmt.Errorf("DB: Failed to prune rows from table %s: %w", tableName, err)
	}

	logger.Info().Msg("Pruning completed ...")
	return nil
}

func fetchRowsToPrune(ctx context.Context, dbClient db.Queryable, tableName string, pruningThreshold int64) ([]interface{}, error) {
	rows, err := db.GetRowsToPruneByBlockHeight(ctx, dbClient, tableName, pruningThreshold)
	if err != nil {
		return nil, fmt.Errorf("DB: Failed to fetch rows to prune from table %s: %w", tableName, err)
	}

	var result []interface{}
	for rows.Next() {
		table, ok := db.ValidTablesMap[tableName]
		if !ok {
			return nil, fmt.Errorf("failed to get table interface from table name %s", tableName)
		}
		row, err := table.Unmarshal(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, nil
}

func backupToGCS(ctx context.Context, storageClient storage.Client, bucketName string, tableName string, fileName string, data []interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// create zip
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	w, err := zipWriter.Create(fmt.Sprintf("%s.json", strings.Split(fileName, ".")[0]))
	if err != nil {
		return fmt.Errorf("failed to create zip entry: %w", err)
	}
	_, err = w.Write(jsonData)
	if err != nil {
		return fmt.Errorf("failed to write data to zip: %w", err)
	}
	if err = zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close zip writer: %w", err)
	}

	objectName := fmt.Sprintf("%s/%s", tableName, fileName)

	// upload to GCS
	err = storageClient.UploadFile(bucketName, objectName, buffer.Bytes())
	if err != nil {
		return fmt.Errorf("failed to upload file to GCS: %w", err)
	}

	return nil
}

func pruneRows(ctx context.Context, dbClient db.Queryable, tableName string, pruningThreshold int64) error {
	err := db.DeleteRowsToPrune(ctx, dbClient, tableName, pruningThreshold)
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
