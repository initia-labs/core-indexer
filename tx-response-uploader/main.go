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
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
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

func getHashAndHeightFromHeaders(headers []kafka.Header) (hash string, height string) {
	for _, header := range headers {
		switch header.Key {
		case HeaderHashKey:
			hash = string(header.Value)
		case HeaderHeightKey:
			height = string(header.Value)
		}
	}

	return
}

func (u *TxResponseUploader) process(message *kafka.Message) error {
	hash, height := getHashAndHeightFromHeaders(message.Headers)

	if hash == "" || height == "" {
		logger.Error().Msgf("Error getting hash and height from headers")
		return fmt.Errorf("either no hash or height from headers")
	}

	var data []byte
	if strings.HasPrefix(string(message.Key), NewLcdTxResponseClaimCheckKafkaMessageKey) {
		logger.Info().Msgf("Processing claim check message: %s", string(message.Key))
		var claimCheckMessage ClaimCheckMessage
		err := json.Unmarshal(message.Value, &claimCheckMessage)
		if err != nil {
			logger.Error().Msgf("Error unmarshalling claim check message: %v", err)
			return err
		}

		retryCount := 0
		// read from storage until succeeds
		for {
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
		data = message.Value
	}

	return u.storageClient.UploadFile(u.config.LCDTXResponseBucket, fmt.Sprintf("%s/%s", strings.ToUpper(hash), height), data)
}

func (u *TxResponseUploader) close() {
	u.producer.Flush(10000)
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

	consumer, err := mq.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":    config.BootstrapServer,
		"group.id":             config.KafkaConsumerGroup,
		"client.id":            config.KafkaConsumerGroup + "-" + config.ID,
		"auto.offset.reset":    "earliest",
		"enable.auto.commit":   false,
		"security.protocol":    "SASL_SSL",
		"sasl.mechanisms":      "PLAIN",
		"sasl.username":        config.KafkaAPIKey,
		"sasl.password":        config.KafkaAPISecret,
		"max.poll.interval.ms": 600000,
	})

	if err != nil {
		logger.Fatal().Msgf("Failed to create consumer: %s\n", err)
	}

	producer, err := mq.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": config.BootstrapServer,
		"client.id":         config.KafkaConsumerGroup + "-" + config.ID,
		"acks":              "all",
		"linger.ms":         200,
		"security.protocol": "SASL_SSL",
		"sasl.mechanisms":   "PLAIN",
		"sasl.username":     config.KafkaAPIKey,
		"sasl.password":     config.KafkaAPISecret,
		"message.max.bytes": 7340032,
	})
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

	err := u.consumer.SubscribeTopics([]string{u.config.Topic}, nil)
	if err != nil {
		logger.Fatal().Msgf("Failed to subscribe to topic: %s\n", err)
	}

	logger.Info().Msgf("Subscribed to topic: %s\n", u.config.Topic)

	for {
		select {
		case <-signalContext.Done():
			return
		default:
			message, err := u.consumer.ReadMessage(10 * time.Second)
			if err != nil {
				if err.(kafka.Error).IsTimeout() {
					continue
				}

				logger.Error().Msgf("Error reading message: %v", err)
				continue
			}

			sentry.ConfigureScope(func(scope *sentry.Scope) {
				scope.SetTag("id", u.config.ID)
				scope.SetTag("partition", fmt.Sprint(message.TopicPartition.Partition))
				scope.SetTag("offset", message.TopicPartition.Offset.String())
			})

			transaction, ctx := sentry_integration.StartSentryTransaction(context.Background(), "Upload", "Read Tx Response from Kafka and upload to GCS")
			err = u.process(ctx, message)
			if err != nil {
				sentry_integration.CaptureCurrentHubException(err, sentry.LevelError)
				logger.Error().Msgf("Error processing message: %v", err)
				u.producer.ProduceToDLQ(u.config.Chain, "tx-responses", message, err, logger)
			}

			_, err = u.consumer.CommitMessage(message)
			if err != nil {
				sentry_integration.CaptureCurrentHubException(err, sentry.LevelError)
				logger.Error().Msgf("Non-retryable Error committing message: %v", err)
			}
			transaction.Finish()
		}
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
