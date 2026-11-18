package indexer_cmd

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/initia-labs/core-indexer/generic-indexer/indexer"
)

// List of CLI flags
const (
	FlagID                            = "id"
	FlagRPCEndpoints                  = "rpcs"
	FlagPubSubProjectID               = "pubsub-project-id"
	FlagPubSubEndpoint                = "pubsub-endpoint"
	FlagDBConnectionString            = "db"
	FlagNumWorkers                    = "workers"
	FlagChain                         = "chain"
	TxResponseTopic                   = "tx-topic"
	FlagAWSAccessKey                  = "aws-access-key"
	FlagAWSSecretKey                  = "aws-secret-key"
	FlagBlockClaimCheckBucket         = "block-claim-check-bucket"
	FlagClaimCheckThresholdInMB       = "claim-check-threshold-mb"
	FlagLCDTxResponseClaimCheckBucket = "lcd-tx-response-claim-check-bucket"
	FlagBlockResultsSubscription      = "block-results-subscription"
	FlagEnvironment                   = "environment"
	FlagRebalanceInterval             = "rebalance-interval"
	FlagRPCTimeoutInSeconds           = "rpc-timeout-in-seconds"
	FlagSentryDSN                     = "sentry-dsn"
	FlagCommitSHA                     = "commit-sha"
	FlagSentryProfilesSampleRate      = "sentry-profiles-sample-rate"
	FlagSentryTracesSampleRate        = "sentry-traces-sample-rate"
	FlagBlockResultsClaimCheckBucket  = "block-results-claim-check-bucket"
)

