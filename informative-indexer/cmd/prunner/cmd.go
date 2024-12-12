package prunner_cmd

import (
	"github.com/initia-labs/core-indexer/informative-indexer/prunner"
	"github.com/spf13/cobra"
	"os"
	"strconv"
)

const (
	FlagDBConnectionString = "db"
	FlagBackupBucketName   = "backup-bucket-name"
	FlagBackupFilePrefix   = "backup-file-prefix"
	FlagPruningBlockHeight = "pruning-block-height"
	//FlagPruningAgeDays     = "pruning-age-days"
	FlagPruningInterval = "pruning-interval"
	FlagChain           = "chain"
	FlagEnvironment     = "environment"
	FlagCommitSHA       = "commit-sha"
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
			pruningBlockHeight, _ := cmd.Flags().GetUint64(FlagPruningBlockHeight)
			//pruningAgeDays, _ := cmd.Flags().GetUint64(FlagPruningAgeDays)
			pruningInterval, _ := cmd.Flags().GetUint64(FlagPruningInterval)
			chain, _ := cmd.Flags().GetString(FlagChain)
			environment, _ := cmd.Flags().GetString(FlagEnvironment)
			commitSHA, _ := cmd.Flags().GetString(FlagCommitSHA)

			//if pruningBlockHeight > 0 && pruningAgeDays > 0 {
			//	fmt.Fprintln(os.Stderr, "Error: --pruning-block-height and --pruning-age-days cannot both be specified")
			//	os.Exit(1)
			//}

			p, err := prunner.NewPrunner(&prunner.PrunnerConfig{
				DBConnectionString: dbConnectionString,
				BackupBucketName:   backupBucketName,
				BackupFilePrefix:   filePrefix,
				PruningBlockHeight: int64(pruningBlockHeight),
				//PruningAgeDays:     int64(pruningAgeDays),
				PruningInterval: int64(pruningInterval),
				Chain:           chain,
				Environment:     environment,
				CommitSHA:       commitSHA,
			})

			if err != nil {
				return err
			}

			p.Prune()

			return nil
		},
	}

	//pruningAgeDays, err := strconv.ParseInt(os.Getenv("PRUNING_AGE_DAYS"), 10, 64)
	//if err != nil {
	//	pruningAgeDays = 0
	//}

	pruningBlockheight, err := strconv.ParseInt(os.Getenv("PRUNING_BLOCK_HEIGHT"), 10, 64)
	if err != nil {
		pruningBlockheight = 500000
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
	cmd.Flags().Uint64(FlagPruningBlockHeight, uint64(pruningBlockheight), "Pruning by block height")
	//cmd.Flags().Uint64(FlagPruningAgeDays, uint64(pruningAgeDays), "Pruning by days")
	cmd.Flags().Uint64(FlagPruningInterval, uint64(pruningInterval), "Pruning interval, days")
	cmd.Flags().String(FlagChain, os.Getenv("CHAIN"), "Chain ID to prune")
	cmd.Flags().String(FlagEnvironment, os.Getenv("ENVIRONMENT"), "Environment")
	cmd.Flags().String(FlagCommitSHA, os.Getenv("COMMIT_SHA"), "Commit SHA")

	return cmd
}
