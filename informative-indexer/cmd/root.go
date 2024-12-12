package cmd

import (
	"os"

	"github.com/spf13/cobra"

	flusher "github.com/initia-labs/core-indexer/informative-indexer/cmd/flusher"
	prunner "github.com/initia-labs/core-indexer/informative-indexer/cmd/prunner"
	sweeper "github.com/initia-labs/core-indexer/informative-indexer/cmd/sweeper"
)

func Execute() {
	var rootCmd = &cobra.Command{
		Use:   "informative-indexer",
		Short: "Informative Indexer Runner",
		Long:  "Informative Indexer Runner - Polls data from RPC and flushes into database",
	}

	rootCmd.AddCommand(
		sweeper.SweepCmd(),
		flusher.FlushCmd(),
		prunner.PruneCmd(),
	)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
