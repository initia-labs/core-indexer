package sweeper_cmd

import (
	"os"
	"runtime"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/initia-labs/core-indexer/informative-indexer/sweeper"
)

const (
	FlagRPCEndpoints             = "rpcs"
	FlagChain                    = "chain"
	FlagRPCTimeoutInSeconds      = "rpc-timeout-in-seconds"
	FlagDBConnectionString       = "db"
	FlagNumWorkers               = "workers"
	FlagRebalanceInterval        = "rebalance-interval"
	FlagKafkaBootstrapServer     = "bootstrap-server"
	FlagKafkaTopic               = "block-results-topic"
	FlagKafkaAPIKey              = "kafka-api-key"
	FlagKafkaAPISecret           = "kafka-api-secret"
	FlagClaimCheckBucket         = "claim-check-bucket"
	FlagClaimCheckThresholdInMB  = "claim-check-threshold-mb"
	FlagEnvironment              = "environment"
	FlagSentryDSN                = "sentry-dsn"
	FlagCommitSHA                = "commit-sha"
	FlagSentryProfilesSampleRate = "sentry-profiles-sample-rate"
	FlagSentryTracesSampleRate   = "sentry-traces-sample-rate"
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
			claimCheckBucket, _ := cmd.Flags().GetString(FlagClaimCheckBucket)
			claimCheckThresholdInMB, _ := cmd.Flags().GetUint64(FlagClaimCheckThresholdInMB)
			environment, _ := cmd.Flags().GetString(FlagEnvironment)
			sentryDSN, _ := cmd.Flags().GetString(FlagSentryDSN)
			commitSHA, _ := cmd.Flags().GetString(FlagCommitSHA)
			sentryProfilesSampleRate, _ := cmd.Flags().GetFloat64(FlagSentryProfilesSampleRate)
			sentryTracesSampleRate, _ := cmd.Flags().GetFloat64(FlagSentryTracesSampleRate)

			s, err := sweeper.NewSweeper(&sweeper.SweeperConfig{
				RPCEndpoints:             rpcEndpoints,
				RPCTimeOutInSeconds:      rpcTimeOutInSeconds,
				Chain:                    chain,
				DBConnectionString:       dbConnectionString,
				NumWorkers:               int64(numWorkers),
				RebalanceInterval:        rebalanceInterval,
				KafkaBootstrapServer:     kafkaBootstrapServer,
				KafkaTopic:               kafkaTopic,
				KafkaAPIKey:              kafkaAPIKey,
				KafkaAPISecret:           kafkaAPISecret,
				ClaimCheckBucket:         claimCheckBucket,
				ClaimCheckThresholdInMB:  int64(claimCheckThresholdInMB),
				Environment:              environment,
				SentryDSN:                sentryDSN,
				CommitSHA:                commitSHA,
				SentryProfilesSampleRate: sentryProfilesSampleRate,
				SentryTracesSampleRate:   sentryTracesSampleRate,
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

	threshold, err := strconv.ParseInt(os.Getenv("CLAIM_CHECK_THRESHOLD_IN_MB"), 10, 64)
	if err != nil {
		threshold = 1
	}

	sentryProfilesSampleRate, err := strconv.ParseFloat(os.Getenv("SENTRY_PROFILES_SAMPLE_RATE"), 64)
	if err != nil {
		sentryProfilesSampleRate = 0.01
	}

	sentryTracesSampleRate, err := strconv.ParseFloat(os.Getenv("SENTRY_TRACES_SAMPLE_RATE"), 64)
	if err != nil {
		sentryTracesSampleRate = 0.01
	}

	cmd.Flags().String(FlagRPCEndpoints, os.Getenv("RPC_ENDPOINTS"), "")
	cmd.Flags().Int64(FlagRPCTimeoutInSeconds, rpcTimeOutInSeconds, "RPC timeout in seconds")
	cmd.Flags().String(FlagChain, os.Getenv("CHAIN"), "Chain ID to sweep")
	cmd.Flags().String(FlagDBConnectionString, os.Getenv("DB_CONNECTION_STRING"), "Database connection string")
	cmd.Flags().Uint64(FlagNumWorkers, uint64(runtime.NumCPU()), "Worker count")
	cmd.Flags().Int64(FlagRebalanceInterval, rebalanceInterval, "RPC providers rebalance interval")
	cmd.Flags().String(FlagKafkaBootstrapServer, os.Getenv("BOOTSTRAP_SERVER"), "<host>:<port> to Kafka bootstrap server")
	cmd.Flags().String(FlagKafkaTopic, os.Getenv("BLOCK_RESULTS_TOPIC"), "Kafka topic")
	cmd.Flags().String(FlagKafkaAPIKey, os.Getenv("KAFKA_API_KEY"), "Kafka API key")
	cmd.Flags().String(FlagKafkaAPISecret, os.Getenv("KAFKA_API_SECRET"), "Kafka API secret")
	cmd.Flags().String(FlagClaimCheckBucket, os.Getenv("CLAIM_CHECK_BUCKET"), "Claim check bucket")
	cmd.Flags().Uint64(FlagClaimCheckThresholdInMB, uint64(threshold), "Claim check threshold in MB")
	cmd.Flags().String(FlagEnvironment, os.Getenv("ENVIRONMENT"), "Environment")
	cmd.Flags().String(FlagSentryDSN, os.Getenv("SENTRY_DSN"), "Sentry DSN")
	cmd.Flags().String(FlagCommitSHA, os.Getenv("COMMIT_SHA"), "Commit SHA")
	cmd.Flags().Float64(FlagSentryProfilesSampleRate, sentryProfilesSampleRate, "Sentry profiles sample rate")
	cmd.Flags().Float64(FlagSentryTracesSampleRate, sentryTracesSampleRate, "Sentry traces sample rate")

	return cmd
}
