package cmd

import (
	"os"

	"github.com/spf13/cobra"

	indexer "github.com/initia-labs/core-indexer/informative-indexer/cmd/indexer"
	migrate "github.com/initia-labs/core-indexer/informative-indexer/cmd/migrate"
)

func Execute() {
	var rootCmd = &cobra.Command{
		Use:   "informative-indexer",
		Short: "Informative Indexer Runner",
		Long:  "Informative Indexer Runner - Polls data from RPC and indexes into database",
	}

	rootCmd.AddCommand(
		migrate.MigrateCmd(),
		indexer.RunCmd(),
	)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
