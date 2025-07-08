package flusher_cmd

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/alleslabs/initia-mono/generic-indexer/flusher"
)

// List of CLI flags
const (
	FlagID                                = "id"
	FlagRPCEndpoints                      = "rpcs"
	FlagKafkaBootstrapServer              = "bootstrap-server"
	FlagDBConnectionString                = "db"
	FlagNumWorkers                        = "workers"
	FlagChain                             = "chain"
	KafkaBlockTopic                       = "block-topic"
	KafkaTxResponseTopic                  = "tx-topic"
	FlagKafkaAPIKey                       = "kafka-api-key"
	FlagKafkaAPISecret                    = "kafka-api-secret"
	FlagAWSAccessKey                      = "aws-access-key"
	FlagAWSSecretKey                      = "aws-secret-key"
	FlagBlockClaimCheckBucket             = "block-claim-check-bucket"
	FlagClaimCheckThresholdInMB           = "claim-check-threshold-mb"
	FlagLCDTxResponseClaimCheckBucket     = "lcd-tx-response-claim-check-bucket"
	FlagKafkaBlockConsumerGroup           = "block-consumer-group"
	FlagDisableLCDTxResponse              = "disable-lcd-tx-response"
	FlagDisableIndexingAccountTransaction = "disable-indexing-account-transaction"
	FlagEnvironment                       = "environment"
	FlagRebalanceInterval                 = "rebalance-interval"
	FlagRPCTimeoutInSeconds               = "rpc-timeout-in-seconds"
	FlagSentryDSN                         = "sentry-dsn"
	FlagCommitSHA                         = "commit-sha"
	FlagSentryProfilesSampleRate          = "sentry-profiles-sample-rate"
	FlagSentryTracesSampleRate            = "sentry-traces-sample-rate"
)

