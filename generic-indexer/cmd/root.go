package cmd

import (
	"os"

	"github.com/spf13/cobra"

	flusher "github.com/alleslabs/initia-mono/generic-indexer/cmd/flusher"
	sweeper "github.com/alleslabs/initia-mono/generic-indexer/cmd/sweeper"
	validatorcron "github.com/alleslabs/initia-mono/generic-indexer/cmd/validatorcron"
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
		sweeper.SweepCmd(),
		flusher.FlushCmd(),
		validatorcron.FlushCmd(),
	)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
