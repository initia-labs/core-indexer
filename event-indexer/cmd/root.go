package cmd

import (
	"os"

	"github.com/spf13/cobra"

	indexer "github.com/initia-labs/core-indexer/event-indexer/cmd/indexer"
	migrate "github.com/initia-labs/core-indexer/event-indexer/cmd/migrate"
	prunner "github.com/initia-labs/core-indexer/event-indexer/cmd/prunner"
)

func Execute() {
	var rootCmd = &cobra.Command{
		Use:   "event-indexer",
		Short: "Event Indexer Runner",
		Long:  "Event Indexer Runner - Polls data from RPC and indexes into database",
	}

	rootCmd.AddCommand(
		migrate.MigrateCmd(),
		indexer.RunCmd(),
		prunner.PruneCmd(),
	)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
