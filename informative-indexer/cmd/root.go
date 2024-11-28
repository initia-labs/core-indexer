package cmd

import (
	sweeper "github.com/initia-labs/core-indexer/informative-indexer/cmd/sweeper"
	"github.com/spf13/cobra"
	"os"
)

func Execute() {
	var rootCmd = &cobra.Command{
		Use:   "informative-indexer",
		Short: "Informative Indexer Runner",
		Long:  "Informative Indexer Runner - Polls data from RPC and flushes into database",
	}

	rootCmd.AddCommand(sweeper.SweepCmd())

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
