package validator_cmd

// TODO: bring back validator cron
// import (
// 	"os"
// 	"strconv"

// 	"github.com/spf13/cobra"

// 	"github.com/initia-labs/core-indexer/generic-indexer/validatorcron"
// )

// // List of CLI flags
// const (
// 	FlagRPCEndpoints                  = "rpcs"
// 	FlagDBConnectionString            = "db"
// 	FlagChain                         = "chain"
// 	FlagValidatorUpdateInterval       = "validator-update-interval"
// 	FlagValidatorUptimeUpdateInterval = "validator-uptime-update-interval"
// 	FlagEnvironment                   = "environment"
// 	FlagKeepLatestCommitSignatures    = "keep-latest-commit-signatures"
// 	FlagRPCTimeoutInSeconds           = "rpc-timeout-in-seconds"
// 	FlagSentryDSN                     = "sentry-dsn"
// 	FlagCommitSHA                     = "commit-sha"
// 	FlagSentryProfilesSampleRate      = "sentry-profiles-sample-rate"
// 	FlagSentryTracesSampleRate        = "sentry-traces-sample-rate"
// )

// // FlushCmd consumes from Kafka and flushes into database.
// func FlushCmd() *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "validatorcron",
// 		Short: "Consumes from Kafka and flushes the validators for data",
// 		Args:  cobra.ExactArgs(0),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			rpcEndpoints, _ := cmd.Flags().GetString(FlagRPCEndpoints)
// 			chain, _ := cmd.Flags().GetString(FlagChain)
// 			dbConnectionString, err := cmd.Flags().GetString(FlagDBConnectionString)
// 			if err != nil {
// 				panic(err)
// 			}

// 			validatorUpdateInterval, _ := cmd.Flags().GetInt64(FlagValidatorUpdateInterval)
// 			validatorUptimeUpdateInterval, _ := cmd.Flags().GetInt64(FlagValidatorUptimeUpdateInterval)
// 			environment, _ := cmd.Flags().GetString(FlagEnvironment)
// 			keepLatestCommitSignatures, _ := cmd.Flags().GetInt64(FlagKeepLatestCommitSignatures)
// 			rpcTimeOutInSeconds, _ := cmd.Flags().GetInt64(FlagRPCTimeoutInSeconds)

// 			sentryDSN, _ := cmd.Flags().GetString(FlagSentryDSN)
// 			commitSHA, _ := cmd.Flags().GetString(FlagCommitSHA)
// 			sentryProfilesSampleRate, _ := cmd.Flags().GetFloat64(FlagSentryProfilesSampleRate)
// 			sentryTracesSampleRate, _ := cmd.Flags().GetFloat64(FlagSentryTracesSampleRate)

// 			f, err := validatorcron.NewValidatorCronFlusher(&validatorcron.ValidatorCronConfig{
// 				RPCEndpoints:                           rpcEndpoints,
// 				Chain:                                  chain,
// 				DBConnectionString:                     dbConnectionString,
// 				ValidatorUpdateIntervalInSeconds:       int64(validatorUpdateInterval),
// 				ValidatorUptimeUpdateIntervalInSeconds: int64(validatorUptimeUpdateInterval),
// 				Environment:                            environment,
// 				KeepLatestCommitSignatures:             keepLatestCommitSignatures,
// 				RPCTimeOutInSeconds:                    rpcTimeOutInSeconds,
// 				SentryDSN:                              sentryDSN,
// 				CommitSHA:                              commitSHA,
// 				SentryProfilesSampleRate:               sentryProfilesSampleRate,
// 				SentryTracesSampleRate:                 sentryTracesSampleRate,
// 			})

// 			if err != nil {
// 				return err
// 			}

// 			f.Run()

// 			return nil
// 		},
// 	}

// 	validatorUpdateInterval, err := strconv.Atoi(os.Getenv("VALIDATOR_UPDATE_INTERVAL"))
// 	if err != nil {
// 		validatorUpdateInterval = 60
// 	}

// 	validatorUptimeUpdateInterval, err := strconv.Atoi(os.Getenv("VALIDATOR_UPTIME_UPDATE_INTERVAL"))
// 	if err != nil {
// 		validatorUptimeUpdateInterval = 60
// 	}

// 	keepLatestCommitSignatures, err := strconv.Atoi(os.Getenv("KEEP_LATEST_COMMIT_SIGNATURES"))
// 	if err != nil {
// 		keepLatestCommitSignatures = 11000
// 	}

// 	rpcTimeOutInSeconds, err := strconv.ParseInt(os.Getenv("RPC_TIMEOUT_IN_SECONDS"), 10, 64)
// 	if err != nil {
// 		rpcTimeOutInSeconds = 30
// 	}

// 	sentryProfilesSampleRate, err := strconv.ParseFloat(os.Getenv("SENTRY_PROFILES_SAMPLE_RATE"), 64)
// 	if err != nil {
// 		sentryProfilesSampleRate = 0.05
// 	}

// 	sentryTracesSampleRate, err := strconv.ParseFloat(os.Getenv("SENTRY_TRACES_SAMPLE_RATE"), 64)
// 	if err != nil {
// 		sentryTracesSampleRate = 0.05
// 	}

// 	cmd.Flags().String(FlagRPCEndpoints, os.Getenv("RPC_ENDPOINTS"), "")
// 	cmd.Flags().String(FlagDBConnectionString, os.Getenv("DB_CONNECTION_STRING"), "Database connection string")
// 	cmd.Flags().String(FlagChain, os.Getenv("CHAIN"), "Chain ID to run validator crons")
// 	cmd.Flags().Int64(FlagValidatorUpdateInterval, int64(validatorUpdateInterval), "Interval to update validators")
// 	cmd.Flags().Int64(FlagValidatorUptimeUpdateInterval, int64(validatorUptimeUpdateInterval), "Interval to update validators")
// 	cmd.Flags().String(FlagEnvironment, os.Getenv("ENVIRONMENT"), "Environment")
// 	cmd.Flags().Int64(FlagKeepLatestCommitSignatures, int64(keepLatestCommitSignatures), "Keep latest commit signatures")
// 	cmd.Flags().Int64(FlagRPCTimeoutInSeconds, rpcTimeOutInSeconds, "RPC timeout in seconds")
// 	cmd.Flags().String(FlagSentryDSN, os.Getenv("SENTRY_DSN"), "Sentry DSN")
// 	cmd.Flags().String(FlagCommitSHA, os.Getenv("COMMIT_SHA"), "Commit SHA")
// 	cmd.Flags().Float64(FlagSentryProfilesSampleRate, sentryProfilesSampleRate, "Sentry profiles sample rate")
// 	cmd.Flags().Float64(FlagSentryTracesSampleRate, sentryTracesSampleRate, "Sentry traces sample rate")

// 	return cmd
// }