// IndexerCmd consumes from Pub/Sub and indexes into database.
func IndexerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Consumes from Pub/Sub and indexes the blockchain for data",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			rpcEndpoints, _ := cmd.Flags().GetString(FlagRPCEndpoints)
			pubSubProjectID, _ := cmd.Flags().GetString(FlagPubSubProjectID)
			pubSubEndpoint, _ := cmd.Flags().GetString(FlagPubSubEndpoint)
			chain, _ := cmd.Flags().GetString(FlagChain)
			dbConnectionString, _ := cmd.Flags().GetString(FlagDBConnectionString)
			txResponseTopic, _ := cmd.Flags().GetString(TxResponseTopic)
			blockResultsSubscription, _ := cmd.Flags().GetString(FlagBlockResultsSubscription)

			numWorkers, _ := cmd.Flags().GetUint64(FlagNumWorkers)

			awsAccessKey, _ := cmd.Flags().GetString(FlagAWSAccessKey)
			awsSecretKey, _ := cmd.Flags().GetString(FlagAWSSecretKey)
			blockClaimCheckBucket, _ := cmd.Flags().GetString(FlagBlockClaimCheckBucket)
			claimCheckThresholdInMB, _ := cmd.Flags().GetUint64(FlagClaimCheckThresholdInMB)
			lcdTxResponseClaimCheckBucket, _ := cmd.Flags().GetString(FlagLCDTxResponseClaimCheckBucket)

			workerID, _ := cmd.Flags().GetString(FlagID)

			environment, _ := cmd.Flags().GetString(FlagEnvironment)
			rebalanceInterval, _ := cmd.Flags().GetInt64(FlagRebalanceInterval)
			rpcTimeOutInSeconds, _ := cmd.Flags().GetInt64(FlagRPCTimeoutInSeconds)

			sentryDSN, _ := cmd.Flags().GetString(FlagSentryDSN)
			commitSHA, _ := cmd.Flags().GetString(FlagCommitSHA)
			sentryProfilesSampleRate, _ := cmd.Flags().GetFloat64(FlagSentryProfilesSampleRate)
			sentryTracesSampleRate, _ := cmd.Flags().GetFloat64(FlagSentryTracesSampleRate)
			blockResultsClaimCheckBucket, _ := cmd.Flags().GetString(FlagBlockResultsClaimCheckBucket)

			f, err := indexer.New(&indexer.IndexerConfig{
				ID:                            workerID,
				RPCEndpoints:                  rpcEndpoints,
				PubSubProjectID:               pubSubProjectID,
				PubSubEndpoint:                pubSubEndpoint,
				TxResponseTopic:               txResponseTopic,
				BlockResultsSubscription:      blockResultsSubscription,
				NumWorkers:                    int64(numWorkers),
				Chain:                         chain,
				DBConnectionString:            dbConnectionString,
				AWSAccessKey:                  awsAccessKey,
				AWSSecretKey:                  awsSecretKey,
				BlockClaimCheckBucket:         blockClaimCheckBucket,
				ClaimCheckThresholdInMB:       int64(claimCheckThresholdInMB),
				LCDTxResponseClaimCheckBucket: lcdTxResponseClaimCheckBucket,
				Environment:                   environment,
				RebalanceInterval:             rebalanceInterval,
				RPCTimeOutInSeconds:           rpcTimeOutInSeconds,
				SentryDSN:                     sentryDSN,
				CommitSHA:                     commitSHA,
				SentryProfilesSampleRate:      sentryProfilesSampleRate,
				SentryTracesSampleRate:        sentryTracesSampleRate,
				BlockResultsClaimCheckBucket:  blockResultsClaimCheckBucket,
			})

			if err != nil {
				return err
			}

			f.Run()

			return nil
		},
	}

	threshold, err := strconv.ParseInt(os.Getenv("CLAIM_CHECK_THRESHOLD_IN_MB"), 10, 64)
	if err != nil {
		threshold = 1
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
	cmd.Flags().String(FlagPubSubProjectID, os.Getenv("PUBSUB_PROJECT_ID"), "Google Cloud project for Pub/Sub")
	cmd.Flags().String(FlagPubSubEndpoint, os.Getenv("PUBSUB_ENDPOINT"), "Pub/Sub regional endpoint (optional)")
	cmd.Flags().String(TxResponseTopic, os.Getenv("TX_TOPIC"), "Pub/Sub topic about TxResponses to produce")
	cmd.Flags().String(FlagBlockResultsSubscription, os.Getenv("BLOCK_RESULTS_SUBSCRIPTION"), "Pub/Sub subscription for block results")
	cmd.Flags().String(FlagDBConnectionString, os.Getenv("DB_CONNECTION_STRING"), "Database connection string")
	cmd.Flags().Uint64(FlagNumWorkers, 10, "Worker count")
	cmd.Flags().String(FlagChain, os.Getenv("CHAIN"), "Chain ID to sweep")
	cmd.Flags().String(FlagAWSAccessKey, os.Getenv("AWS_ACCESS_KEY"), "AWS access key")
	cmd.Flags().String(FlagAWSSecretKey, os.Getenv("AWS_SECRET_KEY"), "AWS secret key")
	cmd.Flags().String(FlagBlockClaimCheckBucket, os.Getenv("BLOCK_CLAIM_CHECK_BUCKET"), "Block claim check bucket")
	cmd.Flags().Uint64(FlagClaimCheckThresholdInMB, uint64(threshold), "Claim check threshold in MB")
	cmd.Flags().String(FlagLCDTxResponseClaimCheckBucket, os.Getenv("TX_CLAIM_CHECK_BUCKET"), "LCD TxResponse claim check bucket")
	cmd.Flags().String(FlagID, os.Getenv("ID"), "Worker ID")
	cmd.Flags().String(FlagEnvironment, os.Getenv("ENVIRONMENT"), "Environment")
	cmd.Flags().Int64(FlagRebalanceInterval, rebalanceInterval, "RPC providers rebalance interval")
	cmd.Flags().Int64(FlagRPCTimeoutInSeconds, rpcTimeOutInSeconds, "RPC timeout in seconds")
	cmd.Flags().String(FlagSentryDSN, os.Getenv("SENTRY_DSN"), "Sentry DSN")
	cmd.Flags().String(FlagCommitSHA, os.Getenv("COMMIT_SHA"), "Commit SHA")
	cmd.Flags().Float64(FlagSentryProfilesSampleRate, sentryProfilesSampleRate, "Sentry profiles sample rate")
	cmd.Flags().Float64(FlagSentryTracesSampleRate, sentryTracesSampleRate, "Sentry traces sample rate")
	cmd.Flags().String(FlagBlockResultsClaimCheckBucket, os.Getenv("BLOCK_RESULTS_CLAIM_CHECK_BUCKET"), "Block results claim check bucket")

	return cmd
}
