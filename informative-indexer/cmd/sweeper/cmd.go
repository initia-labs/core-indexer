package sweeper_cmd

import (
	"github.com/initia-labs/core-indexer/informative-indexer/sweeper"
	"github.com/spf13/cobra"
	"os"
	"runtime"
	"strconv"
)

const (
	FlagRPCEndpoints        = "rpcs"
	FlagChain               = "chain"
	FlagRPCTimeoutInSeconds = "rpc-timeout-in-seconds"
	FlagDBConnectionString  = "db"
	FlagNumWorkers          = "workers"
	FlagRebalanceInterval   = "rebalance-interval"

	FlagKafkaBootstrapServer = "bootstrap-server"
	FlagKafkaTopic           = "block-topic"
	FlagKafkaAPIKey          = "kafka-api-key"
	FlagKafkaAPISecret       = "kafka-api-secret"
)

func SweepCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sweep",
		Short: "Sweep the blockchain for data",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			rpcEndpoints, _ := cmd.Flags().GetString(FlagRPCEndpoints)
			rpcTimeOutInSeconds, _ := cmd.Flags().GetInt64(FlagRPCTimeoutInSeconds)
			chain, _ := cmd.Flags().GetString(FlagChain)
			dbConnectionString, _ := cmd.Flags().GetString(FlagDBConnectionString)
			numWorkers, _ := cmd.Flags().GetInt(FlagNumWorkers)
			rebalanceInterval, _ := cmd.Flags().GetInt64(FlagRebalanceInterval)
			kafkaBootstrapServer, _ := cmd.Flags().GetString(FlagKafkaBootstrapServer)
			kafkaTopic, _ := cmd.Flags().GetString(FlagKafkaTopic)
			kafkaAPIKey, _ := cmd.Flags().GetString(FlagKafkaAPIKey)
			kafkaAPISecret, _ := cmd.Flags().GetString(FlagKafkaAPISecret)

			s, err := sweeper.NewSweeper(&sweeper.SweeperConfig{
				RPCEndpoints:         rpcEndpoints,
				RPCTimeOutInSeconds:  rpcTimeOutInSeconds,
				Chain:                chain,
				DBConnectionString:   dbConnectionString,
				NumWorkers:           int64(numWorkers),
				RebalanceInterval:    rebalanceInterval,
				KafkaBootstrapServer: kafkaBootstrapServer,
				KafkaTopic:           kafkaTopic,
				KafkaAPIKey:          kafkaAPIKey,
				KafkaAPISecret:       kafkaAPISecret,
			})

			if err != nil {
				return err
			}

			s.Sweep()

			return nil
		},
	}

	rpcTimeOutInSeconds, err := strconv.ParseInt(os.Getenv("RPC_TIMEOUT_IN_SECONDS"), 10, 64)
	if err != nil {
		rpcTimeOutInSeconds = 30
	}

	rebalanceInterval, err := strconv.ParseInt(os.Getenv("REBALANCE_INTERVAL"), 10, 64)
	if err != nil {
		rebalanceInterval = 0
	}

	cmd.Flags().String(FlagRPCEndpoints, os.Getenv("RPC_ENDPOINTS"), "")
	cmd.Flags().Int64(FlagRPCTimeoutInSeconds, rpcTimeOutInSeconds, "RPC timeout in seconds")
	cmd.Flags().String(FlagChain, os.Getenv("CHAIN"), "Chain ID to sweep")
	cmd.Flags().String(FlagDBConnectionString, os.Getenv("DB_CONNECTION_STRING"), "Database connection string")
	cmd.Flags().Uint64(FlagNumWorkers, uint64(runtime.NumCPU()), "Worker count")
	cmd.Flags().Int64(FlagRebalanceInterval, rebalanceInterval, "RPC providers rebalance interval")
	cmd.Flags().String(FlagKafkaBootstrapServer, os.Getenv("BOOTSTRAP_SERVER"), "<host>:<port> to Kafka bootstrap server")
	cmd.Flags().String(FlagKafkaTopic, os.Getenv("BLOCK_TOPIC"), "Kafka topic")
	cmd.Flags().String(FlagKafkaAPIKey, os.Getenv("KAFKA_API_KEY"), "Kafka API key")
	cmd.Flags().String(FlagKafkaAPISecret, os.Getenv("KAFKA_API_SECRET"), "Kafka API secret")

	return cmd
}
