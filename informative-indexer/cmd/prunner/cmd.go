package prunner_cmd

import (
	"github.com/initia-labs/core-indexer/informative-indexer/prunner"
	"github.com/spf13/cobra"
	"os"
	"strconv"
)

const (
	FlagDBConnectionString   = "db"
	FlagBackupBucketName     = "backup-bucket-name"
	FlagBackupFilePrefix     = "backup-file-prefix"
	FlagPruningKeepBlock     = "pruning-keep-block"
	FlagPruningBlockInterval = "pruning-block-interval"
	FlagPruningInterval      = "pruning-interval"
	FlagChain                = "chain"
	FlagEnvironment          = "environment"
	FlagCommitSHA            = "commit-sha"
)

func PruneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Pruning and backup data",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbConnectionString, _ := cmd.Flags().GetString(FlagDBConnectionString)
			backupBucketName, _ := cmd.Flags().GetString(FlagBackupBucketName)
			filePrefix, _ := cmd.Flags().GetString(FlagBackupFilePrefix)
			pruningKeepBlock, _ := cmd.Flags().GetUint64(FlagPruningKeepBlock)
			pruningBlockInterval, _ := cmd.Flags().GetUint64(FlagPruningBlockInterval)
			pruningInterval, _ := cmd.Flags().GetUint64(FlagPruningInterval)
			chain, _ := cmd.Flags().GetString(FlagChain)
			environment, _ := cmd.Flags().GetString(FlagEnvironment)
			commitSHA, _ := cmd.Flags().GetString(FlagCommitSHA)

			p, err := prunner.NewPrunner(&prunner.PrunnerConfig{
				DBConnectionString:   dbConnectionString,
				BackupBucketName:     backupBucketName,
				BackupFilePrefix:     filePrefix,
				PruningKeepBlock:     int64(pruningKeepBlock),
				PruningBlockInterval: int64(pruningBlockInterval),
				PruningInterval:      int64(pruningInterval),
				Chain:                chain,
				Environment:          environment,
				CommitSHA:            commitSHA,
			})

			if err != nil {
				return err
			}

			p.Prune()

			return nil
		},
	}

	pruningKeepBlock, err := strconv.ParseInt(os.Getenv("PRUNING_KEEP_BLOCK"), 10, 64)
	if err != nil {
		pruningKeepBlock = 500000
	}

	pruningBlockInterval, err := strconv.ParseInt(os.Getenv("PRUNING_BLOCK_INTERVAL"), 10, 64)
	if err != nil {
		pruningBlockInterval = 100000
	}

	pruningInterval, err := strconv.ParseInt(os.Getenv("PRUNING_INTERVAL"), 10, 64)
	{
		if err != nil {
			pruningInterval = 1
		}
	}

	cmd.Flags().String(FlagDBConnectionString, os.Getenv("DB_CONNECTION_STRING"), "Database connection string")
	cmd.Flags().String(FlagBackupBucketName, os.Getenv("BACKUP_BUCKET_NAME"), "Backup bucket name")
	cmd.Flags().String(FlagBackupFilePrefix, os.Getenv("BACKUP_FILE_PREFIX"), "Backup file prefix")
	cmd.Flags().Uint64(FlagPruningKeepBlock, uint64(pruningKeepBlock), "The number of blocks are kept")
	cmd.Flags().Uint64(FlagPruningBlockInterval, uint64(pruningBlockInterval), "n, Pruning at n block interval")
	cmd.Flags().Uint64(FlagPruningInterval, uint64(pruningInterval), "Pruning time interval, Days")
	cmd.Flags().String(FlagChain, os.Getenv("CHAIN"), "Chain ID to prune")
	cmd.Flags().String(FlagEnvironment, os.Getenv("ENVIRONMENT"), "Environment")
	cmd.Flags().String(FlagCommitSHA, os.Getenv("COMMIT_SHA"), "Commit SHA")

	return cmd
}
