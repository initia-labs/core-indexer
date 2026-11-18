package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/certifi/gocertifi"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
	"github.com/initia-labs/core-indexer/pkg/storage"
)

var logger *zerolog.Logger

type TxResponseUploader struct {
	producer *mq.Producer
	consumer *mq.Consumer

	storageClient storage.Client

	config *Config
}

func (u *TxResponseUploader) process(ctx context.Context, message *mq.Message) error {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "process", "Process Pub/Sub Message")
	defer span.Finish()

	hash, height := message.Attributes[HeaderHashKey], message.Attributes[HeaderHeightKey]

	if hash == "" || height == "" {
		logger.Error().Msgf("Error getting hash and height from headers")
		return fmt.Errorf("either no hash or height from headers")
	}

	var data []byte
	if strings.HasPrefix(message.Attributes["key"], NewLcdTxResponseClaimCheckMessageKey) {
		var claimCheckMessage ClaimCheckMessage
		err := json.Unmarshal(message.Data, &claimCheckMessage)
		if err != nil {
			logger.Error().Msgf("Error unmarshalling claim check message: %v", err)
			return err
		}

		retryCount := 0
		// read from storage until succeeds
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			retryCount++
			data, err = u.storageClient.ReadFile(u.config.ClaimCheckBucket, claimCheckMessage.ObjectPath)
			if err == nil {
				break
			}

			if retryCount >= 5 {
				sentry_integration.CaptureCurrentHubException(err, sentry.LevelError)
			} else {
				sentry_integration.CaptureCurrentHubException(err, sentry.LevelWarning)
			}

			logger.Error().Msgf("Error reading file from storage: %v", err)
		}
	} else {
		data = message.Data
	}

	return u.storageClient.UploadFile(u.config.LCDTXResponseBucket, fmt.Sprintf("%s/%s", strings.ToUpper(hash), height), data)
}

func (u *TxResponseUploader) close() {
	u.producer.Close()
	u.consumer.Close()
}

func NewTxResponseUploader(config *Config) *TxResponseUploader {
	sentryClientOptions := sentry.ClientOptions{
		Dsn:                config.SentryDSN,
		ServerName:         config.Chain + "-" + ServiceName,
		EnableTracing:      true,
		ProfilesSampleRate: config.SentryProfileSampleRate,
		TracesSampleRate:   config.SentryTraceSampleRate,
		Environment:        config.Environment,
		Release:            config.CommitSHA,
		Tags: map[string]string{
			"chain":       config.Chain,
			"environment": config.Environment,
			"component":   ServiceName,
			"commit_sha":  config.CommitSHA,
		},
	}

	rootCAs, err := gocertifi.CACerts()
	if err != nil {
		logger.Fatal().Msgf("Sentry: Error getting root CAs: %v\n", err)
	} else {
		sentryClientOptions.CaCerts = rootCAs
	}

	err = sentry.Init(sentryClientOptions)

	if err != nil {
		logger.Fatal().Msgf("Sentry: Error initializing sentry: %v\n", err)
	}

	storageClient, err := storage.NewGCSClient()
	if err != nil {
		logger.Fatal().Msgf("Error connecting to GCS: %v", err)
	}

	consumer, err := mq.NewConsumer(context.Background(), config.PubSubProjectID, config.TxResponseSubscription, config.PubSubEndpoint)

	if err != nil {
		logger.Fatal().Msgf("Failed to create consumer: %s\n", err)
	}

	producer, err := mq.NewProducer(context.Background(), config.PubSubProjectID, config.PubSubEndpoint)
	if err != nil {
		logger.Fatal().Msgf("Failed to create producer: %s\n", err)
	}

	return &TxResponseUploader{
		producer:      producer,
		consumer:      consumer,
		storageClient: storageClient,
		config:        config,
	}
}

func (u *TxResponseUploader) StartUploading(signalContext context.Context) {
	logger.Info().Msgf("Starting uploading to GCS")
	successLogger := logger.Sample(&zerolog.BurstSampler{Burst: 1, Period: time.Minute})

	err := u.consumer.Receive(signalContext, func(ctx context.Context, message *mq.Message) error {
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("id", u.config.ID)
			scope.SetTag("message_id", message.ID)
		})
		transaction, ctx := sentry_integration.StartSentryTransaction(ctx, "Upload", "Read tx response from Pub/Sub and upload to GCS")
		defer transaction.Finish()
		if err := u.process(ctx, message); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			sentry_integration.CaptureCurrentHubException(err, sentry.LevelError)
			logger.Error().Err(err).Msg("Error processing message")
			return u.producer.ProduceToDLQ(ctx, u.config.Chain, "tx-responses", message, err)
		}
		successLogger.Info().Str("hash", message.Attributes[HeaderHashKey]).Str("height", message.Attributes[HeaderHeightKey]).Msg("Uploaded tx response to GCS")
		return nil
	})
	if err != nil && signalContext.Err() == nil {
		logger.Fatal().Err(err).Msg("Pub/Sub consumer stopped")
	}
}

func main() {
	config := getConfig()
	logger = zerolog.Ctx(
		log.With().
			Str("component", ServiceName).
			Str("chain", config.Chain).
			Str("id", config.ID).
			Str("environment", config.Environment).
			Str("commit_sha", config.CommitSHA).
			Logger().
			WithContext(context.Background()),
	)

	// graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	uploader := NewTxResponseUploader(config)

	uploader.StartUploading(ctx)

	logger.Info().Msgf("Shutting down ...")
	uploader.close()
}
