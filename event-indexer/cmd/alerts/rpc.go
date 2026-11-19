package alerts_cmd

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/initia-labs/core-indexer/event-indexer/alerts"
	"github.com/spf13/cobra"
)

func RPCCmd() *cobra.Command {
	var c alerts.RPCConfig
	timeout := int64(10)
	if value := os.Getenv("RPC_TIMEOUT_IN_SECONDS"); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			timeout = parsed
		}
	}
	cmd := &cobra.Command{
		Use: "alerts-rpc", Short: "Read current RPC blocks into the transfer alert database", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			err := alerts.RunRPC(ctx, c)
			if ctx.Err() != nil {
				return nil
			}
			return err
		},
	}
	cmd.Flags().StringVar(&c.Chain, "chain", os.Getenv("CHAIN"), "Chain ID")
	cmd.Flags().StringVar(&c.RPCEndpoints, "rpc-endpoints", os.Getenv("RPC_ENDPOINTS"), "RPC endpoints JSON, using the indexer format")
	cmd.Flags().DurationVar(&c.Timeout, "rpc-timeout", time.Duration(timeout)*time.Second, "Timeout for each RPC operation")
	cmd.Flags().Int64Var(&c.StartHeight, "start-height", 0, "First block on initial startup; zero starts at the current RPC head. Existing checkpoint takes precedence")
	return cmd
}
