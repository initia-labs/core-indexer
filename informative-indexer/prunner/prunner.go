package prunner

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/storage"
)

var logger *zerolog.Logger

type Prunner struct {
	dbClient      *gorm.DB
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

	query, err := db.BuildPruneQuery(ctx, p.dbClient, tableName, pruningThreshold)
	if err != nil {
		return fmt.Errorf("DB: Failed to prepare query for table %s: %w", tableName, err)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return fmt.Errorf("DB: Failed to count rows for table %s: %w", tableName, err)
	}

	// no rows to prune
	if count == 0 {
		logger.Info().Msgf("Pruning not required for table %s: no rows to prune", tableName)
		return nil
	}

	backupFileName := fmt.Sprintf("%s-%d.zip", p.config.BackupFilePrefix, time.Now().Unix())
	if err := streamBackupToGCS(ctx, p.dbClient, query, p.storageClient, p.config.BackupBucketName,
		tableName, backupFileName); err != nil {
		return fmt.Errorf("GCS: Failed to backup data from table %s to GCS: %w", tableName, err)
	}

	if err = pruneRows(ctx, p.dbClient, tableName, pruningThreshold); err != nil {
		return fmt.Errorf("DB: Failed to prune rows from table %s: %w", tableName, err)
	}

	logger.Info().Msg("Pruning completed ...")
	return nil
}

// streamBackupToGCS streams query results directly to a zip file and uploads to GCS
func streamBackupToGCS(ctx context.Context, dbClient *gorm.DB, query *gorm.DB, storageClient storage.Client, bucketName, tableName, fileName string) error {
	// create zip
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)

	// create a file inside the zip
	zipFile, err := zipWriter.Create(fmt.Sprintf("%s.json", strings.Split(fileName, ".")[0]))
	if err != nil {
		return fmt.Errorf("failed to create zip entry: %w", err)
	}

	// start writing the JSON array opening bracket
	if _, err := zipFile.Write([]byte("[")); err != nil {
		return fmt.Errorf("failed to write opening bracket: %w", err)
	}

	// process data in chunks
	const chunkSize = 1000
	offset := 0
	isFirstRow := true

	for {
		// check if context is canceled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		var chunk []map[string]any
		if err := query.Limit(chunkSize).Offset(offset).Find(&chunk).Error; err != nil {
			return fmt.Errorf("failed to fetch chunk: %w", err)
		}

		// no more data, break the loop
		if len(chunk) == 0 {
			break
		}

		for _, rowData := range chunk {
			if !isFirstRow {
				if _, err := zipFile.Write([]byte(",")); err != nil {
					return fmt.Errorf("failed to write comma: %w", err)
				}
			} else {
				isFirstRow = false
			}

			rowJSON, err := json.Marshal(rowData)
			if err != nil {
				return fmt.Errorf("failed to marshal row: %w", err)
			}

			if _, err := zipFile.Write(rowJSON); err != nil {
				return fmt.Errorf("failed to write row JSON: %w", err)
			}
		}

		offset += len(chunk)

		if len(chunk) < chunkSize {
			break
		}
	}

	if _, err := zipFile.Write([]byte("]")); err != nil {
		return fmt.Errorf("failed to write closing bracket: %w", err)
	}

	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close zip writer: %w", err)
	}

	objectName := fmt.Sprintf("%s/%s", tableName, fileName)
	err = storageClient.UploadFile(bucketName, objectName, buffer.Bytes())
	if err != nil {
		return fmt.Errorf("failed to upload file to GCS: %w", err)
	}

	return nil
}

func pruneRows(ctx context.Context, dbClient *gorm.DB, tableName string, pruningThreshold int64) error {
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
	sqlDB, err := p.dbClient.DB()
	if err == nil {
		sqlDB.Close()
	}
}
