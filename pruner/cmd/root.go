package cmd

import (
	"os"

	"github.com/spf13/cobra"

	pruner "github.com/initia-labs/core-indexer/pruner/cmd/pruner"
)

func Execute() {
	var rootCmd = &cobra.Command{
		Use:   "pruner",
		Short: "Pruner Runner",
		Long:  "Pruner Runner - Pruning and Backup data",
	}

	rootCmd.AddCommand(
		pruner.RunCmd(),
	)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
