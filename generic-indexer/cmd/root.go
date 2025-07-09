package cmd

import (
	"os"

	"github.com/spf13/cobra"

	flusher "github.com/initia-labs/core-indexer/generic-indexer/cmd/flusher"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	var rootCmd = &cobra.Command{
		Use:   "generic-indexer",
		Short: "Generic Indexer Runner",
		Long:  `Generic Indexer Runner - Polls block data from Tendermint RPC and flushes into database`,
	}

	rootCmd.AddCommand(
		flusher.FlushCmd(),
		// validatorcron.FlushCmd(),
	)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
