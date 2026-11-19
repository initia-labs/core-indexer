package alerts_cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/initia-labs/core-indexer/event-indexer/alerts"
	"github.com/spf13/cobra"
)

func RunCmd() *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use: "alerts", Short: "Query indexed event groups from PostgreSQL and send transfer alerts", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := alerts.LoadConfig(path)
			if err != nil {
				return err
			}
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			err = alerts.Run(ctx, c)
			if ctx.Err() != nil {
				return nil
			}
			return err
		},
	}
	cmd.Flags().StringVar(&path, "config", "alerts.json", "Transfer alert configuration")
	return cmd
}