// FlushCmd consumes from Kafka and flushes into database.
func FlushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "flush",
		Short: "Consumes from Kafka and flushes the blockchain for data",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			rpcEndpoints, _ := cmd.Flags().GetString(FlagRPCEndpoints)
			kafkaBootstrapServer, _ := cmd.Flags().GetString(FlagKafkaBootstrapServer)
			chain, _ := cmd.Flags().GetString(FlagChain)
			dbConnectionString, _ := cmd.Flags().GetString(FlagDBConnectionString)
			kafkaBlockTopic, _ := cmd.Flags().GetString(KafkaBlockTopic)
			kafkaTxResponseTopic, _ := cmd.Flags().GetString(KafkaTxResponseTopic)
			kafkaAPIKey, _ := cmd.Flags().GetString(FlagKafkaAPIKey)
			kafkaAPISecret, _ := cmd.Flags().GetString(FlagKafkaAPISecret)
			kafkaBlockConsumerGroup, _ := cmd.Flags().GetString(FlagKafkaBlockConsumerGroup)

			numWorkers, _ := cmd.Flags().GetUint64(FlagNumWorkers)

			awsAccessKey, _ := cmd.Flags().GetString(FlagAWSAccessKey)
			awsSecretKey, _ := cmd.Flags().GetString(FlagAWSSecretKey)
			blockClaimCheckBucket, _ := cmd.Flags().GetString(FlagBlockClaimCheckBucket)
			claimCheckThresholdInMB, _ := cmd.Flags().GetUint64(FlagClaimCheckThresholdInMB)
			lcdTxResponseClaimCheckBucket, _ := cmd.Flags().GetString(FlagLCDTxResponseClaimCheckBucket)

			workerID, _ := cmd.Flags().GetString(FlagID)

			disableLCDTxResponse, _ := cmd.Flags().GetBool(FlagDisableLCDTxResponse)
			environment, _ := cmd.Flags().GetString(FlagEnvironment)
			rebalanceInterval, _ := cmd.Flags().GetInt64(FlagRebalanceInterval)
			rpcTimeOutInSeconds, _ := cmd.Flags().GetInt64(FlagRPCTimeoutInSeconds)

			sentryDSN, _ := cmd.Flags().GetString(FlagSentryDSN)
			commitSHA, _ := cmd.Flags().GetString(FlagCommitSHA)
			sentryProfilesSampleRate, _ := cmd.Flags().GetFloat64(FlagSentryProfilesSampleRate)
			sentryTracesSampleRate, _ := cmd.Flags().GetFloat64(FlagSentryTracesSampleRate)

			f, err := flusher.NewFlusher(&flusher.FlusherConfig{
				ID:                            workerID,
				RPCEndpoints:                  rpcEndpoints,
				KafkaBootstrapServer:          kafkaBootstrapServer,
				KafkaBlockTopic:               kafkaBlockTopic,
				KafkaTxResponseTopic:          kafkaTxResponseTopic,
				KafkaAPIKey:                   kafkaAPIKey,
				KafkaAPISecret:                kafkaAPISecret,
				KafkaBlockConsumerGroup:       kafkaBlockConsumerGroup,
				NumWorkers:                    int64(numWorkers),
				Chain:                         chain,
				DBConnectionString:            dbConnectionString,
				AWSAccessKey:                  awsAccessKey,
				AWSSecretKey:                  awsSecretKey,
				BlockClaimCheckBucket:         blockClaimCheckBucket,
				ClaimCheckThresholdInMB:       int64(claimCheckThresholdInMB),
				LCDTxResponseClaimCheckBucket: lcdTxResponseClaimCheckBucket,
				DisableLCDTXResponse:          disableLCDTxResponse,
				Environment:                   environment,
				RebalanceInterval:             rebalanceInterval,
				RPCTimeOutInSeconds:           rpcTimeOutInSeconds,
				SentryDSN:                     sentryDSN,
				CommitSHA:                     commitSHA,
				SentryProfilesSampleRate:      sentryProfilesSampleRate,
				SentryTracesSampleRate:        sentryTracesSampleRate,
			})

			if err != nil {
				return err
			}

			f.Flush()

			return nil
		},
	}

	threshold, err := strconv.ParseInt(os.Getenv("CLAIM_CHECK_THRESHOLD_IN_MB"), 10, 64)
	if err != nil {
		threshold = 1
	}

	disableTxResponse, err := strconv.ParseBool(os.Getenv("DISABLE_LCD_TX_RESPONSE"))
	if err != nil {
		disableTxResponse = false
	}

	disableIndexingAccountTransaction, err := strconv.ParseBool(os.Getenv("DISABLE_INDEXING_ACCOUNT_TRANSACTION"))
	if err != nil {
		disableIndexingAccountTransaction = false
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
	cmd.Flags().String(KafkaBlockTopic, os.Getenv("BLOCK_TOPIC"), "Kafka topic about Blocks to consume")
	cmd.Flags().String(KafkaTxResponseTopic, os.Getenv("TX_TOPIC"), "Kafka topic about TxResponses to produce")
	cmd.Flags().String(FlagKafkaBlockConsumerGroup, os.Getenv("BLOCK_CONSUMER_GROUP"), "Kafka consumer group for block topic")
	cmd.Flags().String(FlagKafkaAPIKey, os.Getenv("KAFKA_API_KEY"), "Kafka API key")
	cmd.Flags().String(FlagKafkaAPISecret, os.Getenv("KAFKA_API_SECRET"), "Kafka API secret")
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
	cmd.Flags().Bool(FlagDisableLCDTxResponse, disableTxResponse, "Disable LCD TxResponse")
	cmd.Flags().Bool(FlagDisableIndexingAccountTransaction, disableIndexingAccountTransaction, "Disable indexing account transaction")
	cmd.Flags().Int64(FlagRebalanceInterval, rebalanceInterval, "RPC providers rebalance interval")
	cmd.Flags().Int64(FlagRPCTimeoutInSeconds, rpcTimeOutInSeconds, "RPC timeout in seconds")
	cmd.Flags().String(FlagSentryDSN, os.Getenv("SENTRY_DSN"), "Sentry DSN")
	cmd.Flags().String(FlagCommitSHA, os.Getenv("COMMIT_SHA"), "Commit SHA")
	cmd.Flags().Float64(FlagSentryProfilesSampleRate, sentryProfilesSampleRate, "Sentry profiles sample rate")
	cmd.Flags().Float64(FlagSentryTracesSampleRate, sentryTracesSampleRate, "Sentry traces sample rate")

	return cmd
}
