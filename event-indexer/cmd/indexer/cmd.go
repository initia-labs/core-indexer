package indexer_cmd

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/initia-labs/core-indexer/event-indexer/indexer"
)

const (
	FlagID                             = "id"
	FlagRPCEndpoints                   = "rpc-endpoints"
	FlagRPCTimeoutInSeconds            = "rpc-timeout-in-seconds"
	FlagKafkaBootstrapServer           = "bootstrap-server"
	FlagDBConnectionString             = "db"
	FlagChain                          = "chain"
	KafkaBlockResultsTopic             = "block-results-topic"
	FlagKafkaAPIKey                    = "kafka-api-key"
	FlagKafkaAPISecret                 = "kafka-api-secret"
	FlagBlockResultsClaimCheckBucket   = "block-results-claim-check-bucket"
	FlagClaimCheckThresholdInMB        = "claim-check-threshold-mb"
	FlagKafkaBlockResultsConsumerGroup = "block-results-consumer-group"
	FlagEnvironment                    = "environment"
	FlagSentryDSN                      = "sentry-dsn"
	FlagCommitSHA                      = "commit-sha"
	FlagSentryProfilesSampleRate       = "sentry-profiles-sample-rate"
	FlagSentryTracesSampleRate         = "sentry-traces-sample-rate"
	FlagMaxWorkers                     = "max-workers"
)

// RunCmd consumes messages from Kafka and indexes them into the database.
func RunCmd() *cobra.Command {
	RunCmd := &cobra.Command{
		Use:   "run",
		Short: "Consumes messages from Kafka and indexes them into the database.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			rpcEndpoints, _ := cmd.Flags().GetString(FlagRPCEndpoints)
			rpcTimeoutInSeconds, _ := cmd.Flags().GetInt64(FlagRPCTimeoutInSeconds)
			kafkaBootstrapServer, _ := cmd.Flags().GetString(FlagKafkaBootstrapServer)
			chain, _ := cmd.Flags().GetString(FlagChain)
			dbConnectionString, _ := cmd.Flags().GetString(FlagDBConnectionString)
			kafkaBlockResultsTopic, _ := cmd.Flags().GetString(KafkaBlockResultsTopic)
			kafkaAPIKey, _ := cmd.Flags().GetString(FlagKafkaAPIKey)
			kafkaAPISecret, _ := cmd.Flags().GetString(FlagKafkaAPISecret)
			kafkaBlockResultsConsumerGroup, _ := cmd.Flags().GetString(FlagKafkaBlockResultsConsumerGroup)

			blockResultsClaimCheckBucket, _ := cmd.Flags().GetString(FlagBlockResultsClaimCheckBucket)
			claimCheckThresholdInMB, _ := cmd.Flags().GetUint64(FlagClaimCheckThresholdInMB)

			workerID, _ := cmd.Flags().GetString(FlagID)
			maxWorkers, _ := cmd.Flags().GetInt(FlagMaxWorkers)

			environment, _ := cmd.Flags().GetString(FlagEnvironment)
			sentryDSN, _ := cmd.Flags().GetString(FlagSentryDSN)
			commitSHA, _ := cmd.Flags().GetString(FlagCommitSHA)
			sentryProfilesSampleRate, _ := cmd.Flags().GetFloat64(FlagSentryProfilesSampleRate)
			sentryTracesSampleRate, _ := cmd.Flags().GetFloat64(FlagSentryTracesSampleRate)

			f, err := indexer.New(&indexer.Config{
				RPCEndpoints:                   rpcEndpoints,
				RPCTimeoutInSeconds:            rpcTimeoutInSeconds,
				ID:                             workerID,
				Chain:                          chain,
				DBConnectionString:             dbConnectionString,
				KafkaBootstrapServer:           kafkaBootstrapServer,
				KafkaBlockResultsTopic:         kafkaBlockResultsTopic,
				KafkaAPIKey:                    kafkaAPIKey,
				KafkaAPISecret:                 kafkaAPISecret,
				KafkaBlockResultsConsumerGroup: kafkaBlockResultsConsumerGroup,
				ClaimCheckThresholdInMB:        int64(claimCheckThresholdInMB),
				BlockResultsClaimCheckBucket:   blockResultsClaimCheckBucket,
				Environment:                    environment,
				SentryDSN:                      sentryDSN,
				CommitSHA:                      commitSHA,
				SentryProfilesSampleRate:       sentryProfilesSampleRate,
				SentryTracesSampleRate:         sentryTracesSampleRate,
				MaxWorkers:                     maxWorkers,
			})
			if err != nil {
				return err
			}

			f.Run()

			return nil
		},
	}

	rpcTimeOutInSeconds, err := strconv.ParseInt(os.Getenv("RPC_TIMEOUT_IN_SECONDS"), 10, 64)
	if err != nil {
		rpcTimeOutInSeconds = 30
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

	maxWorkers, err := strconv.Atoi(os.Getenv("MAX_WORKERS"))
	if err != nil {
		maxWorkers = 10
	}

	RunCmd.Flags().String(FlagRPCEndpoints, os.Getenv("RPC_ENDPOINTS"), "")
	RunCmd.Flags().String(FlagKafkaBootstrapServer, os.Getenv("BOOTSTRAP_SERVER"), "<host>:<port> to Kafka bootstrap server")
	RunCmd.Flags().Int64(FlagRPCTimeoutInSeconds, rpcTimeOutInSeconds, "RPC timeout in seconds")
	RunCmd.Flags().String(KafkaBlockResultsTopic, os.Getenv("BLOCK_RESULTS_TOPIC"), "Kafka topic to consume block_results message")
	RunCmd.Flags().String(FlagKafkaBlockResultsConsumerGroup, os.Getenv("BLOCK_RESULTS_CONSUMER_GROUP"), "Kafka consumer group for block_results topic")
	RunCmd.Flags().String(FlagKafkaAPIKey, os.Getenv("KAFKA_API_KEY"), "Kafka API key")
	RunCmd.Flags().String(FlagKafkaAPISecret, os.Getenv("KAFKA_API_SECRET"), "Kafka API secret")
	RunCmd.Flags().String(FlagDBConnectionString, os.Getenv("DB_CONNECTION_STRING"), "Database connection string")
	RunCmd.Flags().String(FlagChain, os.Getenv("CHAIN"), "Chain ID to sweep")
	RunCmd.Flags().String(FlagBlockResultsClaimCheckBucket, os.Getenv("BLOCK_RESULTS_CLAIM_CHECK_BUCKET"), "Block results claim check bucket")
	RunCmd.Flags().Uint64(FlagClaimCheckThresholdInMB, uint64(threshold), "Claim check threshold in MB")
	RunCmd.Flags().String(FlagID, os.Getenv("ID"), "Worker ID")
	RunCmd.Flags().String(FlagEnvironment, os.Getenv("ENVIRONMENT"), "Environment")
	RunCmd.Flags().String(FlagSentryDSN, os.Getenv("SENTRY_DSN"), "Sentry DSN")
	RunCmd.Flags().String(FlagCommitSHA, os.Getenv("COMMIT_SHA"), "Commit SHA")
	RunCmd.Flags().Float64(FlagSentryProfilesSampleRate, sentryProfilesSampleRate, "Sentry profiles sample rate")
	RunCmd.Flags().Float64(FlagSentryTracesSampleRate, sentryTracesSampleRate, "Sentry traces sample rate")
	RunCmd.Flags().Int(FlagMaxWorkers, maxWorkers, "Max workers")

	return RunCmd
}
