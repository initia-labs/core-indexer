package sweeper_cmd

import (
	"os"
	"runtime"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/alleslabs/initia-mono/generic-indexer/sweeper"
)

// List of CLI flags
const (
	FlagRPCEndpoints             = "rpcs"
	FlagKafkaBootstrapServer     = "bootstrap-server"
	FlagDBConnectionString       = "db"
	FlagMaxRetries               = "max-retries"
	FlagNumWorkers               = "workers"
	FlagChain                    = "chain"
	FlagKafkaTopic               = "block-topic"
	FlagKafkaAPIKey              = "kafka-api-key"
	FlagKafkaAPISecret           = "kafka-api-secret"
	FlagPollIntervalInMs         = "poll-interval"
	FlagAWSAccessKey             = "aws-access-key"
	FlagAWSSecretKey             = "aws-secret-key"
	FlagClaimCheckBucket         = "claim-check-bucket"
	FlagClaimCheckThresholdInMB  = "claim-check-threshold-mb"
	FlagEnvironment              = "environment"
	FlagRebalanceInterval        = "rebalance-interval"
	FlagRPCTimeoutInSeconds      = "rpc-timeout-in-seconds"
	FlagSentryDSN                = "sentry-dsn"
	FlagCommitSHA                = "commit-sha"
	FlagSentryProfilesSampleRate = "sentry-profiles-sample-rate"
	FlagSentryTracesSampleRate   = "sentry-traces-sample-rate"
)

// SweepCmd sweeps the blockchain for data.
func SweepCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sweep",
		Short: "Sweep the blockchain for data",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			rpcEndpoints, _ := cmd.Flags().GetString(FlagRPCEndpoints)
			kafkaBootstrapServer, _ := cmd.Flags().GetString(FlagKafkaBootstrapServer)
			chain, _ := cmd.Flags().GetString(FlagChain)
			dbConnectionString, _ := cmd.Flags().GetString(FlagDBConnectionString)
			kafkaTopic, _ := cmd.Flags().GetString(FlagKafkaTopic)
			kafkaAPIKey, _ := cmd.Flags().GetString(FlagKafkaAPIKey)
			kafkaAPISecret, _ := cmd.Flags().GetString(FlagKafkaAPISecret)

			maxRetries, _ := cmd.Flags().GetUint64(FlagMaxRetries)
			numWorkers, _ := cmd.Flags().GetUint64(FlagNumWorkers)
			pollIntervalInMs, _ := cmd.Flags().GetUint64(FlagPollIntervalInMs)

			awsAccessKey, _ := cmd.Flags().GetString(FlagAWSAccessKey)
			awsSecretKey, _ := cmd.Flags().GetString(FlagAWSSecretKey)
			claimCheckBucket, _ := cmd.Flags().GetString(FlagClaimCheckBucket)
			claimCheckThresholdInMB, _ := cmd.Flags().GetUint64(FlagClaimCheckThresholdInMB)

			environment, _ := cmd.Flags().GetString(FlagEnvironment)
			rebalanceInterval, _ := cmd.Flags().GetInt64(FlagRebalanceInterval)
			rpcTimeOutInSeconds, _ := cmd.Flags().GetInt64(FlagRPCTimeoutInSeconds)

			sentryDSN, _ := cmd.Flags().GetString(FlagSentryDSN)
			commitSHA, _ := cmd.Flags().GetString(FlagCommitSHA)
			sentryProfilesSampleRate, _ := cmd.Flags().GetFloat64(FlagSentryProfilesSampleRate)
			sentryTracesSampleRate, _ := cmd.Flags().GetFloat64(FlagSentryTracesSampleRate)

			s, err := sweeper.NewSweeper(&sweeper.SweeperConfig{
				RPCEndpoints:             rpcEndpoints,
				KafkaBootstrapServer:     kafkaBootstrapServer,
				KafkaTopic:               kafkaTopic,
				KafkaAPIKey:              kafkaAPIKey,
				KafkaAPISecret:           kafkaAPISecret,
				MaxRetries:               int64(maxRetries),
				NumWorkers:               int64(numWorkers),
				Chain:                    chain,
				DBConnectionString:       dbConnectionString,
				PollSleepIntervalInMs:    int64(pollIntervalInMs),
				AWSAccessKey:             awsAccessKey,
				AWSSecretKey:             awsSecretKey,
				ClaimCheckBucket:         claimCheckBucket,
				ClaimCheckThresholdInMB:  int64(claimCheckThresholdInMB),
				Environment:              environment,
				RebalanceInterval:        rebalanceInterval,
				RPCTimeOutInSeconds:      rpcTimeOutInSeconds,
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

	threshold, err := strconv.ParseInt(os.Getenv("CLAIM_CHECK_THRESHOLD_IN_MB"), 10, 64)
	if err != nil {
		threshold = 1
	}

	pollInterval, err := strconv.ParseInt(os.Getenv("POLL_INTERVAL"), 10, 64)
	if err != nil {
		pollInterval = 1000
	}

	rebalanceInterval, err := strconv.ParseInt(os.Getenv("REBALANCE_INTERVAL"), 10, 64)
	if err != nil {
		rebalanceInterval = 0
	}

	rpcTimeOutInSeconds, err := strconv.ParseInt(os.Getenv("RPC_TIMEOUT_IN_SECONDS"), 10, 64)
	if err != nil {
		rpcTimeOutInSeconds = 30
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
	cmd.Flags().String(FlagKafkaBootstrapServer, os.Getenv("BOOTSTRAP_SERVER"), "<host>:<port> to Kafka bootstrap server")
	cmd.Flags().String(FlagKafkaTopic, os.Getenv("BLOCK_TOPIC"), "Kafka topic")
	cmd.Flags().String(FlagKafkaAPIKey, os.Getenv("KAFKA_API_KEY"), "Kafka API key")
	cmd.Flags().String(FlagKafkaAPISecret, os.Getenv("KAFKA_API_SECRET"), "Kafka API secret")
	cmd.Flags().String(FlagDBConnectionString, os.Getenv("DB_CONNECTION_STRING"), "Database connection string")
	cmd.Flags().Uint64(FlagMaxRetries, 10, "Maximum retry count")
	cmd.Flags().Uint64(FlagNumWorkers, uint64(runtime.NumCPU()), "Worker count")
	cmd.Flags().Uint64(FlagPollIntervalInMs, uint64(pollInterval), "Poll interval in milliseconds")
	cmd.Flags().String(FlagChain, os.Getenv("CHAIN"), "Chain ID to sweep")
	cmd.Flags().String(FlagAWSAccessKey, os.Getenv("AWS_ACCESS_KEY"), "AWS access key")
	cmd.Flags().String(FlagAWSSecretKey, os.Getenv("AWS_SECRET_KEY"), "AWS secret key")
	cmd.Flags().String(FlagClaimCheckBucket, os.Getenv("CLAIM_CHECK_BUCKET"), "Claim check bucket")
	cmd.Flags().Uint64(FlagClaimCheckThresholdInMB, uint64(threshold), "Claim check threshold in MB")
	cmd.Flags().String(FlagEnvironment, os.Getenv("ENVIRONMENT"), "Environment")
	cmd.Flags().Int64(FlagRebalanceInterval, rebalanceInterval, "RPC providers rebalance interval")
	cmd.Flags().Int64(FlagRPCTimeoutInSeconds, rpcTimeOutInSeconds, "RPC timeout in seconds")
	cmd.Flags().String(FlagSentryDSN, os.Getenv("SENTRY_DSN"), "Sentry DSN")
	cmd.Flags().String(FlagCommitSHA, os.Getenv("COMMIT_SHA"), "Commit SHA")
	cmd.Flags().Float64(FlagSentryProfilesSampleRate, sentryProfilesSampleRate, "Sentry profiles sample rate")
	cmd.Flags().Float64(FlagSentryTracesSampleRate, sentryTracesSampleRate, "Sentry traces sample rate")

	return cmd
}
